package services

import (
	"context"

	"github.com/vytor/chessflash/internal/jobs"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

// ImportService handles game import business logic
type ImportService interface {
	ImportGames(ctx context.Context, profile models.Profile)
}

type importService struct {
	jobQueue jobs.JobQueue
}

// NewImportService creates a new ImportService
func NewImportService(jobQueue jobs.JobQueue) ImportService {
	return &importService{jobQueue: jobQueue}
}

func (s *importService) ImportGames(ctx context.Context, profile models.Profile) {
	log := logger.FromContext(ctx)
	log = log.WithFields(map[string]any{
		"username":   profile.Username,
		"profile_id": profile.ID,
	})
	log.Info("queueing game import job")

	// The actual import logic is in the worker job
	// This service just orchestrates the job submission
	if err := s.jobQueue.EnqueueImport(profile.ID, profile.Username); err != nil {
		log.Error("failed to enqueue import job: %v", err)
	}
}
