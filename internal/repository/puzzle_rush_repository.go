package repository

import (
	"context"

	"github.com/vytor/chessflash/internal/models"
)

// PuzzleRushRepository handles puzzle rush data access
type PuzzleRushRepository interface {
	InsertSession(ctx context.Context, session models.PuzzleRushSession) (int64, error)
	UpdateSession(ctx context.Context, session models.PuzzleRushSession) error
	GetSession(ctx context.Context, sessionID int64) (*models.PuzzleRushSession, error)
	GetActiveSession(ctx context.Context, profileID int64) (*models.PuzzleRushSession, error)
	InsertAttempt(ctx context.Context, attempt models.PuzzleRushAttempt) (int64, error)
	GetSessionAttempts(ctx context.Context, sessionID int64) ([]models.PuzzleRushAttempt, error)
	GetUserStats(ctx context.Context, profileID int64) (*models.PuzzleRushStats, error)
	GetBestScores(ctx context.Context, profileID int64) ([]models.PuzzleRushBestScore, error)
}
