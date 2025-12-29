package services

import (
	"context"
	"database/sql"

	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/flashcard"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

// FlashcardService handles flashcard-related business logic
type FlashcardService interface {
	GetNextFlashcard(ctx context.Context, profileID int64) (*models.FlashcardWithPosition, error)
	ReviewFlashcard(ctx context.Context, flashcardID int64, profileID int64, quality int, timeSeconds float64) error
}

type flashcardService struct {
	db *db.DB
}

// NewFlashcardService creates a new FlashcardService
func NewFlashcardService(db *db.DB) FlashcardService {
	return &flashcardService{db: db}
}

func (s *flashcardService) GetNextFlashcard(ctx context.Context, profileID int64) (*models.FlashcardWithPosition, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting next flashcard: profile_id=%d", profileID)

	cards, err := s.db.NextFlashcards(ctx, profileID, 1)
	if err != nil {
		log.Error("failed to get next flashcards: %v", err)
		return nil, errors.NewInternalError(err)
	}

	if len(cards) == 0 {
		log.Debug("no flashcards due for review")
		return nil, nil
	}

	card, err := s.db.FlashcardWithPosition(ctx, cards[0].ID, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("flashcard", cards[0].ID)
		}
		log.Error("failed to load flashcard with position: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return card, nil
}

func (s *flashcardService) ReviewFlashcard(ctx context.Context, flashcardID int64, profileID int64, quality int, timeSeconds float64) error {
	log := logger.FromContext(ctx)
	log.Debug("reviewing flashcard: flashcard_id=%d, quality=%d", flashcardID, quality)

	// Validate quality
	if quality < 0 || quality > 5 {
		return errors.NewValidationError("quality", "must be between 0 and 5")
	}

	// Get flashcard and verify it belongs to profile
	card, err := s.db.FlashcardWithPosition(ctx, flashcardID, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewNotFoundError("flashcard", flashcardID)
		}
		log.Error("failed to get flashcard: %v", err)
		return errors.NewInternalError(err)
	}

	if card == nil {
		return errors.NewNotFoundError("flashcard", flashcardID)
	}

	// Apply spaced repetition algorithm
	updated := flashcard.ApplyReview(card.Flashcard, quality)
	updated.ID = card.ID

	log.Debug("applied review, new interval=%d days, ease_factor=%.2f", updated.IntervalDays, updated.EaseFactor)

	// Update flashcard
	if err := s.db.UpdateFlashcard(ctx, updated); err != nil {
		log.Error("failed to update flashcard: %v", err)
		return errors.NewInternalError(err)
	}

	// Store review history with timing data (non-blocking)
	if timeSeconds > 0 {
		if err := s.db.InsertReviewHistory(ctx, card.ID, quality, timeSeconds); err != nil {
			log.Warn("failed to store review history: %v", err)
			// Don't fail the review if history storage fails
		}
	}

	return nil
}
