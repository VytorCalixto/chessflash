package repository

import (
	"context"

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
