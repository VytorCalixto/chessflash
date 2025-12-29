package repository

import (
	"context"

	"github.com/vytor/chessflash/internal/models"
)

// FlashcardRepository handles flashcard data access
type FlashcardRepository interface {
	Insert(ctx context.Context, flashcard models.Flashcard) (int64, error)
	Update(ctx context.Context, flashcard models.Flashcard) error
	NextFlashcards(ctx context.Context, profileID int64, limit int) ([]models.Flashcard, error)
	FlashcardWithPosition(ctx context.Context, id int64, profileID int64) (*models.FlashcardWithPosition, error)
	InsertReviewHistory(ctx context.Context, flashcardID int64, quality int, timeSeconds float64) error
}
