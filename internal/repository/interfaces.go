package repository

import (
	"context"
	"time"

	"github.com/vytor/chessflash/internal/models"
)

// GameRepository handles game data access
type GameRepository interface {
	Get(ctx context.Context, id int64) (*models.Game, error)
	List(ctx context.Context, filter models.GameFilter) ([]models.Game, error)
	Count(ctx context.Context, filter models.GameFilter) (int, error)
	Insert(ctx context.Context, game models.Game) (int64, error)
	InsertBatch(ctx context.Context, games []models.Game) ([]int64, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateOpening(ctx context.Context, id int64, ecoCode, openingName string) error
	ResetProcessingToPending(ctx context.Context, profileID int64) error
	GamesNeedingAnalysis(ctx context.Context, profileID int64) ([]models.Game, error)
	CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error)
	GamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) ([]models.Game, error)
	CountGamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) (int, error)
	CountGamesByStatusWithFilter(ctx context.Context, profileID int64, status string, filter models.AnalysisFilter) (int, error)
	GetExistingChessComIDs(ctx context.Context, profileID int64) (map[string]bool, error)
	CountByStatus(ctx context.Context, profileID int64, status string) (int, error)
	GetAverageAnalysisTime(ctx context.Context, profileID int64) (float64, error)
}

// PositionRepository handles position data access
type PositionRepository interface {
	Insert(ctx context.Context, position models.Position) (int64, error)
	InsertBatch(ctx context.Context, positions []models.Position) ([]int64, error)
	PositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error)
}

// FlashcardRepository handles flashcard data access
type FlashcardRepository interface {
	Insert(ctx context.Context, flashcard models.Flashcard) (int64, error)
	Update(ctx context.Context, flashcard models.Flashcard) error
	NextFlashcards(ctx context.Context, profileID int64, limit int) ([]models.Flashcard, error)
	FlashcardWithPosition(ctx context.Context, id int64, profileID int64) (*models.FlashcardWithPosition, error)
	InsertReviewHistory(ctx context.Context, flashcardID int64, quality int, timeSeconds float64) error
}

// ProfileRepository handles profile data access
type ProfileRepository interface {
	Get(ctx context.Context, id int64) (*models.Profile, error)
	List(ctx context.Context) ([]models.Profile, error)
	Upsert(ctx context.Context, username string) (*models.Profile, error)
	UpdateSync(ctx context.Context, id int64, t time.Time) error
	Delete(ctx context.Context, id int64) error
}

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
