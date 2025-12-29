package services

import (
	"context"

	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/worker"
)

// ImportService handles game import business logic
type ImportService interface {
	ImportGames(ctx context.Context, profile models.Profile, importPool *worker.Pool, analysisPool *worker.Pool, chessClient *chesscom.Client, stockfishPath string, stockfishDepth int, archiveLimit int, maxConcurrent int)
}

type importService struct {
	db *db.DB
}

// NewImportService creates a new ImportService
func NewImportService(db *db.DB) ImportService {
	return &importService{db: db}
}

func (s *importService) ImportGames(ctx context.Context, profile models.Profile, importPool *worker.Pool, analysisPool *worker.Pool, chessClient *chesscom.Client, stockfishPath string, stockfishDepth int, archiveLimit int, maxConcurrent int) {
	log := logger.FromContext(ctx)
	log = log.WithFields(map[string]any{
		"username":   profile.Username,
		"profile_id": profile.ID,
	})
	log.Info("queueing game import job")

	// The actual import logic is in the worker job
	// This service just orchestrates the job submission
	job := &worker.ImportGamesJob{
		DB:             s.db,
		ChessClient:    chessClient,
		Profile:        profile,
		AnalysisPool:   analysisPool,
		StockfishPath:  stockfishPath,
		StockfishDepth: stockfishDepth,
		ArchiveLimit:   archiveLimit,
		MaxConcurrent:  maxConcurrent,
	}
	importPool.Submit(job)
}
