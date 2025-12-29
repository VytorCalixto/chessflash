package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vytor/chessflash/internal/models"
)

// MockGameRepository is a mock implementation of repository.GameRepository
type MockGameRepository struct {
	mock.Mock
}

func (m *MockGameRepository) Get(ctx context.Context, id int64) (*models.Game, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Game), args.Error(1)
}

func (m *MockGameRepository) List(ctx context.Context, filter models.GameFilter) ([]models.Game, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Game), args.Error(1)
}

func (m *MockGameRepository) Count(ctx context.Context, filter models.GameFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockGameRepository) Insert(ctx context.Context, game models.Game) (int64, error) {
	args := m.Called(ctx, game)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockGameRepository) InsertBatch(ctx context.Context, games []models.Game) ([]int64, error) {
	args := m.Called(ctx, games)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockGameRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockGameRepository) UpdateOpening(ctx context.Context, id int64, ecoCode, openingName string) error {
	args := m.Called(ctx, id, ecoCode, openingName)
	return args.Error(0)
}

func (m *MockGameRepository) ResetProcessingToPending(ctx context.Context, profileID int64) error {
	args := m.Called(ctx, profileID)
	return args.Error(0)
}

func (m *MockGameRepository) GamesNeedingAnalysis(ctx context.Context, profileID int64) ([]models.Game, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Game), args.Error(1)
}

func (m *MockGameRepository) CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error) {
	args := m.Called(ctx, profileID)
	return args.Int(0), args.Error(1)
}

func (m *MockGameRepository) GamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) ([]models.Game, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Game), args.Error(1)
}

func (m *MockGameRepository) CountGamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockGameRepository) CountGamesByStatusWithFilter(ctx context.Context, profileID int64, status string, filter models.AnalysisFilter) (int, error) {
	args := m.Called(ctx, profileID, status, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockGameRepository) GetExistingChessComIDs(ctx context.Context, profileID int64) (map[string]bool, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockGameRepository) CountByStatus(ctx context.Context, profileID int64, status string) (int, error) {
	args := m.Called(ctx, profileID, status)
	return args.Int(0), args.Error(1)
}

func (m *MockGameRepository) GetAverageAnalysisTime(ctx context.Context, profileID int64) (float64, error) {
	args := m.Called(ctx, profileID)
	return args.Get(0).(float64), args.Error(1)
}
