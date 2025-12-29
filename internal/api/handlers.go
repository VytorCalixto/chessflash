package api

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vytor/chessflash/internal/analysis"
	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/flashcard"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/worker"
)

type Server struct {
	DB                   *db.DB
	AnalysisPool         *worker.Pool
	ImportPool           *worker.Pool
	ChessClient          *chesscom.Client
	Templates            *template.Template
	StockfishPath        string
	StockfishDepth       int
	ArchiveLimit         int
	MaxConcurrentArchive int
}

type pageData map[string]any

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("rendering profiles page")

	profiles, err := s.DB.ListProfiles(r.Context())
	if err != nil {
		log.Error("failed to list profiles: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, r, "pages/profiles.html", pageData{
		"profiles": profiles,
		"current":  profileFromContext(r.Context()),
	})
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	if username == "" {
		log.Warn("create profile with empty username")
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	profile, err := s.DB.UpsertProfile(r.Context(), username)
	if err != nil {
		log.Error("failed to create profile: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setProfileCookie(w, profile.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleSelectProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid profile id: %s", idStr)
		http.Error(w, "invalid profile id", http.StatusBadRequest)
		return
	}

	profile, err := s.DB.GetProfile(r.Context(), id)
	if err != nil || profile == nil {
		log.Warn("profile not found for selection: %s", idStr)
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	setProfileCookie(w, profile.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid profile id for delete: %s", idStr)
		http.Error(w, "invalid profile id", http.StatusBadRequest)
		return
	}

	if err := s.DB.DeleteProfile(r.Context(), id); err != nil {
		log.Error("failed to delete profile: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if current := profileFromContext(r.Context()); current != nil && current.ID == id {
		clearProfileCookie(w)
	}
	http.Redirect(w, r, "/profiles", http.StatusSeeOther)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("rendering home page")
	profile := profileFromContext(r.Context())

	var pendingCount, totalGames, dueFlashcardsCount int
	if profile != nil {
		if count, err := s.DB.CountGamesNeedingAnalysis(r.Context(), profile.ID); err != nil {
			log.Warn("failed to count pending games: %v", err)
		} else {
			pendingCount = count
		}

		// Get total games count
		if count, err := s.DB.CountGames(r.Context(), models.GameFilter{ProfileID: profile.ID}); err != nil {
			log.Warn("failed to count total games: %v", err)
		} else {
			totalGames = count
		}

		// Get due flashcards count (using a high limit to count all due)
		if cards, err := s.DB.NextFlashcards(r.Context(), profile.ID, 10000); err != nil {
			log.Warn("failed to count due flashcards: %v", err)
		} else {
			dueFlashcardsCount = len(cards)
		}
	}

	s.render(w, r, "pages/home.html", pageData{
		"profile":              profile,
		"pending_count":        pendingCount,
		"total_games":          totalGames,
		"due_flashcards_count": dueFlashcardsCount,
	})
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context during import")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	username := profile.Username
	log = log.WithField("username", username)
	log.Info("starting game import for user")

	job := &worker.ImportGamesJob{
		DB:             s.DB,
		ChessClient:    s.ChessClient,
		Profile:        *profile,
		AnalysisPool:   s.AnalysisPool,
		StockfishPath:  s.StockfishPath,
		StockfishDepth: s.StockfishDepth,
		ArchiveLimit:   s.ArchiveLimit,
		MaxConcurrent:  s.MaxConcurrentArchive,
	}
	s.ImportPool.Submit(job)
	log.Info("import job queued")
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

func (s *Server) handleResumeAnalysis(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context during resume")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	log = log.WithFields(map[string]any{
		"profile_id": profile.ID,
		"username":   profile.Username,
	})

	if err := s.DB.ResetProcessingToPending(ctx, profile.ID); err != nil {
		log.Warn("failed to reset processing games: %v", err)
	}

	games, err := s.DB.GamesNeedingAnalysis(ctx, profile.ID)
	if err != nil {
		log.Error("failed to list games needing analysis: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, g := range games {
		s.AnalysisPool.Submit(&worker.AnalyzeGameJob{
			DB:             s.DB,
			GameID:         g.ID,
			StockfishPath:  s.StockfishPath,
			StockfishDepth: s.StockfishDepth,
		})
	}

	log.Info("queued %d games for analysis resume", len(games))
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

func (s *Server) handleQueueGameAnalysis(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid game ID for queue: %s", idStr)
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	game, err := s.DB.GetGame(ctx, id)
	if err != nil || game == nil {
		log.Warn("game not found for queue: %s", idStr)
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	if game.ProfileID != profile.ID {
		log.Warn("game does not belong to current profile during queue")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Preserve query parameters from referer
	redirectURL := "/games"
	if referer := r.Header.Get("Referer"); referer != "" {
		if parsedURL, err := url.Parse(referer); err == nil && parsedURL.Path == "/games" {
			if parsedURL.RawQuery != "" {
				redirectURL = "/games?" + parsedURL.RawQuery
			}
		}
	}

	if game.AnalysisStatus == "processing" || game.AnalysisStatus == "completed" {
		log.Info("game already %s, skipping queue", game.AnalysisStatus)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	log = log.WithFields(map[string]any{
		"game_id":   game.ID,
		"opponent":  game.Opponent,
		"status":    game.AnalysisStatus,
		"timeclass": game.TimeClass,
	})
	log.Info("queueing single game for analysis")

	s.AnalysisPool.Submit(&worker.AnalyzeGameJob{
		DB:             s.DB,
		GameID:         game.ID,
		StockfishPath:  s.StockfishPath,
		StockfishDepth: s.StockfishDepth,
	})

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	result := r.URL.Query().Get("result")
	timeClass := r.URL.Query().Get("time_class")
	opening := r.URL.Query().Get("opening")
	opponent := r.URL.Query().Get("opponent")
	pageParam := r.URL.Query().Get("page")
	perPageParam := r.URL.Query().Get("per_page")
	orderBy := r.URL.Query().Get("order_by")
	orderDir := strings.ToUpper(r.URL.Query().Get("order_dir"))

	log = log.WithFields(map[string]any{
		"result":     result,
		"time_class": timeClass,
		"opening":    opening,
		"opponent":   opponent,
		"page":       pageParam,
		"per_page":   perPageParam,
		"order_by":   orderBy,
		"order_dir":  orderDir,
	})
	log.Debug("listing games with filters")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	page := 1
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}

	perPage := 25
	switch perPageParam {
	case "10":
		perPage = 10
	case "25":
		perPage = 25
	case "50":
		perPage = 50
	case "100":
		perPage = 100
	}

	offset := (page - 1) * perPage

	if orderBy != "played_at" {
		orderBy = "played_at"
	}
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	filter := models.GameFilter{
		ProfileID:   profile.ID,
		Result:      result,
		TimeClass:   timeClass,
		OpeningName: opening,
		Opponent:    opponent,
		Limit:       perPage,
		Offset:      offset,
		OrderBy:     orderBy,
		OrderDir:    orderDir,
	}

	games, err := s.DB.ListGames(r.Context(), filter)
	if err != nil {
		log.Error("failed to list games: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := s.DB.CountGames(r.Context(), filter)
	if err != nil {
		log.Error("failed to count games: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := totalCount / perPage
	if totalCount%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	log.Debug("found %d games", len(games))
	s.render(w, r, "pages/games.html", pageData{
		"games":       games,
		"filters":     r.URL.Query(),
		"profile":     profile,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
		"total_count": totalCount,
		"order_by":    orderBy,
		"order_dir":   orderDir,
	})
}

func (s *Server) handleGameDetail(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid game ID: %s", idStr)
		http.Error(w, "invalid game ID", http.StatusBadRequest)
		return
	}

	log = log.WithField("game_id", id)
	log.Debug("fetching game detail")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	game, err := s.DB.GetGame(r.Context(), id)
	if err != nil {
		log.Error("failed to get game: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if game.ProfileID != profile.ID {
		log.Warn("game does not belong to current profile")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	positions, err := s.DB.PositionsForGame(r.Context(), id)
	if err != nil {
		log.Warn("failed to get positions for game: %v", err)
	} else {
		log.Debug("found %d positions for game", len(positions))
	}

	s.render(w, r, "pages/game_detail.html", pageData{
		"game":      game,
		"positions": positions,
	})
}

func (s *Server) handleFlashcards(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("fetching next flashcard")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	cards, err := s.DB.NextFlashcards(r.Context(), profile.ID, 1)
	if err != nil {
		log.Error("failed to get next flashcards: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var card *models.FlashcardWithPosition
	if len(cards) > 0 {
		log.Debug("loading flashcard with position, card_id=%d", cards[0].ID)
		card, err = s.DB.FlashcardWithPosition(r.Context(), cards[0].ID, profile.ID)
		if err != nil {
			log.Warn("failed to load flashcard with position: %v", err)
		}
	} else {
		log.Debug("no flashcards due for review")
	}

	s.render(w, r, "pages/flashcards.html", pageData{
		"card": card,
	})
}

func (s *Server) handleReviewFlashcard(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid flashcard ID: %s", idStr)
		http.Error(w, "invalid flashcard ID", http.StatusBadRequest)
		return
	}

	quality, err := strconv.Atoi(r.FormValue("quality"))
	if err != nil {
		log.Warn("invalid quality value: %s", r.FormValue("quality"))
		http.Error(w, "invalid quality", http.StatusBadRequest)
		return
	}

	// Get time_seconds from form (optional, defaults to 0)
	timeSeconds, _ := strconv.ParseFloat(r.FormValue("time_seconds"), 64)
	if timeSeconds < 0 {
		timeSeconds = 0
	}

	log = log.WithFields(map[string]any{
		"flashcard_id": id,
		"quality":      quality,
		"time_seconds": timeSeconds,
	})
	log.Debug("reviewing flashcard")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	card, err := s.DB.FlashcardWithPosition(r.Context(), id, profile.ID)
	if err != nil || card == nil {
		log.Warn("flashcard not found")
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}

	updated := flashcard.ApplyReview(card.Flashcard, quality)
	updated.ID = card.ID

	log.Debug("applied review, new interval=%d days, ease_factor=%.2f", updated.IntervalDays, updated.EaseFactor)

	if err := s.DB.UpdateFlashcard(r.Context(), updated); err != nil {
		log.Error("failed to update flashcard: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store review history with timing data
	if timeSeconds > 0 {
		if err := s.DB.InsertReviewHistory(r.Context(), card.ID, quality, timeSeconds); err != nil {
			log.Warn("failed to store review history: %v", err)
			// Don't fail the review if history storage fails
		}
	}

	log.Info("flashcard reviewed successfully")
	http.Redirect(w, r, "/flashcards", http.StatusSeeOther)
}

func (s *Server) handleEvaluatePosition(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	// Get FEN from query parameter
	fen := r.URL.Query().Get("fen")
	if fen == "" {
		http.Error(w, "fen parameter required", http.StatusBadRequest)
		return
	}

	// Use a lighter depth for real-time evaluation (faster response)
	depth := 15
	if s.StockfishDepth > 0 && s.StockfishDepth < 15 {
		depth = s.StockfishDepth
	}

	log = log.WithFields(map[string]any{
		"fen":   fen,
		"depth": depth,
	})
	log.Debug("evaluating position")

	// Create a new Stockfish engine for this request
	// Note: In production, you might want to use a pool of engines
	sf, err := analysis.NewEngine(s.StockfishPath)
	if err != nil {
		log.Error("failed to initialize stockfish: %v", err)
		http.Error(w, "failed to initialize engine", http.StatusInternalServerError)
		return
	}
	defer sf.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := sf.EvaluateFEN(ctx, fen, depth)
	if err != nil {
		log.Error("failed to evaluate position: %v", err)
		http.Error(w, "evaluation failed", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"cp":   result.CP,
		"mate": result.Mate,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response: %v", err)
	}
}

func (s *Server) handleOpenings(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	pageParam := r.URL.Query().Get("page")
	perPageParam := r.URL.Query().Get("per_page")

	page := 1
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}

	perPage := 25
	switch perPageParam {
	case "10":
		perPage = 10
	case "25":
		perPage = 25
	case "50":
		perPage = 50
	case "100":
		perPage = 100
	}

	offset := (page - 1) * perPage

	log = log.WithFields(map[string]any{
		"username": profile.Username,
		"page":     page,
		"per_page": perPage,
	})
	log.Debug("fetching opening stats")

	stats, err := s.DB.OpeningStats(r.Context(), profile.ID, perPage, offset)
	if err != nil {
		log.Error("failed to get opening stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := s.DB.CountOpeningStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to count opening stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := totalCount / perPage
	if totalCount%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	log.Debug("found %d opening stats (page %d of %d)", len(stats), page, totalPages)
	s.render(w, r, "pages/openings.html", pageData{
		"stats":       stats,
		"profile":     profile,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
		"total_count": totalCount,
	})
}

func (s *Server) handleOpponents(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	pageParam := r.URL.Query().Get("page")
	perPageParam := r.URL.Query().Get("per_page")
	orderBy := r.URL.Query().Get("order_by")
	orderDir := strings.ToUpper(r.URL.Query().Get("order_dir"))

	page := 1
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}

	perPage := 25
	switch perPageParam {
	case "10":
		perPage = 10
	case "25":
		perPage = 25
	case "50":
		perPage = 50
	case "100":
		perPage = 100
	}

	if orderBy != "total_games" && orderBy != "last_played_at" {
		orderBy = "total_games"
	}
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	offset := (page - 1) * perPage

	log = log.WithFields(map[string]any{
		"username":  profile.Username,
		"page":      page,
		"per_page":  perPage,
		"order_by":  orderBy,
		"order_dir": orderDir,
	})
	log.Debug("fetching opponent stats")

	stats, err := s.DB.OpponentStats(r.Context(), profile.ID, perPage, offset, orderBy, orderDir)
	if err != nil {
		log.Error("failed to get opponent stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := s.DB.CountOpponentStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to count opponent stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := totalCount / perPage
	if totalCount%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	log.Debug("found %d opponent stats (page %d of %d)", len(stats), page, totalPages)
	s.render(w, r, "pages/opponents.html", pageData{
		"stats":       stats,
		"profile":     profile,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
		"total_count": totalCount,
		"order_by":    orderBy,
		"order_dir":   orderDir,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log = log.WithField("username", profile.Username)
	log.Debug("fetching aggregate stats")

	timeStats, err := s.DB.TimeClassStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get time class stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	colorStats, err := s.DB.ColorStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get color stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	monthlyStats, err := s.DB.MonthlyStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get monthly stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mistakeStats, err := s.DB.MistakePhaseStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get mistake phase stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ratingStats, err := s.DB.RatingStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get rating stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, r, "pages/stats.html", pageData{
		"time_stats":    timeStats,
		"color_stats":   colorStats,
		"monthly_stats": monthlyStats,
		"mistake_stats": mistakeStats,
		"rating_stats":  ratingStats,
		"profile":       profile,
	})
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log.Debug("rendering analytics hub page")
	s.render(w, r, "pages/analytics.html", pageData{
		"profile": profile,
	})
}

func (s *Server) handleFlashcardAnalytics(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log = log.WithField("username", profile.Username)
	log.Debug("fetching flashcard analytics")

	overallStats, err := s.DB.FlashcardStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get flashcard stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	classificationStats, err := s.DB.FlashcardClassificationStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get classification stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	phaseStats, err := s.DB.FlashcardPhaseStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get phase stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	openingStats, err := s.DB.FlashcardOpeningStats(r.Context(), profile.ID, 20)
	if err != nil {
		log.Error("failed to get opening stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	timeStats, err := s.DB.FlashcardTimeStats(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed to get time stats: %v", err)
		// Don't fail if time stats aren't available (no reviews yet)
		timeStats = nil
	}

	s.render(w, r, "pages/flashcard_analytics.html", pageData{
		"overall_stats":        overallStats,
		"classification_stats":  classificationStats,
		"phase_stats":          phaseStats,
		"opening_stats":        openingStats,
		"time_stats":           timeStats,
		"profile":              profile,
	})
}

func (s *Server) handleRefreshStats(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context during refresh")
		http.Error(w, "no profile selected", http.StatusBadRequest)
		return
	}

	log = log.WithField("profile_id", profile.ID)
	log.Info("manually refreshing cached stats")

	if err := s.DB.RefreshProfileStats(r.Context(), profile.ID); err != nil {
		log.Error("failed to refresh stats: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info("stats refreshed successfully")

	// Redirect back to referrer or analytics page
	redirectTo := r.FormValue("redirect")
	if redirectTo == "" {
		redirectTo = r.Header.Get("Referer")
	}
	if redirectTo == "" {
		redirectTo = "/analytics"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data pageData) {
	if data == nil {
		data = pageData{}
	}
	if _, ok := data["profile"]; !ok {
		data["profile"] = profileFromContext(r.Context())
	}

	log := logger.FromContext(r.Context())
	if err := s.Templates.ExecuteTemplate(w, name, data); err != nil {
		log.Error("failed to render template %s: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deriveResult(username string, mg chesscom.MonthlyGame) (playedAs, opponent, result string) {
	if strings.EqualFold(mg.White.Username, username) {
		playedAs = "white"
		opponent = mg.Black.Username
		result = normalizeResult(mg.White.Result)
		return
	}
	playedAs = "black"
	opponent = mg.White.Username
	result = normalizeResult(mg.Black.Result)
	return
}

func normalizeResult(res string) string {
	res = strings.ToLower(res)
	switch res {
	case "win":
		return "win"
	case "stalemate", "agreed", "repetition", "timevsinsufficient", "insufficient", "fiftymove", "draw":
		return "draw"
	case "checkmated", "resigned", "timeout", "abandoned", "kingofthehill", "threecheck", "bughousepartnerlose":
		return "loss"
	default:
		return "loss"
	}
}
