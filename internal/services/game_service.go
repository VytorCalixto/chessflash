package services

import (
	"context"
	"database/sql"

	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/worker"
)

// GameService handles game-related business logic
type GameService interface {
	GetGame(ctx context.Context, id int64, profileID int64) (*models.Game, error)
	ListGames(ctx context.Context, filter models.GameFilter) ([]models.Game, int, error)
	GetPositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error)
	QueueGameAnalysis(ctx context.Context, gameID int64, profileID int64, analysisPool *worker.Pool, stockfishPath string, stockfishDepth int) error
	ResumeAnalysis(ctx context.Context, profileID int64, analysisPool *worker.Pool, stockfishPath string, stockfishDepth int) (int, error)
	CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error)
}

type gameService struct {
	db *db.DB
}

// NewGameService creates a new GameService
func NewGameService(db *db.DB) GameService {
	return &gameService{db: db}
}

func (s *gameService) GetGame(ctx context.Context, id int64, profileID int64) (*models.Game, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting game: id=%d, profile_id=%d", id, profileID)

	game, err := s.db.GetGame(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("game", id)
		}
		log.Error("failed to get game: %v", err)
		return nil, errors.NewInternalError(err)
	}

	if game == nil {
		return nil, errors.NewNotFoundError("game", id)
	}

	// Verify game belongs to profile
	if game.ProfileID != profileID {
		return nil, errors.NewNotFoundError("game", id)
	}

	return game, nil
}

func (s *gameService) ListGames(ctx context.Context, filter models.GameFilter) ([]models.Game, int, error) {
	log := logger.FromContext(ctx)
	log.Debug("listing games with filter: profile_id=%d", filter.ProfileID)

	games, err := s.db.ListGames(ctx, filter)
	if err != nil {
		log.Error("failed to list games: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

	totalCount, err := s.db.CountGames(ctx, filter)
	if err != nil {
		log.Error("failed to count games: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

	return games, totalCount, nil
}

func (s *gameService) GetPositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting positions for game: game_id=%d", gameID)

	positions, err := s.db.PositionsForGame(ctx, gameID)
	if err != nil {
		log.Error("failed to get positions: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return positions, nil
}

func (s *gameService) QueueGameAnalysis(ctx context.Context, gameID int64, profileID int64, analysisPool *worker.Pool, stockfishPath string, stockfishDepth int) error {
	log := logger.FromContext(ctx)
	log.Debug("queueing game analysis: game_id=%d", gameID)

	game, err := s.db.GetGame(ctx, gameID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewNotFoundError("game", gameID)
		}
		log.Error("failed to get game: %v", err)
		return errors.NewInternalError(err)
	}

	if game == nil {
		return errors.NewNotFoundError("game", gameID)
	}

	// Verify game belongs to profile
	if game.ProfileID != profileID {
		return errors.NewNotFoundError("game", gameID)
	}

	// Check if already processing or completed
	if game.AnalysisStatus == "processing" || game.AnalysisStatus == "completed" {
		log.Debug("game already %s, skipping queue", game.AnalysisStatus)
		return nil
	}

	analysisPool.Submit(&worker.AnalyzeGameJob{
		DB:             s.db,
		GameID:         gameID,
		StockfishPath:  stockfishPath,
		StockfishDepth: stockfishDepth,
	})

	return nil
}

func (s *gameService) ResumeAnalysis(ctx context.Context, profileID int64, analysisPool *worker.Pool, stockfishPath string, stockfishDepth int) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("resuming analysis for profile: profile_id=%d", profileID)

	// Reset any stuck processing games
	if err := s.db.ResetProcessingToPending(ctx, profileID); err != nil {
		log.Warn("failed to reset processing games: %v", err)
	}

	games, err := s.db.GamesNeedingAnalysis(ctx, profileID)
	if err != nil {
		log.Error("failed to list games needing analysis: %v", err)
		return 0, errors.NewInternalError(err)
	}

	for _, g := range games {
		analysisPool.Submit(&worker.AnalyzeGameJob{
			DB:             s.db,
			GameID:         g.ID,
			StockfishPath:  stockfishPath,
			StockfishDepth: stockfishDepth,
		})
	}

	return len(games), nil
}

func (s *gameService) CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("counting games needing analysis: profile_id=%d", profileID)

	count, err := s.db.CountGamesNeedingAnalysis(ctx, profileID)
	if err != nil {
		log.Error("failed to count games needing analysis: %v", err)
		return 0, errors.NewInternalError(err)
	}

	return count, nil
}
