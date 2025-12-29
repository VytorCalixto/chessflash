package services

import (
	"context"
	"database/sql"

	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/jobs"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

// GameService handles game-related business logic
type GameService interface {
	GetGame(ctx context.Context, id int64, profileID int64) (*models.Game, error)
	ListGames(ctx context.Context, filter models.GameFilter) ([]models.Game, int, error)
	GetPositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error)
	QueueGameAnalysis(ctx context.Context, gameID int64, profileID int64) error
	ResumeAnalysis(ctx context.Context, profileID int64) (int, error)
	CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error)
	CountGamesByStatus(ctx context.Context, profileID int64, status string) (int, error)
	GetAverageAnalysisTime(ctx context.Context, profileID int64) (float64, error)
	CountGamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) (int, error)
	QueueGamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) (int, error)
	CountGamesByStatusWithFilter(ctx context.Context, profileID int64, status string, filter models.AnalysisFilter) (int, error)
}

type gameService struct {
	gameRepo     repository.GameRepository
	positionRepo repository.PositionRepository
	jobQueue     jobs.JobQueue
	db           *db.DB // Temporary: will be removed in Phase 4 when jobs use repositories
}

// NewGameService creates a new GameService
func NewGameService(gameRepo repository.GameRepository, positionRepo repository.PositionRepository, jobQueue jobs.JobQueue, database *db.DB) GameService {
	return &gameService{
		gameRepo:     gameRepo,
		positionRepo: positionRepo,
		jobQueue:     jobQueue,
		db:           database,
	}
}

func (s *gameService) GetGame(ctx context.Context, id int64, profileID int64) (*models.Game, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting game: id=%d, profile_id=%d", id, profileID)

	game, err := s.gameRepo.Get(ctx, id)
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

	games, err := s.gameRepo.List(ctx, filter)
	if err != nil {
		log.Error("failed to list games: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

	totalCount, err := s.gameRepo.Count(ctx, filter)
	if err != nil {
		log.Error("failed to count games: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

	return games, totalCount, nil
}

func (s *gameService) GetPositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting positions for game: game_id=%d", gameID)

	positions, err := s.positionRepo.PositionsForGame(ctx, gameID)
	if err != nil {
		log.Error("failed to get positions: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return positions, nil
}

func (s *gameService) QueueGameAnalysis(ctx context.Context, gameID int64, profileID int64) error {
	log := logger.FromContext(ctx)
	log.Debug("queueing game analysis: game_id=%d", gameID)

	game, err := s.gameRepo.Get(ctx, gameID)
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

	return s.jobQueue.EnqueueAnalysis(gameID)
}

func (s *gameService) ResumeAnalysis(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("resuming analysis for profile: profile_id=%d", profileID)

	// Reset any stuck processing games
	if err := s.gameRepo.ResetProcessingToPending(ctx, profileID); err != nil {
		log.Warn("failed to reset processing games: %v", err)
	}

	games, err := s.gameRepo.GamesNeedingAnalysis(ctx, profileID)
	if err != nil {
		log.Error("failed to list games needing analysis: %v", err)
		return 0, errors.NewInternalError(err)
	}

	for _, g := range games {
		if err := s.jobQueue.EnqueueAnalysis(g.ID); err != nil {
			log.Warn("failed to enqueue analysis for game %d: %v", g.ID, err)
		}
	}

	return len(games), nil
}

func (s *gameService) CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("counting games needing analysis: profile_id=%d", profileID)

	count, err := s.gameRepo.CountGamesNeedingAnalysis(ctx, profileID)
	if err != nil {
		log.Error("failed to count games needing analysis: %v", err)
		return 0, errors.NewInternalError(err)
	}

	return count, nil
}

func (s *gameService) CountGamesByStatus(ctx context.Context, profileID int64, status string) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("counting games by status: profile_id=%d, status=%s", profileID, status)

	count, err := s.gameRepo.CountByStatus(ctx, profileID, status)
	if err != nil {
		log.Error("failed to count games by status: %v", err)
		return 0, errors.NewInternalError(err)
	}

	return count, nil
}

func (s *gameService) GetAverageAnalysisTime(ctx context.Context, profileID int64) (float64, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting average analysis time: profile_id=%d", profileID)

	avgTime, err := s.gameRepo.GetAverageAnalysisTime(ctx, profileID)
	if err != nil {
		log.Error("failed to get average analysis time: %v", err)
		return 0, errors.NewInternalError(err)
	}

	return avgTime, nil
}

func (s *gameService) CountGamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("counting games for analysis: profile_id=%d", filter.ProfileID)

	count, err := s.gameRepo.CountGamesForAnalysis(ctx, filter)
	if err != nil {
		log.Error("failed to count games for analysis: %v", err)
		return 0, errors.NewInternalError(err)
	}

	return count, nil
}

func (s *gameService) QueueGamesForAnalysis(ctx context.Context, filter models.AnalysisFilter) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("queueing games for analysis: profile_id=%d", filter.ProfileID)

	// Reset any stuck processing games
	if err := s.gameRepo.ResetProcessingToPending(ctx, filter.ProfileID); err != nil {
		log.Warn("failed to reset processing games: %v", err)
	}

	games, err := s.gameRepo.GamesForAnalysis(ctx, filter)
	if err != nil {
		log.Error("failed to list games for analysis: %v", err)
		return 0, errors.NewInternalError(err)
	}

	queuedCount := 0
	rejectedCount := 0
	for _, g := range games {
		if err := s.jobQueue.EnqueueAnalysis(g.ID); err != nil {
			if err.Error() == "job queue is full" {
				rejectedCount++
			}
			log.Warn("failed to enqueue analysis for game %d: %v", g.ID, err)
		} else {
			queuedCount++
		}
	}

	log.Info("queued %d games for analysis (rejected: %d)", queuedCount, rejectedCount)

	// Start automatic backfill if there were rejected games or if we have more games to process
	if rejectedCount > 0 || len(games) > queuedCount {
		if backfillQueue, ok := s.jobQueue.(interface {
			StartBackfill(models.AnalysisFilter)
		}); ok {
			backfillQueue.StartBackfill(filter)
		}
	}

	return queuedCount, nil
}

// StopBackfill stops the automatic backfill process if it's running
func (s *gameService) StopBackfill() {
	if backfillQueue, ok := s.jobQueue.(interface {
		StopBackfill()
	}); ok {
		backfillQueue.StopBackfill()
	}
}

func (s *gameService) CountGamesByStatusWithFilter(ctx context.Context, profileID int64, status string, filter models.AnalysisFilter) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("counting games by status with filter: profile_id=%d, status=%s", profileID, status)

	count, err := s.gameRepo.CountGamesByStatusWithFilter(ctx, profileID, status, filter)
	if err != nil {
		log.Error("failed to count games by status with filter: %v", err)
		return 0, errors.NewInternalError(err)
	}

	return count, nil
}
