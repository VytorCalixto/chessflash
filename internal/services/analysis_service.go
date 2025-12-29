package services

import (
	"context"
	"time"

	"github.com/vytor/chessflash/internal/analysis"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

// AnalysisService handles position analysis business logic
type AnalysisService interface {
	EvaluatePosition(ctx context.Context, fen string, stockfishPath string, stockfishDepth int) (analysis.EvalResult, error)
}

type analysisService struct{}

// NewAnalysisService creates a new AnalysisService
func NewAnalysisService() AnalysisService {
	return &analysisService{}
}

func (s *analysisService) EvaluatePosition(ctx context.Context, fen string, stockfishPath string, stockfishDepth int) (analysis.EvalResult, error) {
	log := logger.FromContext(ctx)
	log.Debug("evaluating position: fen=%s, depth=%d", fen, stockfishDepth)

	if fen == "" {
		return analysis.EvalResult{}, errors.NewValidationError("fen", "cannot be empty")
	}

	// Use a lighter depth for real-time evaluation (faster response)
	depth := 15
	if stockfishDepth > 0 && stockfishDepth < 15 {
		depth = stockfishDepth
	}

	// Create a new Stockfish engine for this request
	// Note: In production, you might want to use a pool of engines
	sf, err := analysis.NewEngine(stockfishPath)
	if err != nil {
		log.Error("failed to initialize stockfish: %v", err)
		return analysis.EvalResult{}, errors.NewInternalError(err)
	}
	defer sf.Close()

	// Set timeout for evaluation
	evalCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := sf.EvaluateFEN(evalCtx, fen, depth)
	if err != nil {
		log.Error("failed to evaluate position: %v", err)
		return analysis.EvalResult{}, errors.NewInternalError(err)
	}

	return result, nil
}
