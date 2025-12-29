package jobs

import (
	"context"

	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/repository"
	"github.com/vytor/chessflash/internal/worker"
)

// WorkerQueue implements JobQueue using worker pools
type WorkerQueue struct {
	analysisPool    *worker.Pool
	importPool      *worker.Pool
	db              *db.DB
	profileRepo     repository.ProfileRepository
	analysisService worker.AnalysisServiceInterface
	chessClient     *chesscom.Client
	stockfishPath   string
	stockfishDepth  int
	archiveLimit    int
	maxConcurrent   int
}

// NewWorkerQueue creates a new WorkerQueue implementation
func NewWorkerQueue(
	analysisPool *worker.Pool,
	importPool *worker.Pool,
	db *db.DB,
	profileRepo repository.ProfileRepository,
	analysisService worker.AnalysisServiceInterface,
	chessClient *chesscom.Client,
	stockfishPath string,
	stockfishDepth int,
	archiveLimit int,
	maxConcurrent int,
) JobQueue {
	return &WorkerQueue{
		analysisPool:    analysisPool,
		importPool:      importPool,
		db:              db,
		profileRepo:     profileRepo,
		analysisService: analysisService,
		chessClient:     chessClient,
		stockfishPath:   stockfishPath,
		stockfishDepth:  stockfishDepth,
		archiveLimit:    archiveLimit,
		maxConcurrent:   maxConcurrent,
	}
}

func (q *WorkerQueue) EnqueueAnalysis(gameID int64) error {
	err := q.analysisPool.Submit(&worker.AnalyzeGameJob{
		AnalysisService: q.analysisService,
		GameID:          gameID,
	})
	return err
}

func (q *WorkerQueue) EnqueueImport(profileID int64, username string) error {
	// Get profile from repository
	ctx := context.Background()
	profile, err := q.profileRepo.Get(ctx, profileID)
	if err != nil || profile == nil {
		// If profile not found, try to upsert it
		profile, err = q.profileRepo.Upsert(ctx, username)
		if err != nil {
			return err
		}
	}
	
	err = q.importPool.Submit(&worker.ImportGamesJob{
		DB:             q.db,
		ChessClient:    q.chessClient,
		Profile:        *profile,
		AnalysisPool:   q.analysisPool,
		StockfishPath:  q.stockfishPath,
		StockfishDepth: q.stockfishDepth,
		ArchiveLimit:   q.archiveLimit,
		MaxConcurrent:  q.maxConcurrent,
	})
	return err
}
