package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

func (s *Server) handlePuzzleRushPage(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("rendering puzzle rush page")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	// Get current session if exists
	currentSession, err := s.PuzzleRushService.GetCurrentSession(r.Context(), profile.ID)
	if err != nil {
		log.Warn("failed to get current session: %v", err)
		// Continue anyway, might not have a session
	}

	// Get stats
	stats, err := s.PuzzleRushService.GetStats(r.Context(), profile.ID)
	if err != nil {
		log.Warn("failed to get stats: %v", err)
		stats = nil
	}

	// Get best scores
	bestScores, err := s.PuzzleRushService.GetBestScores(r.Context(), profile.ID)
	if err != nil {
		log.Warn("failed to get best scores: %v", err)
		bestScores = nil
	}

	s.render(w, r, "pages/puzzle_rush.html", pageData{
		"currentSession": currentSession,
		"stats":          stats,
		"bestScores":     bestScores,
	})
}

func (s *Server) handlePuzzleRushStart(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("starting puzzle rush")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context")
		handleError(w, r, errors.NewBadRequestError("no profile selected"))
		return
	}

	if r.Method != http.MethodPost {
		handleError(w, r, errors.NewBadRequestError("method not allowed"))
		return
	}

	difficulty := r.FormValue("difficulty")
	if difficulty == "" {
		handleError(w, r, errors.NewBadRequestError("difficulty required"))
		return
	}

	session, err := s.PuzzleRushService.StartRush(r.Context(), profile.ID, difficulty)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Get next flashcard for the session
	card, err := s.FlashcardService.GetNextFlashcard(r.Context(), profile.ID)
	if err != nil {
		log.Warn("failed to get next flashcard: %v", err)
		card = nil
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"session": session,
		"card":    card,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlePuzzleRushAnswer(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("submitting puzzle rush answer")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context")
		handleError(w, r, errors.NewBadRequestError("no profile selected"))
		return
	}

	if r.Method != http.MethodPost {
		handleError(w, r, errors.NewBadRequestError("method not allowed"))
		return
	}

	sessionIDStr := r.FormValue("session_id")
	sessionID, err := strconv.ParseInt(sessionIDStr, 10, 64)
	if err != nil {
		handleError(w, r, errors.NewBadRequestError("invalid session_id"))
		return
	}

	flashcardIDStr := r.FormValue("flashcard_id")
	flashcardID, err := strconv.ParseInt(flashcardIDStr, 10, 64)
	if err != nil {
		handleError(w, r, errors.NewBadRequestError("invalid flashcard_id"))
		return
	}

	qualityStr := r.FormValue("quality")
	quality, err := strconv.Atoi(qualityStr)
	if err != nil {
		handleError(w, r, errors.NewBadRequestError("invalid quality"))
		return
	}

	timeSeconds, _ := strconv.ParseFloat(r.FormValue("time_seconds"), 64)
	if timeSeconds < 0 {
		timeSeconds = 0
	}

	session, err := s.PuzzleRushService.SubmitAnswer(r.Context(), sessionID, profile.ID, flashcardID, quality, timeSeconds)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Get next flashcard if session is still active
	var nextCard interface{}
	if session.CompletedAt == nil {
		card, err := s.FlashcardService.GetNextFlashcard(r.Context(), profile.ID)
		if err != nil {
			log.Warn("failed to get next flashcard: %v", err)
		} else {
			nextCard = card
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"session":  session,
		"nextCard": nextCard,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlePuzzleRushCurrent(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("getting current puzzle rush session")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context")
		handleError(w, r, errors.NewBadRequestError("no profile selected"))
		return
	}

	session, err := s.PuzzleRushService.GetCurrentSession(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Get next flashcard if session is active
	var card interface{}
	if session != nil && session.CompletedAt == nil {
		nextCard, err := s.FlashcardService.GetNextFlashcard(r.Context(), profile.ID)
		if err != nil {
			log.Warn("failed to get next flashcard: %v", err)
		} else {
			card = nextCard
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"session": session,
		"card":    card,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlePuzzleRushStats(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("getting puzzle rush stats")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context")
		handleError(w, r, errors.NewBadRequestError("no profile selected"))
		return
	}

	stats, err := s.PuzzleRushService.GetStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Error("failed to encode response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
