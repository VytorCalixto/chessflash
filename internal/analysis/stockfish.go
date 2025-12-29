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
	CP       float64 // centipawns from white perspective
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

func (e *Engine) EvaluateFEN(ctx context.Context, fen string, depth int) (EvalResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	log := e.log.WithFields(map[string]any{
		"depth": depth,
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

	if err := e.sendLocked(fmt.Sprintf("go depth %d", depth)); err != nil {
		log.Error("failed to start analysis: %v", err)
		return EvalResult{}, err
	}

	var best EvalResult
	var isMate bool
	deadline := time.Now().Add(8 * time.Second)
	for {
		if ctx.Err() != nil {
			log.Warn("evaluation cancelled: %v", ctx.Err())
			return EvalResult{}, ctx.Err()
		}
		if time.Now().After(deadline) {
			log.Error("evaluation timed out after 8s")
			return EvalResult{}, errors.New("stockfish timeout")
		}
		line, err := e.stdout.ReadString('\n')
		if err != nil {
			log.Error("failed to read from stockfish: %v", err)
			return EvalResult{}, err
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "info") {
			if cp, ok, mate := parseScore(line); ok {
				if mate {
					isMate = true
					// Mate score: positive means the side to move can force mate
					// Convert to white's perspective
					if isBlackToMove {
						// Black to move, mate means black wins → negative from white's perspective
						best.CP = -cp
					} else {
						// White to move, mate means white wins → positive from white's perspective
						best.CP = cp
					}
				} else {
					// Normalize to white's perspective
					if isBlackToMove {
						best.CP = -cp
					} else {
						best.CP = cp
					}
				}
			}
		}
		if strings.HasPrefix(line, "bestmove") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				best.BestMove = parts[1]
			}
			if isMate {
				log.Debug("evaluation completed in %v: mate, bestmove=%s", time.Since(start), best.BestMove)
			} else {
				log.Debug("evaluation completed in %v: cp=%.0f, bestmove=%s", time.Since(start), best.CP, best.BestMove)
			}
			return best, nil
		}
	}
}

func parseScore(line string) (float64, bool, bool) {
	parts := strings.Fields(line)
	for i := 0; i < len(parts); i++ {
		if parts[i] == "score" && i+2 < len(parts) {
			if parts[i+1] == "cp" {
				if v, err := strconv.Atoi(parts[i+2]); err == nil {
					return float64(v), true, false
				}
			} else if parts[i+1] == "mate" {
				// Mate score: positive means the side to move can force mate
				// Convert to centipawns: use 10000 - (mateMoves * 10)
				// This gives mate in 1 = 9990, mate in 2 = 9980, etc.
				if mateMoves, err := strconv.Atoi(parts[i+2]); err == nil {
					cpValue := 10000.0 - float64(mateMoves)*10.0
					return cpValue, true, true
				}
			}
		}
	}
	return 0, false, false
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
