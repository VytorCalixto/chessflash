package services

import (
	"context"
	"strings"
	"time"

	"github.com/corentings/chess/v2"
	"github.com/corentings/chess/v2/opening"
	"github.com/vytor/chessflash/internal/analysis"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

// AnalysisService handles position analysis business logic
type AnalysisService interface {
	EvaluatePosition(ctx context.Context, fen string, stockfishPath string, stockfishDepth int) (analysis.EvalResult, error)
	AnalyzeGame(ctx context.Context, gameID int64) error
}

type analysisService struct {
	gameRepo       repository.GameRepository
	positionRepo   repository.PositionRepository
	flashcardRepo  repository.FlashcardRepository
	statsRepo      repository.StatsRepository
	config         AnalysisConfig
	pool           *analysis.EnginePool
}

// NewAnalysisService creates a new AnalysisService
func NewAnalysisService(
	gameRepo repository.GameRepository,
	positionRepo repository.PositionRepository,
	flashcardRepo repository.FlashcardRepository,
	statsRepo repository.StatsRepository,
	config AnalysisConfig,
	pool *analysis.EnginePool,
) AnalysisService {
	return &analysisService{
		gameRepo:      gameRepo,
		positionRepo:  positionRepo,
		flashcardRepo: flashcardRepo,
		statsRepo:     statsRepo,
		config:        config,
		pool:          pool,
	}
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

	// Use shorter time for real-time evaluation
	maxTimeMs := 2000 // 2 seconds for real-time
	if s.config.StockfishMaxTime > 0 && s.config.StockfishMaxTime < maxTimeMs {
		maxTimeMs = s.config.StockfishMaxTime
	}

	// Set timeout for evaluation
	evalCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := s.pool.Evaluate(evalCtx, fen, depth, maxTimeMs)
	if err != nil {
		log.Error("failed to evaluate position: %v", err)
		return analysis.EvalResult{}, errors.NewInternalError(err)
	}

	return result, nil
}

func (s *analysisService) AnalyzeGame(ctx context.Context, gameID int64) error {
	log := logger.FromContext(ctx).WithField("game_id", gameID)
	log.Info("starting game analysis")

	game, err := s.gameRepo.Get(ctx, gameID)
	if err != nil {
		log.Error("failed to get game: %v", err)
		return err
	}

	if game == nil {
		return errors.NewNotFoundError("game", gameID)
	}

	if game.AnalysisStatus == "completed" {
		log.Debug("game already analyzed, skipping")
		return nil
	}

	log = log.WithFields(map[string]any{
		"opponent":   game.Opponent,
		"time_class": game.TimeClass,
		"result":     game.Result,
	})

	log.Debug("updating game status to processing")
	if err := s.gameRepo.UpdateStatus(ctx, gameID, "processing"); err != nil {
		log.Error("failed to update game status: %v", err)
		return err
	}

	log.Debug("acquiring stockfish engine from pool")
	engine, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Error("failed to acquire engine from pool: %v", err)
		_ = s.gameRepo.UpdateStatus(ctx, gameID, "failed")
		return err
	}
	defer s.pool.Release(engine)

	depth := s.config.StockfishDepth
	if depth <= 0 {
		depth = 18
	}
	maxTimeMs := s.config.StockfishMaxTime
	log = log.WithFields(map[string]any{
		"depth":       depth,
		"max_time_ms": maxTimeMs,
	})

	log.Debug("parsing PGN")
	pgnOpt, err := chess.PGN(strings.NewReader(game.PGN))
	if err != nil {
		log.Error("failed to parse PGN: %v", err)
		_ = s.gameRepo.UpdateStatus(ctx, gameID, "failed")
		return err
	}
	chessGame := chess.NewGame(pgnOpt)

	// Detect opening if missing
	if game.OpeningName == "" {
		book := opening.NewBookECO()
		foundOpening := book.Find(chessGame.Moves())
		if foundOpening != nil {
			game.ECOCode = foundOpening.Code()
			game.OpeningName = foundOpening.Title()
			if err := s.gameRepo.UpdateOpening(ctx, game.ID, game.ECOCode, game.OpeningName); err != nil {
				log.Warn("failed to update game opening: %v", err)
			} else {
				log.Debug("updated opening to %s (%s)", game.OpeningName, game.ECOCode)
			}
		}
	}

	positions := chessGame.Positions()
	moves := chessGame.Moves()
	log.Debug("analyzing %d moves", len(moves))

	if len(positions) != len(moves)+1 {
		log.Warn("unexpected positions length: got %d positions for %d moves", len(positions), len(moves))
	}

	userIsWhite := game.PlayedAs == "white"

	var blunders, mistakes, inaccuracies int
	var flashcardsCreated int
	var prevEval *analysis.EvalResult
	var positionsToInsert []models.Position
	var flashcardIndices []int // Track which positions need flashcards (indices in positionsToInsert)

	for i := 0; i < len(moves); i++ {
		if i >= len(positions)-1 {
			break
		}

		if ctx.Err() != nil {
			log.Warn("analysis cancelled: %v", ctx.Err())
			return ctx.Err()
		}

		// Determine whose turn it is to move (i even = white, i odd = black)
		isWhiteMove := i%2 == 0

		posBefore := positions[i]
		posAfter := positions[i+1]

		fenBefore := posBefore.String()
		fenAfter := posAfter.String()
		log = log.WithField("move_number", i+1).WithField("move_played", moves[i].String())
		log.Debug("fen before: %s", fenBefore)
		log.Debug("fen after: %s", fenAfter)

		// Reuse previous evaluation for evalBefore (the "after" position of move i-1 is the "before" position of move i)
		var evalBefore analysis.EvalResult
		if prevEval != nil {
			evalBefore = *prevEval
		} else {
			// First move: evaluate fenBefore normally
			var err error
			evalBefore, err = engine.EvaluateFEN(ctx, fenBefore, depth, maxTimeMs)
			if err != nil {
				log.Warn("eval before move %d failed: %v", i+1, err)
				continue
			}
		}

		// Always evaluate fenAfter
		evalAfter, err := engine.EvaluateFEN(ctx, fenAfter, depth, maxTimeMs)
		if err != nil {
			log.Warn("eval after move %d failed: %v", i+1, err)
			continue
		}
		// Store evalAfter as prevEval for the next iteration
		prevEval = &evalAfter

		// Normalize evaluation values for storage
		var mateBefore *int
		var mateAfter *int
		evalBeforeCP := evalBefore.CP
		evalAfterCP := evalAfter.CP
		if evalBefore.Mate != nil {
			mateBefore = evalBefore.Mate
			// When mate is present, store 0 in CP fields (mate takes precedence)
			evalBeforeCP = 0
		}
		if evalAfter.Mate != nil {
			mateAfter = evalAfter.Mate
			evalAfterCP = 0
		}

		diff := evalAfterCP - evalBeforeCP
		log.Debug("evalBefore: %+v", evalBefore)
		log.Debug("evalAfter: %+v", evalAfter)

		classification := analysis.ClassifyMove(evalBeforeCP, evalAfterCP, isWhiteMove)
		log.Debug("classification: %s", classification)

		switch classification {
		case "blunder":
			blunders++
		case "mistake":
			mistakes++
		case "inaccuracy":
			inaccuracies++
		}

		// Collect position for batch insert
		position := models.Position{
			GameID:         game.ID,
			MoveNumber:     i + 1,
			FEN:            fenBefore,
			MovePlayed:     moves[i].String(),
			BestMove:       evalBefore.BestMove,
			EvalBefore:     evalBeforeCP,
			EvalAfter:      evalAfterCP,
			EvalDiff:       diff,
			MateBefore:     mateBefore,
			MateAfter:      mateAfter,
			Classification: classification,
		}
		positionsToInsert = append(positionsToInsert, position)

		// Track which positions need flashcards (only for player moves with blunders/mistakes)
		isPlayerMove := isWhiteMove == userIsWhite
		if isPlayerMove && (classification == "blunder" || classification == "mistake") {
			flashcardIndices = append(flashcardIndices, len(positionsToInsert)-1)
		}
	}

	// Batch insert all positions
	positionIDs, err := s.positionRepo.InsertBatch(ctx, positionsToInsert)
	if err != nil {
		log.Error("failed to batch insert positions: %v", err)
		_ = s.gameRepo.UpdateStatus(ctx, gameID, "failed")
		return err
	}

	// Create flashcards for positions that need them
	for _, idx := range flashcardIndices {
		if idx < len(positionIDs) {
			card := models.Flashcard{
				PositionID:    positionIDs[idx],
				DueAt:         time.Now(),
				IntervalDays:  0,
				EaseFactor:    2.5,
				TimesReviewed: 0,
				TimesCorrect:  0,
			}
			if _, err := s.flashcardRepo.Insert(ctx, card); err != nil {
				log.Warn("failed to insert flashcard for position %d: %v", positionIDs[idx], err)
			} else {
				flashcardsCreated++
				log.Debug("flashcard created: %+v", card)
			}
		}
	}

	log.Info("analysis completed: %d moves, %d blunders, %d mistakes, %d inaccuracies, %d flashcards created",
		len(moves), blunders, mistakes, inaccuracies, flashcardsCreated)

	if err := s.gameRepo.UpdateStatus(ctx, gameID, "completed"); err != nil {
		log.Error("failed to update game status to completed: %v", err)
	}

	if err := s.statsRepo.RefreshProfileStats(ctx, game.ProfileID); err != nil {
		log.Warn("failed to refresh cached stats: %v", err)
	}

	return nil
}
