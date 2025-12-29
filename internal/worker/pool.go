package worker

import (
	"context"
	"sync"
	"time"

	"github.com/vytor/chessflash/internal/logger"
)

type Job interface {
	Run(context.Context) error
	Name() string
}

type Pool struct {
	jobs     chan Job
	wg       sync.WaitGroup
	workers  int
	queue    int
	cancel   context.CancelFunc
	mu       sync.Mutex
	stopped  bool
	log      *logger.Logger
}

func NewPool(workers, queueSize int) *Pool {
	if workers <= 0 {
		workers = 2
	}
	if queueSize <= 0 {
		queueSize = 64
	}
	log := logger.Default().WithPrefix("worker-pool")
	log.Debug("creating worker pool with %d workers and queue size %d", workers, queueSize)
	return &Pool{
		jobs:    make(chan Job, queueSize),
		workers: workers,
		queue:   queueSize,
		log:     log,
	}
}

func (p *Pool) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		// Already started, wait for workers to finish if they're stopping
		p.wg.Wait()
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.stopped = false
	p.log.Info("starting worker pool with %d workers", p.workers)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			workerLog := p.log.WithField("worker_id", id)
			workerLog.Debug("worker started")

			for {
				select {
				case <-ctx.Done():
					workerLog.Debug("worker shutting down (context cancelled)")
					return
				case job := <-p.jobs:
					if job == nil {
						workerLog.Debug("worker shutting down (nil job received)")
						return
					}

					jobLog := workerLog.WithField("job", job.Name())
					jobLog.Debug("starting job")
					start := time.Now()

					// Create a context with the logger for the job
					jobCtx := logger.NewContext(ctx, jobLog)

					if err := job.Run(jobCtx); err != nil {
						jobLog.Error("job failed after %v: %v", time.Since(start), err)
					} else {
						jobLog.Info("job completed in %v", time.Since(start))
					}
				}
			}
		}(i + 1)
	}
}

func (p *Pool) Stop() {
	p.log.Info("stopping worker pool")
	if p.cancel != nil {
		p.cancel()
	}
	close(p.jobs)
	p.wg.Wait()
	p.log.Info("worker pool stopped")
}

// Cancel cancels the context, stopping new jobs from being accepted and cancelling current jobs.
// This is a non-blocking operation. Workers will exit when their current jobs complete or are cancelled.
func (p *Pool) Cancel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log.Info("cancelling worker pool")
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
		p.stopped = true
	}
}

// Restart restarts the pool with a new context. It waits for any existing workers to finish first.
func (p *Pool) Restart(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Wait for existing workers to finish if they're still running
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.wg.Wait()

	// Start with new context
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.stopped = false
	p.log.Info("restarting worker pool with %d workers", p.workers)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			workerLog := p.log.WithField("worker_id", id)
			workerLog.Debug("worker started")

			for {
				select {
				case <-ctx.Done():
					workerLog.Debug("worker shutting down (context cancelled)")
					return
				case job := <-p.jobs:
					if job == nil {
						workerLog.Debug("worker shutting down (nil job received)")
						return
					}

					jobLog := workerLog.WithField("job", job.Name())
					jobLog.Debug("starting job")
					start := time.Now()

					// Create a context with the logger for the job
					jobCtx := logger.NewContext(ctx, jobLog)

					if err := job.Run(jobCtx); err != nil {
						jobLog.Error("job failed after %v: %v", time.Since(start), err)
					} else {
						jobLog.Info("job completed in %v", time.Since(start))
					}
				}
			}
		}(i + 1)
	}
}

func (p *Pool) Submit(job Job) {
	p.log.Debug("submitting job: %s", job.Name())
	p.jobs <- job
}

// QueueSize returns the current number of pending jobs.
func (p *Pool) QueueSize() int {
	return len(p.jobs)
}
