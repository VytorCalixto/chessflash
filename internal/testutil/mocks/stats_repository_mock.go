package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/vytor/chessflash/internal/models"
)

// MockStatsRepository is a mock implementation of repository.StatsRepository
type MockStatsRepository struct {
	mock.Mock
}

func (m *MockStatsRepository) OpeningStats(ctx context.Context, profileID int64, limit, offset int) ([]models.OpeningStat, error) {
	args := m.Called(ctx, profileID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.OpeningStat), args.Error(1)
}

func (m *MockStatsRepository) CountOpeningStats(ctx context.Context, profileID int64) (int, error) {
	args := m.Called(ctx, profileID)
	return args.Int(0), args.Error(1)
}

func (m *MockStatsRepository) OpponentStats(ctx context.Context, profileID int64, limit, offset int, orderBy, orderDir string) ([]models.OpponentStat, error) {
	args := m.Called(ctx, profileID, limit, offset, orderBy, orderDir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.OpponentStat), args.Error(1)
}

func (m *MockStatsRepository) CountOpponentStats(ctx context.Context, profileID int64) (int, error) {
	args := m.Called(ctx, profileID)
	return args.Int(0), args.Error(1)
}

func (m *MockStatsRepository) TimeClassStats(ctx context.Context, profileID int64, dateCutoff *time.Time) ([]models.TimeClassStat, error) {
	args := m.Called(ctx, profileID, dateCutoff)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TimeClassStat), args.Error(1)
}

func (m *MockStatsRepository) ColorStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.ColorStat, error) {
	args := m.Called(ctx, profileID, timeClass, dateCutoff)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ColorStat), args.Error(1)
}

func (m *MockStatsRepository) MonthlyStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.MonthlyStat, error) {
	args := m.Called(ctx, profileID, timeClass, dateCutoff)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.MonthlyStat), args.Error(1)
}

func (m *MockStatsRepository) MistakePhaseStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.MistakePhaseStat, error) {
	args := m.Called(ctx, profileID, timeClass, dateCutoff)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.MistakePhaseStat), args.Error(1)
}

func (m *MockStatsRepository) RatingStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.RatingStat, error) {
	args := m.Called(ctx, profileID, timeClass, dateCutoff)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RatingStat), args.Error(1)
}

func (m *MockStatsRepository) FlashcardStats(ctx context.Context, profileID int64) (*models.FlashcardStat, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FlashcardStat), args.Error(1)
}

func (m *MockStatsRepository) FlashcardClassificationStats(ctx context.Context, profileID int64) ([]models.FlashcardClassificationStat, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FlashcardClassificationStat), args.Error(1)
}

func (m *MockStatsRepository) FlashcardPhaseStats(ctx context.Context, profileID int64) ([]models.FlashcardPhaseStat, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FlashcardPhaseStat), args.Error(1)
}

func (m *MockStatsRepository) FlashcardOpeningStats(ctx context.Context, profileID int64, limit int) ([]models.FlashcardOpeningStat, error) {
	args := m.Called(ctx, profileID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FlashcardOpeningStat), args.Error(1)
}

func (m *MockStatsRepository) FlashcardTimeStats(ctx context.Context, profileID int64) (*models.FlashcardTimeStat, error) {
	args := m.Called(ctx, profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FlashcardTimeStat), args.Error(1)
}

func (m *MockStatsRepository) SummaryStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) (*models.SummaryStat, error) {
	args := m.Called(ctx, profileID, timeClass, dateCutoff)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SummaryStat), args.Error(1)
}

func (m *MockStatsRepository) RefreshProfileStats(ctx context.Context, profileID int64) error {
	args := m.Called(ctx, profileID)
	return args.Error(0)
}
