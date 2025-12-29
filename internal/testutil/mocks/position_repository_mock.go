package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vytor/chessflash/internal/models"
)

// MockPositionRepository is a mock implementation of repository.PositionRepository
type MockPositionRepository struct {
	mock.Mock
}

func (m *MockPositionRepository) Insert(ctx context.Context, position models.Position) (int64, error) {
	args := m.Called(ctx, position)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPositionRepository) InsertBatch(ctx context.Context, positions []models.Position) ([]int64, error) {
	args := m.Called(ctx, positions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockPositionRepository) PositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error) {
	args := m.Called(ctx, gameID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Position), args.Error(1)
}
