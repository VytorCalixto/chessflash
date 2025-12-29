package worker

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/corentings/chess"
	"github.com/vytor/chessflash/internal/analysis"
	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

type AnalyzeGameJob struct {
	DB             *db.DB
	GameID         int64
	StockfishPath  string
	StockfishDepth int
}

func (j *AnalyzeGameJob) Name() string { return "analyze_game" }

func (j *AnalyzeGameJob) Run(ctx context.Context) error {
	log := logger.FromContext(ctx).WithField("game_id", j.GameID)
	log.Info("starting game analysis")

	game, err := j.DB.GetGame(ctx, j.GameID)
	if err != nil {
		log.Error("failed to get game: %v", err)
		return err
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
	if err := j.DB.UpdateGameStatus(ctx, j.GameID, "processing"); err != nil {
		log.Error("failed to update game status: %v", err)
		return err
	}

	log.Debug("initializing stockfish engine")
	sf, err := analysis.NewEngine(j.StockfishPath)
	if err != nil {
		log.Error("failed to initialize stockfish: %v", err)
		_ = j.DB.UpdateGameStatus(ctx, j.GameID, "failed")
		return err
	}
	defer sf.Close()

	depth := j.StockfishDepth
	if depth <= 0 {
		depth = 18
	}
	log = log.WithField("depth", depth)

	log.Debug("parsing PGN")
	pgnOpt, err := chess.PGN(strings.NewReader(game.PGN))
	if err != nil {
		log.Error("failed to parse PGN: %v", err)
		_ = j.DB.UpdateGameStatus(ctx, j.GameID, "failed")
		return err
	}
	chessGame := chess.NewGame(pgnOpt)

	positions := chessGame.Positions()
	moves := chessGame.Moves()
	log.Debug("analyzing %d moves", len(moves))

	if len(positions) != len(moves)+1 {
		log.Warn("unexpected positions length: got %d positions for %d moves", len(positions), len(moves))
	}

	userIsWhite := game.PlayedAs == "white"

	var blunders, mistakes, inaccuracies int
	var flashcardsCreated int

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

		evalBefore, err := sf.EvaluateFEN(ctx, fenBefore, depth)
		if err != nil {
			log.Warn("eval before move %d failed: %v", i+1, err)
			continue
		}
		evalAfter, err := sf.EvaluateFEN(ctx, fenAfter, depth)
		if err != nil {
			log.Warn("eval after move %d failed: %v", i+1, err)
			continue
		}

		diff := evalAfter.CP - evalBefore.CP
		log.Debug("evalBefore: %+v", evalBefore)
		log.Debug("evalAfter: %+v", evalAfter)

		// Only create flashcards for moves made by the user (i even -> white, i odd -> black)
		isPlayerMove := isWhiteMove == userIsWhite
		if !isPlayerMove {
			continue
		}

		classification := analysis.ClassifyMove(evalBefore.CP, evalAfter.CP, isWhiteMove)
		log.Debug("classification: %s", classification)

		switch classification {
		case "blunder":
			blunders++
		case "mistake":
			mistakes++
		case "inaccuracy":
			inaccuracies++
		}

		posID, err := j.DB.InsertPosition(ctx, models.Position{
			GameID:         game.ID,
			MoveNumber:     i + 1,
			FEN:            fenBefore,
			MovePlayed:     moves[i].String(),
			BestMove:       evalBefore.BestMove,
			EvalBefore:     evalBefore.CP,
			EvalAfter:      evalAfter.CP,
			EvalDiff:       diff,
			Classification: classification,
		})
		if err != nil {
			log.Warn("failed to insert position for move %d: %v", i+1, err)
			continue
		}

		if classification == "blunder" || classification == "mistake" {
			card := models.Flashcard{
				PositionID:    posID,
				DueAt:         time.Now(),
				IntervalDays:  0,
				EaseFactor:    2.5,
				TimesReviewed: 0,
				TimesCorrect:  0,
			}
			if _, err := j.DB.InsertFlashcard(ctx, card); err != nil {
				log.Warn("failed to insert flashcard for position %d: %v", posID, err)
			} else {
				flashcardsCreated++
				log.Debug("flashcard created: %+v", card)
			}
		}
	}

	log.Info("analysis completed: %d moves, %d blunders, %d mistakes, %d inaccuracies, %d flashcards created",
		len(moves), blunders, mistakes, inaccuracies, flashcardsCreated)

	if err := j.DB.UpdateGameStatus(ctx, j.GameID, "completed"); err != nil {
		log.Error("failed to update game status to completed: %v", err)
	}

	return nil
}

// ImportGamesJob fetches recent archives, inserts games, and enqueues analysis.
type ImportGamesJob struct {
	DB             *db.DB
	ChessClient    *chesscom.Client
	Profile        models.Profile
	AnalysisPool   *Pool
	StockfishPath  string
	StockfishDepth int
	ArchiveLimit   int
	MaxConcurrent  int
}

func (j *ImportGamesJob) Name() string { return "import_games" }

func (j *ImportGamesJob) Run(ctx context.Context) error {
	log := logger.FromContext(ctx).WithFields(map[string]any{
		"username":   j.Profile.Username,
		"profile_id": j.Profile.ID,
	})
	log.Info("starting background import")

	archives, err := j.ChessClient.FetchArchives(ctx, j.Profile.Username)
	if err != nil {
		log.Error("failed to fetch archives: %v", err)
		return err
	}

	limit := j.ArchiveLimit
	if limit <= 0 {
		limit = 6
	}
	if len(archives) > limit {
		archives = archives[len(archives)-limit:]
		log.Debug("limiting to last %d archives", limit)
	}

	maxConc := j.MaxConcurrent
	if maxConc <= 0 {
		maxConc = 3
	}

	type archiveResult struct {
		games []chesscom.MonthlyGame
		err   error
	}

	results := make(chan archiveResult, len(archives))
	sem := make(chan struct{}, maxConc)

	var wg sync.WaitGroup
	for _, url := range archives {
		wg.Add(1)
		go func(archiveURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			monthly, err := j.ChessClient.FetchMonthly(ctx, archiveURL)
			select {
			case results <- archiveResult{games: monthly, err: err}:
			case <-ctx.Done():
				return
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var totalGames int
	for res := range results {
		if ctx.Err() != nil {
			log.Warn("import cancelled: %v", ctx.Err())
			return ctx.Err()
		}
		if res.err != nil {
			log.Error("failed to fetch monthly games: %v", res.err)
			continue
		}

		for _, mg := range res.games {
			gameMeta := parsePGNHeadersLocal(mg.PGN)
			playedAs, opponent, result := deriveResultLocal(strings.ToLower(j.Profile.Username), mg)

			game := models.Game{
				ProfileID:      j.Profile.ID,
				ChessComID:     extractGameIDLocal(mg.URL),
				PGN:            mg.PGN,
				TimeClass:      mg.TimeClass,
				Result:         result,
				PlayedAs:       playedAs,
				Opponent:       opponent,
				PlayedAt:       time.Unix(mg.EndTime, 0),
				ECOCode:        gameMeta["ECO"],
				OpeningName:    gameMeta["Opening"],
				OpeningURL:     gameMeta["ECOUrl"],
				AnalysisStatus: "pending",
			}

			gameID, err := j.DB.InsertGame(ctx, game)
			if err != nil {
				log.Warn("failed to insert game chess_com_id=%s: %v", game.ChessComID, err)
				continue
			}

			j.AnalysisPool.Submit(&AnalyzeGameJob{
				DB:             j.DB,
				GameID:         gameID,
				StockfishPath:  j.StockfishPath,
				StockfishDepth: j.StockfishDepth,
			})
			totalGames++
		}
	}

	log.Info("imported %d games and queued for analysis", totalGames)
	if err := j.DB.UpdateProfileSync(ctx, j.Profile.ID, time.Now()); err != nil {
		log.Warn("failed to update profile sync time: %v", err)
	}
	return nil
}

var headerReLocal = regexp.MustCompile(`\[(\w+)\s+"([^"]+)"\]`)

func parsePGNHeadersLocal(pgn string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(pgn, "\n") {
		if !strings.HasPrefix(line, "[") {
			continue
		}
		m := headerReLocal.FindStringSubmatch(line)
		if len(m) == 3 {
			out[m[1]] = m[2]
		}
	}
	return out
}

var gameIDReLocal = regexp.MustCompile(`.*/game/[^/]+/([0-9]+)`)

func extractGameIDLocal(url string) string {
	m := gameIDReLocal.FindStringSubmatch(url)
	if len(m) == 2 {
		return m[1]
	}
	return url
}

func deriveResultLocal(username string, mg chesscom.MonthlyGame) (playedAs, opponent, result string) {
	if strings.EqualFold(mg.White.Username, username) {
		playedAs = "white"
		opponent = mg.Black.Username
		result = normalizeResultLocal(mg.White.Result)
		return
	}
	playedAs = "black"
	opponent = mg.White.Username
	result = normalizeResultLocal(mg.Black.Result)
	return
}

func normalizeResultLocal(res string) string {
	res = strings.ToLower(res)
	switch res {
	case "win":
		return "win"
	case "stalemate", "agreed", "repetition", "timevsinsufficient", "insufficient", "fiftymove", "draw":
		return "draw"
	case "checkmated", "resigned", "timeout", "abandoned", "kingofthehill", "threecheck", "bughousepartnerlose":
		return "loss"
	default:
		return "loss"
	}
}
