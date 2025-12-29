package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

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

	s.ImportService.ImportGames(r.Context(), *profile)
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

	// Restart the analysis pool if it was stopped
	// Use context.Background() instead of r.Context() because HTTP request contexts
	// are cancelled when the request completes, which would stop the workers
	s.AnalysisPool.Restart(context.Background())

	ctx := r.Context()
	count, err := s.GameService.ResumeAnalysis(ctx, profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	log.Info("queued %d games for analysis resume", count)
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

func (s *Server) handleStopAnalysis(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context during stop analysis")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log.Info("stopping background analysis")
	s.AnalysisPool.Cancel()
	log.Info("background analysis stopped")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleQueueGameAnalysis(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid game ID for queue: %s", idStr)
		handleError(w, r, errors.NewBadRequestError("invalid game ID"))
		return
	}

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
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

	ctx := r.Context()
	if err := s.GameService.QueueGameAnalysis(ctx, id, profile.ID); err != nil {
		handleError(w, r, err)
		return
	}

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

	games, totalCount, err := s.GameService.ListGames(r.Context(), filter)
	if err != nil {
		handleError(w, r, err)
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
		handleError(w, r, errors.NewBadRequestError("invalid game ID"))
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

	game, err := s.GameService.GetGame(r.Context(), id, profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	positions, err := s.GameService.GetPositionsForGame(r.Context(), id)
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
