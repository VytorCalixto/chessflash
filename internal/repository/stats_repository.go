package repository

import (
	"context"
	"time"

	"github.com/vytor/chessflash/internal/models"
)

// StatsRepository handles statistics data access
type StatsRepository interface {
	OpeningStats(ctx context.Context, profileID int64, limit, offset int) ([]models.OpeningStat, error)
	CountOpeningStats(ctx context.Context, profileID int64) (int, error)
	OpponentStats(ctx context.Context, profileID int64, limit, offset int, orderBy, orderDir string) ([]models.OpponentStat, error)
	CountOpponentStats(ctx context.Context, profileID int64) (int, error)
	TimeClassStats(ctx context.Context, profileID int64, dateCutoff *time.Time) ([]models.TimeClassStat, error)
	ColorStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.ColorStat, error)
	MonthlyStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.MonthlyStat, error)
	MistakePhaseStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.MistakePhaseStat, error)
	RatingStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.RatingStat, error)
	FlashcardStats(ctx context.Context, profileID int64) (*models.FlashcardStat, error)
	FlashcardClassificationStats(ctx context.Context, profileID int64) ([]models.FlashcardClassificationStat, error)
	FlashcardPhaseStats(ctx context.Context, profileID int64) ([]models.FlashcardPhaseStat, error)
	FlashcardOpeningStats(ctx context.Context, profileID int64, limit int) ([]models.FlashcardOpeningStat, error)
	FlashcardTimeStats(ctx context.Context, profileID int64) (*models.FlashcardTimeStat, error)
	SummaryStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) (*models.SummaryStat, error)
	RefreshProfileStats(ctx context.Context, profileID int64) error
}
