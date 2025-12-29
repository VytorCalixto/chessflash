package api

import (
	"net/http"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("rendering home page")
	profile := profileFromContext(r.Context())

	var pendingCount, totalGames, dueFlashcardsCount int
	if profile != nil {
		if count, err := s.GameService.CountGamesNeedingAnalysis(r.Context(), profile.ID); err != nil {
			log.Warn("failed to count pending games: %v", err)
		} else {
			pendingCount = count
		}

		// Get total games count
		_, count, err := s.GameService.ListGames(r.Context(), models.GameFilter{ProfileID: profile.ID, Limit: 1})
		if err != nil {
			log.Warn("failed to count total games: %v", err)
		} else {
			totalGames = count
		}

		// Get due flashcards count - we'll need to add a count method to the service later
		// For now, just check if there's at least one
		if card, err := s.FlashcardService.GetNextFlashcard(r.Context(), profile.ID); err != nil {
			log.Warn("failed to count due flashcards: %v", err)
		} else if card != nil {
			dueFlashcardsCount = 1 // Simplified - would need a count method for accurate count
		}
	}

	s.render(w, r, "pages/home.html", pageData{
		"profile":              profile,
		"pending_count":        pendingCount,
		"total_games":          totalGames,
		"due_flashcards_count": dueFlashcardsCount,
	})
}
