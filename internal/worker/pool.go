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
	jobs    chan Job
	wg      sync.WaitGroup
	workers int
	queue   int
	cancel  context.CancelFunc
	log     *logger.Logger
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
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
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

func (p *Pool) Submit(job Job) {
	p.log.Debug("submitting job: %s", job.Name())
	p.jobs <- job
}

// QueueSize returns the current number of pending jobs.
func (p *Pool) QueueSize() int {
	return len(p.jobs)
}
