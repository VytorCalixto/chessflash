package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

func (s *Server) handleFlashcards(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("fetching next flashcard")

	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	card, err := s.FlashcardService.GetNextFlashcard(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	if card == nil {
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
		handleError(w, r, errors.NewBadRequestError("invalid flashcard ID"))
		return
	}

	quality, err := strconv.Atoi(r.FormValue("quality"))
	if err != nil {
		log.Warn("invalid quality value: %s", r.FormValue("quality"))
		handleError(w, r, errors.NewBadRequestError("invalid quality"))
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

	if err := s.FlashcardService.ReviewFlashcard(r.Context(), id, profile.ID, quality, timeSeconds); err != nil {
		handleError(w, r, err)
		return
	}

	log.Info("flashcard reviewed successfully")
	http.Redirect(w, r, "/flashcards", http.StatusSeeOther)
}
