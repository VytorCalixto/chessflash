package repository

import (
	"context"

	"github.com/vytor/chessflash/internal/models"
)

// PositionRepository handles position data access
type PositionRepository interface {
	Insert(ctx context.Context, position models.Position) (int64, error)
	InsertBatch(ctx context.Context, positions []models.Position) ([]int64, error)
	PositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error)
}
