package analysis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vytor/chessflash/internal/logger"
)

type EvalResult struct {
	BestMove string
	CP       float64 // centipawns from white perspective (only when Mate is nil)
	Mate     *int    // mate in N (positive = white mates in N, negative = black mates in N)
}

type Engine struct {
	path string
	log  *logger.Logger

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  ioWriter
	stdout *bufio.Reader
}

type ioWriter interface {
	Write([]byte) (int, error)
}

func NewEngine(path string) (*Engine, error) {
	log := logger.Default().WithPrefix("stockfish")

	if path == "" {
		path = "stockfish"
	}

	log.Info("starting stockfish engine: %s", path)
	cmd := exec.Command(path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Error("failed to create stdin pipe: %v", err)
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("failed to create stdout pipe: %v", err)
		return nil, err
	}

	engine := &Engine{
		path:   path,
		log:    log,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	if err := cmd.Start(); err != nil {
		log.Error("failed to start stockfish: %v", err)
		return nil, err
	}

	log.Debug("initializing UCI protocol")
	if err := engine.init(); err != nil {
		log.Error("failed to initialize UCI: %v", err)
		return nil, err
	}

	log.Info("stockfish engine ready")
	return engine, nil
}

func (e *Engine) init() error {
	if err := e.send("uci"); err != nil {
		return err
	}
	if err := e.waitFor("uciok", 2*time.Second); err != nil {
		return err
	}
	if err := e.send("isready"); err != nil {
		return err
	}
	return e.waitFor("readyok", 2*time.Second)
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cmd == nil {
		return nil
	}

	e.log.Debug("closing stockfish engine")
	_ = e.sendLocked("quit")
	err := e.cmd.Wait()
	e.cmd = nil

	if err != nil {
		e.log.Debug("stockfish process exited: %v", err)
	} else {
		e.log.Debug("stockfish process exited cleanly")
	}

	return err
}

func (e *Engine) EvaluateFEN(ctx context.Context, fen string, depth int, maxTimeMs int) (EvalResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	log := e.log.WithFields(map[string]any{
		"depth":      depth,
		"max_time_ms": maxTimeMs,
	})

	if depth == 0 {
		depth = 18
	}

	start := time.Now()
	log.Debug("evaluating position")

	if err := e.sendLocked("ucinewgame"); err != nil {
		log.Error("failed to send ucinewgame: %v", err)
		return EvalResult{}, err
	}
	if err := e.sendLocked("position fen " + fen); err != nil {
		log.Error("failed to set position: %v", err)
		return EvalResult{}, err
	}

	// Parse FEN to determine whose turn it is
	parts := strings.Fields(fen)
	isBlackToMove := len(parts) > 1 && parts[1] == "b"

	// Build go command with optional movetime
	var goCmd string
	if maxTimeMs > 0 {
		goCmd = fmt.Sprintf("go depth %d movetime %d", depth, maxTimeMs)
	} else {
		goCmd = fmt.Sprintf("go depth %d", depth)
	}

	if err := e.sendLocked(goCmd); err != nil {
		log.Error("failed to start analysis: %v", err)
		return EvalResult{}, err
	}

	var best EvalResult
	// Use maxTimeMs + buffer for deadline, or default 8s if no limit
	deadlineDuration := 8 * time.Second
	if maxTimeMs > 0 {
		deadlineDuration = time.Duration(maxTimeMs)*time.Millisecond + 500*time.Millisecond // Add 500ms buffer
	}
	deadline := time.Now().Add(deadlineDuration)
	for {
		if ctx.Err() != nil {
			log.Warn("evaluation cancelled: %v", ctx.Err())
			return EvalResult{}, ctx.Err()
		}
		if time.Now().After(deadline) {
			log.Error("evaluation timed out after %v", deadlineDuration)
			return EvalResult{}, errors.New("stockfish timeout")
		}
		line, err := e.stdout.ReadString('\n')
		if err != nil {
			log.Error("failed to read from stockfish: %v", err)
			return EvalResult{}, err
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "info") {
			if cp, mate, ok := parseScore(line); ok {
				if mate != nil {
					best.Mate = mate
					// Normalize mate to white's perspective by sign:
					// mate > 0 -> side to move mates; mate < 0 -> side to move gets mated
					// If black to move, flip sign to white perspective.
					mateVal := *mate
					if isBlackToMove {
						mateVal = -mateVal
					}
					best.Mate = &mateVal
					// For mate, CP is not meaningful; keep CP at 0
					best.CP = 0
				} else {
					// Normalize centipawns to white's perspective
					if isBlackToMove {
						best.CP = -cp
					} else {
						best.CP = cp
					}
					best.Mate = nil
				}
			}
		}
		if strings.HasPrefix(line, "bestmove") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				best.BestMove = parts[1]
			}
			if best.Mate != nil {
				log.Debug("evaluation completed in %v: mate=%d, bestmove=%s", time.Since(start), *best.Mate, best.BestMove)
			} else {
				log.Debug("evaluation completed in %v: cp=%.0f, bestmove=%s", time.Since(start), best.CP, best.BestMove)
			}
			return best, nil
		}
	}
}

// parseScore returns cp, mate, ok.
// mate: nil if not mate; non-nil value is mate in N (positive: side to move mates in N, negative: side to move gets mated in N).
func parseScore(line string) (float64, *int, bool) {
	parts := strings.Fields(line)
	for i := 0; i < len(parts); i++ {
		if parts[i] == "score" && i+2 < len(parts) {
			if parts[i+1] == "cp" {
				if v, err := strconv.Atoi(parts[i+2]); err == nil {
					return float64(v), nil, true
				}
			} else if parts[i+1] == "mate" {
				// Mate score: positive means the side to move can force mate
				// Negative means the side to move is getting mated
				if mateMoves, err := strconv.Atoi(parts[i+2]); err == nil {
					return 0, &mateMoves, true
				}
			}
		}
	}
	return 0, nil, false
}

func (e *Engine) send(cmd string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sendLocked(cmd)
}

func (e *Engine) sendLocked(cmd string) error {
	_, err := e.stdin.Write([]byte(cmd + "\n"))
	return err
}

func (e *Engine) waitFor(marker string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			e.log.Error("timeout waiting for %s", marker)
			return fmt.Errorf("timeout waiting for %s", marker)
		}
		line, err := e.stdout.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.Contains(line, marker) {
			return nil
		}
	}
}
