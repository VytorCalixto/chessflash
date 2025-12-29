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
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	// Check if filtering by game_id
	gameIDStr := r.URL.Query().Get("game_id")
	if gameIDStr != "" {
		gameID, err := strconv.ParseInt(gameIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid game_id parameter: %s", gameIDStr)
			handleError(w, r, errors.NewBadRequestError("invalid game_id"))
			return
		}

		// Verify game belongs to profile
		game, err := s.GameService.GetGame(r.Context(), gameID, profile.ID)
		if err != nil {
			log.Warn("failed to get game: %v", err)
			handleError(w, r, err)
			return
		}

		// Check if all cards are completed
		completed := r.URL.Query().Get("completed") == "true"

		// Get total count first
		totalCount, err := s.FlashcardService.CountFlashcardsByGame(r.Context(), gameID, profile.ID)
		if err != nil {
			handleError(w, r, err)
			return
		}

		if totalCount == 0 {
			log.Debug("no flashcards found for game")
			s.render(w, r, "pages/flashcards.html", pageData{
				"card":             nil,
				"game":             game,
				"total_count":      0,
				"current_index":    0,
				"filtered_by_game": true,
			})
			return
		}

		// If all cards are completed, show completion message
		if completed {
			s.render(w, r, "pages/flashcards.html", pageData{
				"card":             nil,
				"game":             game,
				"total_count":      totalCount,
				"current_index":    totalCount,
				"filtered_by_game": true,
				"completed":        true,
			})
			return
		}

		// Parse card_index (1-based)
		cardIndex := 1
		if idx, err := strconv.Atoi(r.URL.Query().Get("card_index")); err == nil && idx > 0 && idx <= totalCount {
			cardIndex = idx
		}

		// Fetch all flashcards and select the one at cardIndex
		allCards, _, err := s.FlashcardService.ListFlashcardsByGame(r.Context(), gameID, profile.ID, totalCount, 0)
		if err != nil {
			handleError(w, r, err)
			return
		}

		if len(allCards) == 0 || cardIndex > len(allCards) {
			handleError(w, r, errors.NewBadRequestError("invalid card_index"))
			return
		}

		currentCard := allCards[cardIndex-1]

		log = log.WithFields(map[string]any{
			"game_id":     gameID,
			"card_index":  cardIndex,
			"total_count": totalCount,
		})
		log.Debug("displaying flashcard %d of %d for game", cardIndex, totalCount)

		s.render(w, r, "pages/flashcards.html", pageData{
			"card":             &currentCard,
			"game":             game,
			"total_count":      totalCount,
			"current_index":    cardIndex,
			"filtered_by_game": true,
		})
		return
	}

	// Default behavior: show next flashcard for review
	log.Debug("fetching next flashcard")

	card, err := s.FlashcardService.GetNextFlashcard(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	if card == nil {
		log.Debug("no flashcards due for review")
	}

	s.render(w, r, "pages/flashcards.html", pageData{
		"card":             card,
		"filtered_by_game": false,
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

	// Preserve game_id and card_index if present, and advance to next card
	redirectURL := "/flashcards"
	gameIDStr := r.FormValue("game_id")
	cardIndexStr := r.FormValue("card_index")
	if gameIDStr != "" {
		redirectURL += "?game_id=" + gameIDStr
		if cardIndexStr != "" {
			// Try to increment card_index for next card
			if cardIndex, err := strconv.Atoi(cardIndexStr); err == nil {
				nextIndex := cardIndex + 1
				// Check if there are more cards (get total count)
				if gameID, err := strconv.ParseInt(gameIDStr, 10, 64); err == nil {
					if totalCount, err := s.FlashcardService.CountFlashcardsByGame(r.Context(), gameID, profile.ID); err == nil {
						if nextIndex <= totalCount {
							// More cards available, go to next
							redirectURL += "&card_index=" + strconv.Itoa(nextIndex)
						} else {
							// All cards completed, show completion message
							redirectURL += "&completed=true"
						}
					} else {
						// Fallback: just increment
						redirectURL += "&card_index=" + strconv.Itoa(nextIndex)
					}
				} else {
					// Invalid game_id, just use current card_index
					redirectURL += "&card_index=" + cardIndexStr
				}
			} else {
				// Invalid card_index, just redirect without it
			}
		}
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
