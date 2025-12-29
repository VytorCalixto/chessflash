package worker

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/pgn"
	"github.com/vytor/chessflash/internal/repository"
)

type AnalyzeGameJob struct {
	AnalysisService AnalysisServiceInterface
	GameID          int64
}

func (j *AnalyzeGameJob) Name() string { return "analyze_game" }

func (j *AnalyzeGameJob) Run(ctx context.Context) error {
	return j.AnalysisService.AnalyzeGame(ctx, j.GameID)
}

// ImportGamesJob fetches recent archives, inserts games, and enqueues analysis.
type ImportGamesJob struct {
	GameRepo       repository.GameRepository
	ProfileRepo    repository.ProfileRepository
	StatsRepo      repository.StatsRepository
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

	if j.Profile.LastSyncAt != nil {
		archives = filterArchivesByDate(archives, *j.Profile.LastSyncAt)
		log.Info("filtered archives to %d based on last_sync_at", len(archives))
	}

	// ArchiveLimit of 0 means fetch all archives
	if j.ArchiveLimit > 0 && len(archives) > j.ArchiveLimit {
		archives = archives[len(archives)-j.ArchiveLimit:]
		log.Debug("limiting to last %d archives", j.ArchiveLimit)
	}
	log.Info("fetching %d archives in parallel", len(archives))

	maxConc := j.MaxConcurrent
	if maxConc <= 0 {
		maxConc = 10
	}
	log.Debug("using %d concurrent workers for archive fetching", maxConc)

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

	existingIDs, err := j.GameRepo.GetExistingChessComIDs(ctx, j.Profile.ID)
	if err != nil {
		log.Warn("failed to load existing game ids: %v", err)
		existingIDs = map[string]bool{}
	}

	var monthlyGames []chesscom.MonthlyGame
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

		monthlyGames = append(monthlyGames, res.games...)
	}

	if len(monthlyGames) == 0 {
		log.Info("no monthly games fetched")
		return nil
	}

	var newGames []models.Game
	for _, mg := range monthlyGames {
		gameID := pgn.ExtractGameID(mg.URL)
		if existingIDs[gameID] {
			continue
		}
		existingIDs[gameID] = true // avoid duplicates in batch

		gameMeta := pgn.ParsePGNHeaders(mg.PGN)
		playedAs, opponent, result := chesscom.DeriveResult(strings.ToLower(j.Profile.Username), mg)

		// Extract ratings from PGN headers (best effort).
		var playerRating, opponentRating int
		if playedAs == "white" {
			playerRating, _ = strconv.Atoi(gameMeta["WhiteElo"])
			opponentRating, _ = strconv.Atoi(gameMeta["BlackElo"])
		} else {
			playerRating, _ = strconv.Atoi(gameMeta["BlackElo"])
			opponentRating, _ = strconv.Atoi(gameMeta["WhiteElo"])
		}
		if playerRating == 0 || opponentRating == 0 {
			if playedAs == "white" {
				if playerRating == 0 {
					playerRating = mg.White.Rating
				}
				if opponentRating == 0 {
					opponentRating = mg.Black.Rating
				}
			} else {
				if playerRating == 0 {
					playerRating = mg.Black.Rating
				}
				if opponentRating == 0 {
					opponentRating = mg.White.Rating
				}
			}
		}

		game := models.Game{
			ProfileID:      j.Profile.ID,
			ChessComID:     gameID,
			PGN:            mg.PGN,
			TimeClass:      mg.TimeClass,
			Result:         result,
			PlayedAs:       playedAs,
			Opponent:       opponent,
			PlayerRating:   playerRating,
			OpponentRating: opponentRating,
			PlayedAt:       time.Unix(mg.EndTime, 0),
			ECOCode:        gameMeta["ECO"],
			OpeningName:    gameMeta["Opening"],
			OpeningURL:     gameMeta["ECOUrl"],
			AnalysisStatus: "pending",
		}
		newGames = append(newGames, game)
	}

	inserted, err := j.GameRepo.InsertBatch(ctx, newGames)
	if err != nil {
		log.Error("failed to batch insert games: %v", err)
		return err
	}

	totalGames = len(inserted)
	log.Info("imported %d new games", totalGames)
	if err := j.ProfileRepo.UpdateSync(ctx, j.Profile.ID, time.Now()); err != nil {
		log.Warn("failed to update profile sync time: %v", err)
	}

	if err := j.StatsRepo.RefreshProfileStats(ctx, j.Profile.ID); err != nil {
		log.Warn("failed to refresh cached stats after import: %v", err)
	}
	return nil
}

// filterArchivesByDate keeps archives from the given month/year onwards.
// Archive URLs look like: https://api.chess.com/pub/player/{username}/games/YYYY/MM
func filterArchivesByDate(archives []string, since time.Time) []string {
	if since.IsZero() {
		return archives
	}
	sinceMonth := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, time.UTC)

	var filtered []string
	for _, url := range archives {
		parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
		if len(parts) < 2 {
			continue
		}
		yearStr := parts[len(parts)-2]
		monthStr := parts[len(parts)-1]

		year, err1 := strconv.Atoi(yearStr)
		monthInt, err2 := strconv.Atoi(monthStr)
		if err1 != nil || err2 != nil {
			continue
		}
		archiveMonth := time.Date(year, time.Month(monthInt), 1, 0, 0, 0, 0, time.UTC)
		if archiveMonth.Before(sinceMonth) {
			continue
		}
		filtered = append(filtered, url)
	}
	return filtered
}
