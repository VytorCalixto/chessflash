package analysis

import (
	"context"
	"sync"

	"github.com/vytor/chessflash/internal/logger"
)

// EnginePool manages a pool of reusable Stockfish engines.
type EnginePool struct {
	path    string
	size    int
	engines chan *Engine
	mu      sync.Mutex
	closed  bool
	log     *logger.Logger
}

// NewEnginePool creates a pool with the specified number of engines.
func NewEnginePool(path string, size int) (*EnginePool, error) {
	if size <= 0 {
		size = 2
	}
	log := logger.Default().WithPrefix("stockfish-pool")

	pool := &EnginePool{
		path:    path,
		size:    size,
		engines: make(chan *Engine, size),
		log:     log,
	}

	// Pre-warm the pool
	log.Info("initializing engine pool with %d engines", size)
	for i := 0; i < size; i++ {
		engine, err := NewEngine(path)
		if err != nil {
			pool.Close() // Clean up any already-created engines
			return nil, err
		}
		pool.engines <- engine
	}
	log.Info("engine pool ready")
	return pool, nil
}

// Acquire gets an engine from the pool, blocking if none are available.
func (p *EnginePool) Acquire(ctx context.Context) (*Engine, error) {
	select {
	case engine := <-p.engines:
		return engine, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns an engine to the pool.
func (p *EnginePool) Release(engine *Engine) {
	if engine == nil {
		return
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		// Pool is closed, close the engine
		engine.Close()
		return
	}
	select {
	case p.engines <- engine:
		// Returned to pool
	default:
		// Pool full, close the engine
		engine.Close()
	}
}

// Evaluate acquires an engine, evaluates, and releases it back.
func (p *EnginePool) Evaluate(ctx context.Context, fen string, depth int) (EvalResult, error) {
	engine, err := p.Acquire(ctx)
	if err != nil {
		return EvalResult{}, err
	}
	defer p.Release(engine)

	return engine.EvaluateFEN(ctx, fen, depth)
}

// Close shuts down all engines in the pool.
func (p *EnginePool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	p.log.Info("closing engine pool")
	close(p.engines)
	for engine := range p.engines {
		engine.Close()
	}
}

// Available returns how many engines are currently idle.
func (p *EnginePool) Available() int {
	return len(p.engines)
}
