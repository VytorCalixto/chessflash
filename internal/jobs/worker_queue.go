package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
	"github.com/vytor/chessflash/internal/worker"
)

// WorkerQueue implements JobQueue using worker pools
type WorkerQueue struct {
	analysisPool    *worker.Pool
	importPool      *worker.Pool
	profileRepo     repository.ProfileRepository
	gameRepo        repository.GameRepository
	statsRepo       repository.StatsRepository
	analysisService worker.AnalysisServiceInterface
	chessClient     *chesscom.Client
	stockfishPath   string
	stockfishDepth  int
	archiveLimit    int
	maxConcurrent   int

	// Backfill mechanism
	backfillMu      sync.Mutex
	backfillFilter  *models.AnalysisFilter
	backfillRunning bool
	backfillCancel  context.CancelFunc
	backfillWg      sync.WaitGroup
}

// NewWorkerQueue creates a new WorkerQueue implementation
func NewWorkerQueue(
	analysisPool *worker.Pool,
	importPool *worker.Pool,
	profileRepo repository.ProfileRepository,
	gameRepo repository.GameRepository,
	statsRepo repository.StatsRepository,
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
		profileRepo:     profileRepo,
		gameRepo:        gameRepo,
		statsRepo:       statsRepo,
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

	// If queue is full, start backfill if not already running
	if err != nil && err.Error() == "job queue is full" {
		q.startBackfillIfNeeded()
	}

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
		GameRepo:       q.gameRepo,
		ProfileRepo:    q.profileRepo,
		StatsRepo:      q.statsRepo,
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

// StartBackfill starts the automatic backfill process for the given filter
func (q *WorkerQueue) StartBackfill(filter models.AnalysisFilter) {
	q.backfillMu.Lock()
	defer q.backfillMu.Unlock()

	if q.backfillRunning {
		// Update filter if backfill is already running
		q.backfillFilter = &filter
		return
	}

	q.backfillFilter = &filter
	q.backfillRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	ctx = logger.NewContext(ctx, logger.Default().WithPrefix("backfill"))
	q.backfillCancel = cancel

	q.backfillWg.Add(1)
	go q.backfillLoop(ctx)

	logger.FromContext(ctx).Info("started automatic backfill for analysis queue")
}

// StopBackfill stops the automatic backfill process
func (q *WorkerQueue) StopBackfill() {
	q.backfillMu.Lock()
	defer q.backfillMu.Unlock()

	if !q.backfillRunning {
		return
	}

	if q.backfillCancel != nil {
		q.backfillCancel()
	}
	q.backfillWg.Wait()
	q.backfillRunning = false
	q.backfillFilter = nil

	logger.Default().WithPrefix("backfill").Info("stopped automatic backfill")
}

func (q *WorkerQueue) startBackfillIfNeeded() {
	q.backfillMu.Lock()
	defer q.backfillMu.Unlock()

	// Only start if we have a filter and backfill isn't running
	if q.backfillFilter != nil && !q.backfillRunning {
		q.backfillRunning = true
		ctx, cancel := context.WithCancel(context.Background())
		ctx = logger.NewContext(ctx, logger.Default().WithPrefix("backfill"))
		q.backfillCancel = cancel

		q.backfillWg.Add(1)
		go q.backfillLoop(ctx)

		logger.FromContext(ctx).Info("started automatic backfill (queue was full)")
	}
}

func (q *WorkerQueue) backfillLoop(ctx context.Context) {
	defer q.backfillWg.Done()
	log := logger.FromContext(ctx)

	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("backfill loop stopped")
			return
		case <-ticker.C:
			// Only proceed if pool is running
			if !q.analysisPool.IsRunning() {
				continue
			}

			// Check if queue has space
			queueSize := q.analysisPool.QueueSize()
			queueCapacity := q.analysisPool.QueueCapacity()
			// Use a threshold - start backfilling when queue is less than 80% full
			if queueSize >= int(float64(queueCapacity)*0.8) {
				continue
			}

			q.backfillMu.Lock()
			filter := q.backfillFilter
			q.backfillMu.Unlock()

			if filter == nil {
				continue
			}

			// Query for pending games matching the filter
			games, err := q.gameRepo.GamesForAnalysis(ctx, *filter)
			if err != nil {
				log.Warn("failed to query games for backfill: %v", err)
				continue
			}

			if len(games) == 0 {
				// No more games to process, stop backfill
				log.Info("no more games to backfill, stopping backfill loop")
				q.backfillMu.Lock()
				q.backfillRunning = false
				q.backfillCancel = nil
				q.backfillMu.Unlock()
				return
			}

			// Try to enqueue games until queue is full or we run out of games
			enqueued := 0
			for _, g := range games {
				// Check if pool is still running
				if !q.analysisPool.IsRunning() {
					break
				}

				// Check if queue has space (recalculate on each iteration)
				currentQueueSize := q.analysisPool.QueueSize()
				if currentQueueSize >= int(float64(queueCapacity)*0.9) {
					break
				}

				if err := q.analysisPool.Submit(&worker.AnalyzeGameJob{
					AnalysisService: q.analysisService,
					GameID:          g.ID,
				}); err != nil {
					if err.Error() == "job queue is full" {
						// Queue filled up, break and wait for next tick
						break
					}
					// Other errors (like pool stopped) - stop backfill
					if err == worker.ErrPoolStopped {
						log.Info("pool stopped, stopping backfill")
						q.backfillMu.Lock()
						q.backfillRunning = false
						q.backfillCancel = nil
						q.backfillMu.Unlock()
						return
					}
					log.Warn("failed to enqueue game %d: %v", g.ID, err)
					continue
				}
				enqueued++
			}

			if enqueued > 0 {
				log.Debug("backfilled %d games into queue", enqueued)
			}
		}
	}
}
