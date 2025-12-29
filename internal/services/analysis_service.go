package services

import (
	"context"
	"fmt"
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
	EvaluatePosition(ctx context.Context, fen string) (analysis.EvalResult, error)
	AnalyzeGame(ctx context.Context, gameID int64) error
}

type analysisService struct {
	gameRepo      repository.GameRepository
	positionRepo  repository.PositionRepository
	flashcardRepo repository.FlashcardRepository
	statsRepo     repository.StatsRepository
	config        AnalysisConfig
	pool          *analysis.EnginePool
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

func (s *analysisService) EvaluatePosition(ctx context.Context, fen string) (analysis.EvalResult, error) {
	log := logger.FromContext(ctx)
	log.Debug("evaluating position: fen=%s", fen)

	if fen == "" {
		return analysis.EvalResult{}, errors.NewValidationError("fen", "cannot be empty")
	}

	// Use a lighter depth for real-time evaluation (faster response)
	depth := 15
	if s.config.StockfishDepth > 0 && s.config.StockfishDepth < 15 {
		depth = s.config.StockfishDepth
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

	if err := s.gameRepo.UpdateStatus(ctx, gameID, "processing"); err != nil {
		log.Error("failed to update game status: %v", err)
		return err
	}

	engine, depth, maxTimeMs, err := s.acquireEngineAndConfig(ctx, gameID, log)
	if err != nil {
		return err
	}
	defer s.pool.Release(engine)

	chessGame, err := s.parseGamePGN(ctx, game, log, gameID)
	if err != nil {
		return err
	}

	s.detectOpeningIfMissing(ctx, game, chessGame, log)

	positions := chessGame.Positions()
	moves := chessGame.Moves()
	log.Debug("analyzing %d moves", len(moves))

	if len(positions) != len(moves)+1 {
		log.Warn("unexpected positions length: got %d positions for %d moves", len(positions), len(moves))
	}

	analysisResult := s.analyzePositions(ctx, engine, positions, moves, game, depth, maxTimeMs, log)

	if err := s.saveAnalysisResults(ctx, gameID, analysisResult, log); err != nil {
		return err
	}

	s.finalizeAnalysis(ctx, gameID, game.ProfileID, len(moves), analysisResult, log)
	return nil
}

// analysisResult holds the results of analyzing a game
type analysisResult struct {
	positions         []models.Position
	flashcardIndices  []int
	blunders          int
	mistakes          int
	inaccuracies      int
	flashcardsCreated int
}

// acquireEngineAndConfig acquires an engine and returns configuration
func (s *analysisService) acquireEngineAndConfig(ctx context.Context, gameID int64, log *logger.Logger) (*analysis.Engine, int, int, error) {
	log.Debug("acquiring stockfish engine from pool")
	engine, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Error("failed to acquire engine from pool: %v", err)
		_ = s.gameRepo.UpdateStatus(ctx, gameID, "failed")
		return nil, 0, 0, err
	}

	depth := s.config.StockfishDepth
	if depth <= 0 {
		depth = 18
	}
	maxTimeMs := s.config.StockfishMaxTime
	log = log.WithFields(map[string]any{
		"depth":       depth,
		"max_time_ms": maxTimeMs,
	})

	return engine, depth, maxTimeMs, nil
}

// parseGamePGN parses the game PGN and creates a chess game
func (s *analysisService) parseGamePGN(ctx context.Context, game *models.Game, log *logger.Logger, gameID int64) (*chess.Game, error) {
	log.Debug("parsing PGN")
	pgnOpt, err := chess.PGN(strings.NewReader(game.PGN))
	if err != nil {
		log.Error("failed to parse PGN: %v", err)
		_ = s.gameRepo.UpdateStatus(ctx, gameID, "failed")
		return nil, err
	}
	return chess.NewGame(pgnOpt), nil
}

// detectOpeningIfMissing detects and updates the opening if missing
func (s *analysisService) detectOpeningIfMissing(ctx context.Context, game *models.Game, chessGame *chess.Game, log *logger.Logger) {
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
}

// analyzePositions analyzes all positions in the game
func (s *analysisService) analyzePositions(
	ctx context.Context,
	engine *analysis.Engine,
	positions []*chess.Position,
	moves []*chess.Move,
	game *models.Game,
	depth, maxTimeMs int,
	log *logger.Logger,
) *analysisResult {
	userIsWhite := game.PlayedAs == "white"
	result := &analysisResult{
		positions:        make([]models.Position, 0, len(moves)),
		flashcardIndices: make([]int, 0),
	}

	var prevEval *analysis.EvalResult

	for i := 0; i < len(moves); i++ {
		if i >= len(positions)-1 {
			break
		}

		if ctx.Err() != nil {
			log.Warn("analysis cancelled: %v", ctx.Err())
			break
		}

		isWhiteMove := i%2 == 0
		posBefore := positions[i]
		posAfter := positions[i+1]

		position, _, evalAfter, shouldCreateFlashcard := s.analyzePosition(
			ctx, engine, posBefore, posAfter, moves[i], i+1,
			isWhiteMove, userIsWhite, prevEval, depth, maxTimeMs, game.ID, log,
		)

		if position != nil {
			result.positions = append(result.positions, *position)
			if shouldCreateFlashcard {
				result.flashcardIndices = append(result.flashcardIndices, len(result.positions)-1)
			}

			switch position.Classification {
			case "blunder":
				result.blunders++
			case "mistake":
				result.mistakes++
			case "inaccuracy":
				result.inaccuracies++
			}
		}

		if evalAfter != nil {
			prevEval = evalAfter
		}
	}

	return result
}

// analyzePosition analyzes a single position
func (s *analysisService) analyzePosition(
	ctx context.Context,
	engine *analysis.Engine,
	posBefore, posAfter *chess.Position,
	move *chess.Move,
	moveNumber int,
	isWhiteMove, userIsWhite bool,
	prevEval *analysis.EvalResult,
	depth, maxTimeMs int,
	gameID int64,
	log *logger.Logger,
) (*models.Position, *analysis.EvalResult, *analysis.EvalResult, bool) {
	fenBefore := posBefore.String()
	fenAfter := posAfter.String()
	log = log.WithField("move_number", moveNumber).WithField("move_played", move.String())
	log.Debug("fen before: %s", fenBefore)
	log.Debug("fen after: %s", fenAfter)

	// Get evaluation before move
	var evalBefore analysis.EvalResult
	if prevEval != nil {
		evalBefore = *prevEval
	} else {
		var err error
		evalBefore, err = engine.EvaluateFEN(ctx, fenBefore, depth, maxTimeMs)
		if err != nil {
			log.Warn("eval before move %d failed: %v", moveNumber, err)
			return nil, nil, nil, false
		}
	}

	// Get evaluation after move
	evalAfter, err := engine.EvaluateFEN(ctx, fenAfter, depth, maxTimeMs)
	if err != nil {
		log.Warn("eval after move %d failed: %v", moveNumber, err)
		return nil, nil, nil, false
	}

	evalBeforeCP, mateBefore := normalizeEvaluation(evalBefore)
	evalAfterCP, mateAfter := normalizeEvaluation(evalAfter)

	diff := evalAfterCP - evalBeforeCP
	movePlayedUCI := analysis.MoveToUCI(move)
	bestMoveUCI := evalBefore.BestMove

	classification := analysis.ClassifyMove(evalBeforeCP, evalAfterCP, isWhiteMove, movePlayedUCI, bestMoveUCI)
	log.Debug("classification: %s (movePlayed: %s, bestMove: %s)", classification, movePlayedUCI, bestMoveUCI)

	position := &models.Position{
		GameID:         gameID,
		MoveNumber:     moveNumber,
		FEN:            fenBefore,
		MovePlayed:     movePlayedUCI,
		BestMove:       bestMoveUCI,
		EvalBefore:     evalBeforeCP,
		EvalAfter:      evalAfterCP,
		EvalDiff:       diff,
		MateBefore:     mateBefore,
		MateAfter:      mateAfter,
		Classification: classification,
		CreatedAt:      time.Now(),
	}

	isPlayerMove := isWhiteMove == userIsWhite
	shouldCreateFlashcard := false

	if isPlayerMove {
		shouldCreateFlashcard = s.shouldCreateFlashcardForPosition(
			ctx, engine, posBefore, movePlayedUCI, bestMoveUCI,
			classification, evalAfterCP, mateAfter, isWhiteMove,
			depth, maxTimeMs, log,
		)
	}

	evalAfterPtr := &evalAfter
	return position, &evalBefore, evalAfterPtr, shouldCreateFlashcard
}

// normalizeEvaluation extracts CP and mate values from evaluation result
func normalizeEvaluation(eval analysis.EvalResult) (float64, *int) {
	if eval.Mate != nil {
		return 0, eval.Mate
	}
	return eval.CP, nil
}

// shouldCreateFlashcardForPosition determines if a flashcard should be created for a position
func (s *analysisService) shouldCreateFlashcardForPosition(
	ctx context.Context,
	engine *analysis.Engine,
	posBefore *chess.Position,
	movePlayedUCI, bestMoveUCI, classification string,
	evalAfterCP float64,
	mateAfter *int,
	isWhiteMove bool,
	depth, maxTimeMs int,
	log *logger.Logger,
) bool {
	// If the move played is the best move, don't create a flashcard
	if movePlayedUCI == bestMoveUCI {
		return false
	}

	// Create flashcard for blunders/mistakes
	if classification == "blunder" || classification == "mistake" {
		return true
	}

	// Also create flashcard if there's a better move available (improvement >= 100 centipawns)
	if movePlayedUCI != bestMoveUCI && bestMoveUCI != "" {
		return s.evaluateBestMoveImprovement(
			ctx, engine, posBefore, bestMoveUCI, evalAfterCP, mateAfter, isWhiteMove, depth, maxTimeMs, log,
		)
	}

	return false
}

// evaluateBestMoveImprovement evaluates if the best move significantly improves the position
func (s *analysisService) evaluateBestMoveImprovement(
	ctx context.Context,
	engine *analysis.Engine,
	posBefore *chess.Position,
	bestMoveUCI string,
	evalAfterCP float64,
	mateAfter *int,
	isWhiteMove bool,
	depth, maxTimeMs int,
	log *logger.Logger,
) bool {
	posAfterBestMove, err := applyMoveToPosition(posBefore, bestMoveUCI)
	if err != nil {
		log.Warn("failed to apply best move to position: %v", err)
		return false
	}

	fenAfterBestMove := posAfterBestMove.String()
	evalAfterBestMove, err := engine.EvaluateFEN(ctx, fenAfterBestMove, depth, maxTimeMs)
	if err != nil {
		log.Warn("failed to evaluate position after best move: %v", err)
		return false
	}

	evalAfterBestMoveCP, bestMate := normalizeEvaluation(evalAfterBestMove)

	// Handle mate positions
	if bestMate != nil {
		return s.compareMateEvaluations(*bestMate, mateAfter, isWhiteMove, log)
	}

	// Compare centipawn evaluations if neither leads to mate
	if mateAfter == nil {
		improvement := evalAfterBestMoveCP - evalAfterCP
		if !isWhiteMove {
			improvement = -improvement
		}
		if improvement >= 100 {
			log.Debug("better move found: improvement=%.0f cp (played: %.0f, best: %.0f)",
				improvement, evalAfterCP, evalAfterBestMoveCP)
			return true
		}
	}

	return false
}

// compareMateEvaluations compares mate evaluations to determine if best move is better
func (s *analysisService) compareMateEvaluations(
	bestMateVal int,
	playedMate *int,
	isWhiteMove bool,
	log *logger.Logger,
) bool {
	if playedMate == nil {
		// Played move doesn't lead to mate, but best move does - always create flashcard
		// This includes both winning mates (positive for white, negative for black)
		// and losing mates (the user should learn to avoid them)
		if bestMateVal != 0 {
			log.Debug("better move found: best move leads to mate (played move doesn't) - mate in %d", bestMateVal)
			return true
		}
		return false
	}

	// Both lead to mate - compare mate distances
	playedMateVal := *playedMate
	// Shorter mate distance is better (more positive for white, more negative for black)
	if isWhiteMove {
		if bestMateVal > 0 && (playedMateVal <= 0 || bestMateVal < playedMateVal) {
			log.Debug("better move found: best move mates faster (best: %d, played: %d)", bestMateVal, playedMateVal)
			return true
		}
	} else {
		if bestMateVal < 0 && (playedMateVal >= 0 || bestMateVal > playedMateVal) {
			log.Debug("better move found: best move mates faster (best: %d, played: %d)", bestMateVal, playedMateVal)
			return true
		}
	}

	return false
}

// saveAnalysisResults saves positions and creates flashcards
func (s *analysisService) saveAnalysisResults(
	ctx context.Context,
	gameID int64,
	result *analysisResult,
	log *logger.Logger,
) error {
	positionIDs, err := s.positionRepo.InsertBatch(ctx, result.positions)
	if err != nil {
		log.Error("failed to batch insert positions: %v", err)
		_ = s.gameRepo.UpdateStatus(ctx, gameID, "failed")
		return err
	}

	flashcardsCreated := s.createFlashcards(ctx, positionIDs, result.flashcardIndices, log)
	result.flashcardsCreated = flashcardsCreated // Store for logging

	return nil
}

// createFlashcards creates flashcards for the specified position indices
func (s *analysisService) createFlashcards(
	ctx context.Context,
	positionIDs []int64,
	flashcardIndices []int,
	log *logger.Logger,
) int {
	flashcardsCreated := 0
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
	return flashcardsCreated
}

// finalizeAnalysis updates game status and refreshes stats
func (s *analysisService) finalizeAnalysis(
	ctx context.Context,
	gameID int64,
	profileID int64,
	totalMoves int,
	result *analysisResult,
	log *logger.Logger,
) {
	log.Info("analysis completed: %d moves, %d blunders, %d mistakes, %d inaccuracies, %d flashcards created",
		totalMoves, result.blunders, result.mistakes, result.inaccuracies, result.flashcardsCreated)

	if err := s.gameRepo.UpdateStatus(ctx, gameID, "completed"); err != nil {
		log.Error("failed to update game status to completed: %v", err)
	}

	if err := s.statsRepo.RefreshProfileStats(ctx, profileID); err != nil {
		log.Warn("failed to refresh cached stats: %v", err)
	}
}

// applyMoveToPosition applies a UCI move to a position and returns the new position
func applyMoveToPosition(pos *chess.Position, moveUCI string) (*chess.Position, error) {
	if len(moveUCI) < 4 {
		return nil, fmt.Errorf("invalid UCI move: %s", moveUCI)
	}

	// Create a new game from the position's FEN
	fen := pos.String()
	fenOpt, err := chess.FEN(fen)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FEN: %v", err)
	}

	game := chess.NewGame(fenOpt)

	// Apply the move using UCI notation
	if err := game.PushNotationMove(moveUCI, chess.UCINotation{}, nil); err != nil {
		return nil, fmt.Errorf("failed to apply move %s: %v", moveUCI, err)
	}

	return game.Position(), nil
}
