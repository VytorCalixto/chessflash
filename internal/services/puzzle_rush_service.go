package services

import (
	"context"
	"time"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

// PuzzleRushService handles puzzle rush-related business logic
type PuzzleRushService interface {
	StartRush(ctx context.Context, profileID int64, difficulty string) (*models.PuzzleRushSession, error)
	GetCurrentSession(ctx context.Context, profileID int64) (*models.PuzzleRushSession, error)
	SubmitAnswer(ctx context.Context, sessionID int64, profileID int64, flashcardID int64, quality int, timeSeconds float64) (*models.PuzzleRushSession, error)
	EndRush(ctx context.Context, sessionID int64, profileID int64) error
	GetStats(ctx context.Context, profileID int64) (*models.PuzzleRushStats, error)
	GetBestScores(ctx context.Context, profileID int64) ([]models.PuzzleRushBestScore, error)
}

type puzzleRushService struct {
	rushRepo      repository.PuzzleRushRepository
	flashcardRepo repository.FlashcardRepository
	flashcardSvc  FlashcardService
}

// NewPuzzleRushService creates a new PuzzleRushService
func NewPuzzleRushService(rushRepo repository.PuzzleRushRepository, flashcardRepo repository.FlashcardRepository, flashcardSvc FlashcardService) PuzzleRushService {
	return &puzzleRushService{
		rushRepo:      rushRepo,
		flashcardRepo: flashcardRepo,
		flashcardSvc:  flashcardSvc,
	}
}

func (s *puzzleRushService) StartRush(ctx context.Context, profileID int64, difficulty string) (*models.PuzzleRushSession, error) {
	log := logger.FromContext(ctx)
	log.Debug("starting puzzle rush: profile_id=%d, difficulty=%s", profileID, difficulty)

	// Validate difficulty
	mistakesAllowed := 0
	switch difficulty {
	case "easy":
		mistakesAllowed = 5
	case "medium":
		mistakesAllowed = 3
	case "hard":
		mistakesAllowed = 1
	default:
		return nil, errors.NewValidationError("difficulty", "must be 'easy', 'medium', or 'hard'")
	}

	// Check for active session
	activeSession, err := s.rushRepo.GetActiveSession(ctx, profileID)
	if err != nil {
		log.Error("failed to check for active session: %v", err)
		return nil, errors.NewInternalError(err)
	}
	if activeSession != nil {
		return nil, errors.NewValidationError("session", "an active puzzle rush session already exists")
	}

	// Create new session
	session := models.PuzzleRushSession{
		ProfileID:        profileID,
		Difficulty:       difficulty,
		Score:            0,
		MistakesMade:     0,
		MistakesAllowed:  mistakesAllowed,
		TotalTimeSeconds: 0,
		CreatedAt:        time.Now(),
	}

	sessionID, err := s.rushRepo.InsertSession(ctx, session)
	if err != nil {
		log.Error("failed to create puzzle rush session: %v", err)
		return nil, errors.NewInternalError(err)
	}

	session.ID = sessionID
	log.Info("puzzle rush session started: id=%d, difficulty=%s", sessionID, difficulty)
	return &session, nil
}

func (s *puzzleRushService) GetCurrentSession(ctx context.Context, profileID int64) (*models.PuzzleRushSession, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting current puzzle rush session: profile_id=%d", profileID)

	session, err := s.rushRepo.GetActiveSession(ctx, profileID)
	if err != nil {
		log.Error("failed to get active session: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return session, nil
}

func (s *puzzleRushService) SubmitAnswer(ctx context.Context, sessionID int64, profileID int64, flashcardID int64, quality int, timeSeconds float64) (*models.PuzzleRushSession, error) {
	log := logger.FromContext(ctx)
	log.Debug("submitting puzzle rush answer: session_id=%d, flashcard_id=%d, quality=%d", sessionID, flashcardID, quality)

	// Validate quality
	if quality < 0 || quality > 5 {
		return nil, errors.NewValidationError("quality", "must be between 0 and 5")
	}

	// Get session and verify ownership
	session, err := s.rushRepo.GetSession(ctx, sessionID)
	if err != nil {
		log.Error("failed to get session: %v", err)
		return nil, errors.NewInternalError(err)
	}
	if session == nil {
		return nil, errors.NewNotFoundError("puzzle rush session", sessionID)
	}
	if session.ProfileID != profileID {
		return nil, errors.NewValidationError("session", "session does not belong to profile")
	}
	if session.CompletedAt != nil {
		return nil, errors.NewValidationError("session", "session is already completed")
	}

	// Determine if answer is correct (quality 3-5 = correct, 0-2 = mistake)
	wasCorrect := quality >= 3

	// Get attempt number (count existing attempts + 1)
	attempts, err := s.rushRepo.GetSessionAttempts(ctx, sessionID)
	if err != nil {
		log.Error("failed to get session attempts: %v", err)
		return nil, errors.NewInternalError(err)
	}
	attemptNumber := len(attempts) + 1

	// Record attempt
	attempt := models.PuzzleRushAttempt{
		SessionID:     sessionID,
		FlashcardID:   flashcardID,
		WasCorrect:    wasCorrect,
		TimeSeconds:   timeSeconds,
		AttemptNumber: attemptNumber,
		CreatedAt:     time.Now(),
	}
	_, err = s.rushRepo.InsertAttempt(ctx, attempt)
	if err != nil {
		log.Error("failed to insert attempt: %v", err)
		return nil, errors.NewInternalError(err)
	}

	// Update session
	if wasCorrect {
		session.Score++
	} else {
		session.MistakesMade++
	}
	session.TotalTimeSeconds += timeSeconds

	// Check if session should end
	now := time.Now()
	if session.MistakesMade >= session.MistakesAllowed {
		session.CompletedAt = &now
		log.Info("puzzle rush session completed: id=%d, score=%d, mistakes=%d", sessionID, session.Score, session.MistakesMade)
	}

	// Update flashcard using flashcard service (applies spaced repetition)
	if err := s.flashcardSvc.ReviewFlashcard(ctx, flashcardID, profileID, quality, timeSeconds); err != nil {
		log.Warn("failed to review flashcard: %v", err)
		// Continue even if flashcard review fails
	}

	// Update session in database
	if err := s.rushRepo.UpdateSession(ctx, *session); err != nil {
		log.Error("failed to update session: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return session, nil
}

func (s *puzzleRushService) EndRush(ctx context.Context, sessionID int64, profileID int64) error {
	log := logger.FromContext(ctx)
	log.Debug("ending puzzle rush session: session_id=%d", sessionID)

	// Get session and verify ownership
	session, err := s.rushRepo.GetSession(ctx, sessionID)
	if err != nil {
		log.Error("failed to get session: %v", err)
		return errors.NewInternalError(err)
	}
	if session == nil {
		return errors.NewNotFoundError("puzzle rush session", sessionID)
	}
	if session.ProfileID != profileID {
		return errors.NewValidationError("session", "session does not belong to profile")
	}
	if session.CompletedAt != nil {
		return nil // Already completed
	}

	// Mark as completed
	now := time.Now()
	session.CompletedAt = &now
	if err := s.rushRepo.UpdateSession(ctx, *session); err != nil {
		log.Error("failed to update session: %v", err)
		return errors.NewInternalError(err)
	}

	log.Info("puzzle rush session ended: id=%d, score=%d", sessionID, session.Score)
	return nil
}

func (s *puzzleRushService) GetStats(ctx context.Context, profileID int64) (*models.PuzzleRushStats, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting puzzle rush stats: profile_id=%d", profileID)

	stats, err := s.rushRepo.GetUserStats(ctx, profileID)
	if err != nil {
		log.Error("failed to get stats: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return stats, nil
}

func (s *puzzleRushService) GetBestScores(ctx context.Context, profileID int64) ([]models.PuzzleRushBestScore, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting puzzle rush best scores: profile_id=%d", profileID)

	bestScores, err := s.rushRepo.GetBestScores(ctx, profileID)
	if err != nil {
		log.Error("failed to get best scores: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return bestScores, nil
}
