package worker

import (
	"context"
	"errors"
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
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
	
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

// ClearQueue drains all pending jobs from the queue channel.
// This should be called after Cancel() to ensure the queue is empty.
// It's safe to call even if the pool is stopped.
func (p *Pool) ClearQueue() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Drain the channel
	drained := 0
	for {
		select {
		case <-p.jobs:
			drained++
		default:
			// Channel is empty
			if drained > 0 {
				p.log.Info("cleared %d jobs from queue", drained)
			}
			return
		}
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

var ErrPoolStopped = errors.New("worker pool is stopped")

func (p *Pool) Submit(job Job) error {
	p.mu.Lock()
	stopped := p.stopped
	p.mu.Unlock()
	
	if stopped {
		p.log.Debug("rejected job submission (pool stopped): %s", job.Name())
		return ErrPoolStopped
	}
	
	// Use recover to catch panic if channel is closed
	var submitErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel was closed, which means pool was stopped
				p.log.Debug("recovered from panic during job submission (channel closed): %s", job.Name())
				submitErr = ErrPoolStopped
			}
		}()
		
		select {
		case p.jobs <- job:
			p.log.Debug("submitted job: %s", job.Name())
			submitErr = nil
		default:
			// Channel is full, check if pool was stopped while we were waiting
			p.mu.Lock()
			stopped = p.stopped
			p.mu.Unlock()
			if stopped {
				p.log.Debug("rejected job submission (pool stopped): %s", job.Name())
				submitErr = ErrPoolStopped
			} else {
				// Channel is full, but pool is still running - this shouldn't happen with buffered channel
				// but we'll handle it gracefully
				p.log.Warn("job queue full, rejecting job: %s", job.Name())
				submitErr = errors.New("job queue is full")
			}
		}
	}()
	
	return submitErr
}

// QueueSize returns the current number of pending jobs.
func (p *Pool) QueueSize() int {
	return len(p.jobs)
}

// WorkerCount returns the number of workers in the pool.
func (p *Pool) WorkerCount() int {
	return p.workers
}

// IsRunning returns true if the pool is currently running (not stopped).
func (p *Pool) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.stopped && p.cancel != nil
}
