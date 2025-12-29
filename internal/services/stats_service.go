package services

import (
	"context"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

// StatsService handles statistics-related business logic
type StatsService interface {
	GetOpeningStats(ctx context.Context, profileID int64, limit, offset int) ([]models.OpeningStat, int, error)
	GetOpponentStats(ctx context.Context, profileID int64, limit, offset int, orderBy, orderDir string) ([]models.OpponentStat, int, error)
	GetTimeClassStats(ctx context.Context, profileID int64) ([]models.TimeClassStat, error)
	GetColorStats(ctx context.Context, profileID int64) ([]models.ColorStat, error)
	GetMonthlyStats(ctx context.Context, profileID int64) ([]models.MonthlyStat, error)
	GetMistakePhaseStats(ctx context.Context, profileID int64) ([]models.MistakePhaseStat, error)
	GetRatingStats(ctx context.Context, profileID int64) ([]models.RatingStat, error)
	GetFlashcardStats(ctx context.Context, profileID int64) (*models.FlashcardStat, error)
	GetFlashcardClassificationStats(ctx context.Context, profileID int64) ([]models.FlashcardClassificationStat, error)
	GetFlashcardPhaseStats(ctx context.Context, profileID int64) ([]models.FlashcardPhaseStat, error)
	GetFlashcardOpeningStats(ctx context.Context, profileID int64, limit int) ([]models.FlashcardOpeningStat, error)
	GetFlashcardTimeStats(ctx context.Context, profileID int64) (*models.FlashcardTimeStat, error)
	RefreshStats(ctx context.Context, profileID int64) error
}

type statsService struct {
	statsRepo repository.StatsRepository
}

// NewStatsService creates a new StatsService
func NewStatsService(statsRepo repository.StatsRepository) StatsService {
	return &statsService{statsRepo: statsRepo}
}

func (s *statsService) GetOpeningStats(ctx context.Context, profileID int64, limit, offset int) ([]models.OpeningStat, int, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting opening stats: profile_id=%d, limit=%d, offset=%d", profileID, limit, offset)

	stats, err := s.statsRepo.OpeningStats(ctx, profileID, limit, offset)
	if err != nil {
		log.Error("failed to get opening stats: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

		totalCount, err := s.statsRepo.CountOpeningStats(ctx, profileID)
	if err != nil {
		log.Error("failed to count opening stats: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

	return stats, totalCount, nil
}

func (s *statsService) GetOpponentStats(ctx context.Context, profileID int64, limit, offset int, orderBy, orderDir string) ([]models.OpponentStat, int, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting opponent stats: profile_id=%d, limit=%d, offset=%d", profileID, limit, offset)

	stats, err := s.statsRepo.OpponentStats(ctx, profileID, limit, offset, orderBy, orderDir)
	if err != nil {
		log.Error("failed to get opponent stats: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

		totalCount, err := s.statsRepo.CountOpponentStats(ctx, profileID)
	if err != nil {
		log.Error("failed to count opponent stats: %v", err)
		return nil, 0, errors.NewInternalError(err)
	}

	return stats, totalCount, nil
}

func (s *statsService) GetTimeClassStats(ctx context.Context, profileID int64) ([]models.TimeClassStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting time class stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.TimeClassStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get time class stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetColorStats(ctx context.Context, profileID int64) ([]models.ColorStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting color stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.ColorStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get color stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetMonthlyStats(ctx context.Context, profileID int64) ([]models.MonthlyStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting monthly stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.MonthlyStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get monthly stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetMistakePhaseStats(ctx context.Context, profileID int64) ([]models.MistakePhaseStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting mistake phase stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.MistakePhaseStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get mistake phase stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetRatingStats(ctx context.Context, profileID int64) ([]models.RatingStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting rating stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.RatingStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get rating stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetFlashcardStats(ctx context.Context, profileID int64) (*models.FlashcardStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting flashcard stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.FlashcardStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get flashcard stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetFlashcardClassificationStats(ctx context.Context, profileID int64) ([]models.FlashcardClassificationStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting flashcard classification stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.FlashcardClassificationStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get classification stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetFlashcardPhaseStats(ctx context.Context, profileID int64) ([]models.FlashcardPhaseStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting flashcard phase stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.FlashcardPhaseStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get phase stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetFlashcardOpeningStats(ctx context.Context, profileID int64, limit int) ([]models.FlashcardOpeningStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting flashcard opening stats: profile_id=%d, limit=%d", profileID, limit)

	stats, err := s.statsRepo.FlashcardOpeningStats(ctx, profileID, limit)
	if err != nil {
		log.Error("failed to get opening stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *statsService) GetFlashcardTimeStats(ctx context.Context, profileID int64) (*models.FlashcardTimeStat, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting flashcard time stats: profile_id=%d", profileID)

	stats, err := s.statsRepo.FlashcardTimeStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get time stats: %v", err)
		// Don't fail if time stats aren't available (no reviews yet)
		return nil, nil
	}

	return stats, nil
}

func (s *statsService) RefreshStats(ctx context.Context, profileID int64) error {
	log := logger.FromContext(ctx)
	log.Debug("refreshing stats: profile_id=%d", profileID)

	if err := s.statsRepo.RefreshProfileStats(ctx, profileID); err != nil {
		log.Error("failed to refresh stats: %v", err)
		return errors.NewInternalError(err)
	}

	return nil
}
