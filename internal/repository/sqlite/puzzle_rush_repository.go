package sqlite

import (
	"context"

	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

type puzzleRushRepository struct {
	db *db.DB
}

// NewPuzzleRushRepository creates a new PuzzleRushRepository implementation
func NewPuzzleRushRepository(db *db.DB) repository.PuzzleRushRepository {
	return &puzzleRushRepository{db: db}
}

func (r *puzzleRushRepository) InsertSession(ctx context.Context, session models.PuzzleRushSession) (int64, error) {
	return r.db.InsertPuzzleRushSession(ctx, session)
}

func (r *puzzleRushRepository) UpdateSession(ctx context.Context, session models.PuzzleRushSession) error {
	return r.db.UpdatePuzzleRushSession(ctx, session)
}

func (r *puzzleRushRepository) GetSession(ctx context.Context, sessionID int64) (*models.PuzzleRushSession, error) {
	return r.db.GetPuzzleRushSession(ctx, sessionID)
}

func (r *puzzleRushRepository) GetActiveSession(ctx context.Context, profileID int64) (*models.PuzzleRushSession, error) {
	return r.db.GetActivePuzzleRushSession(ctx, profileID)
}

func (r *puzzleRushRepository) InsertAttempt(ctx context.Context, attempt models.PuzzleRushAttempt) (int64, error) {
	return r.db.InsertPuzzleRushAttempt(ctx, attempt)
}

func (r *puzzleRushRepository) GetSessionAttempts(ctx context.Context, sessionID int64) ([]models.PuzzleRushAttempt, error) {
	return r.db.GetPuzzleRushSessionAttempts(ctx, sessionID)
}

func (r *puzzleRushRepository) GetUserStats(ctx context.Context, profileID int64) (*models.PuzzleRushStats, error) {
	return r.db.GetPuzzleRushUserStats(ctx, profileID)
}

func (r *puzzleRushRepository) GetBestScores(ctx context.Context, profileID int64) ([]models.PuzzleRushBestScore, error) {
	return r.db.GetPuzzleRushBestScores(ctx, profileID)
}
