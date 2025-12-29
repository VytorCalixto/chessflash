package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vytor/chessflash/internal/models"
)

// MockFlashcardRepository is a mock implementation of repository.FlashcardRepository
type MockFlashcardRepository struct {
	mock.Mock
}

func (m *MockFlashcardRepository) Insert(ctx context.Context, flashcard models.Flashcard) (int64, error) {
	args := m.Called(ctx, flashcard)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFlashcardRepository) Update(ctx context.Context, flashcard models.Flashcard) error {
	args := m.Called(ctx, flashcard)
	return args.Error(0)
}

func (m *MockFlashcardRepository) NextFlashcards(ctx context.Context, profileID int64, limit int) ([]models.Flashcard, error) {
	args := m.Called(ctx, profileID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Flashcard), args.Error(1)
}

func (m *MockFlashcardRepository) FlashcardWithPosition(ctx context.Context, id int64, profileID int64) (*models.FlashcardWithPosition, error) {
	args := m.Called(ctx, id, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FlashcardWithPosition), args.Error(1)
}

func (m *MockFlashcardRepository) InsertReviewHistory(ctx context.Context, flashcardID int64, quality int, timeSeconds float64) error {
	args := m.Called(ctx, flashcardID, quality, timeSeconds)
	return args.Error(0)
}
