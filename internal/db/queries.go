package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

func (db *DB) UpsertProfile(ctx context.Context, username string) (*models.Profile, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("upserting profile for username: %s", username)

	var p models.Profile
	err := db.QueryRowContext(ctx, `
INSERT INTO profiles (username)
VALUES (?)
ON CONFLICT(username) DO UPDATE SET username = excluded.username
RETURNING id, username, created_at, last_sync_at
`, username).Scan(&p.ID, &p.Username, &p.CreatedAt, &p.LastSyncAt)
	if err != nil {
		log.Error("failed to upsert profile: %v", err)
		return nil, err
	}
	log.Debug("profile upserted: id=%d", p.ID)
	return &p, nil
}

func (db *DB) UpdateProfileSync(ctx context.Context, id int64, t time.Time) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("updating profile sync time: profile_id=%d", id)

	_, err := db.ExecContext(ctx, `UPDATE profiles SET last_sync_at = ? WHERE id = ?`, t, id)
	if err != nil {
		log.Error("failed to update profile sync: %v", err)
	}
	return err
}

func (db *DB) ListProfiles(ctx context.Context) ([]models.Profile, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("listing profiles")

	rows, err := db.QueryContext(ctx, `
SELECT id, username, created_at, last_sync_at
FROM profiles
ORDER BY created_at ASC
`)
	if err != nil {
		log.Error("failed to list profiles: %v", err)
		return nil, err
	}
	defer rows.Close()

	var profiles []models.Profile
	for rows.Next() {
		var p models.Profile
		if err := rows.Scan(&p.ID, &p.Username, &p.CreatedAt, &p.LastSyncAt); err != nil {
			log.Error("failed to scan profile row: %v", err)
			return nil, err
		}
		profiles = append(profiles, p)
	}

	log.Debug("found %d profiles", len(profiles))
	return profiles, rows.Err()
}

func (db *DB) GetProfile(ctx context.Context, id int64) (*models.Profile, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("getting profile: id=%d", id)

	var p models.Profile
	err := db.QueryRowContext(ctx, `
SELECT id, username, created_at, last_sync_at
FROM profiles
WHERE id = ?
`, id).Scan(&p.ID, &p.Username, &p.CreatedAt, &p.LastSyncAt)
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("profile not found: id=%d", id)
		return nil, nil
	}
	if err != nil {
		log.Error("failed to get profile: %v", err)
		return nil, err
	}
	return &p, nil
}

func (db *DB) DeleteProfile(ctx context.Context, id int64) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("deleting profile and related data: id=%d", id)

	return tx(ctx, db, func(tx *sql.Tx) error {
		// Delete flashcards -> positions -> games -> profile to respect FK constraints.
		if _, err := tx.ExecContext(ctx, `
DELETE FROM flashcards
WHERE position_id IN (
    SELECT p.id FROM positions p
    JOIN games g ON g.id = p.game_id
    WHERE g.profile_id = ?
)
`, id); err != nil {
			log.Error("failed to delete flashcards for profile %d: %v", id, err)
			return err
		}

		if _, err := tx.ExecContext(ctx, `
DELETE FROM positions
WHERE game_id IN (SELECT id FROM games WHERE profile_id = ?)
`, id); err != nil {
			log.Error("failed to delete positions for profile %d: %v", id, err)
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM games WHERE profile_id = ?`, id); err != nil {
			log.Error("failed to delete games for profile %d: %v", id, err)
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id); err != nil {
			log.Error("failed to delete profile %d: %v", id, err)
			return err
		}

		log.Debug("profile %d deleted with cascading data", id)
		return nil
	})
}

func (db *DB) InsertGame(ctx context.Context, g models.Game) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("inserting game: chess_com_id=%s, opponent=%s", g.ChessComID, g.Opponent)

	res, err := db.ExecContext(ctx, `
INSERT INTO games (
    profile_id, chess_com_id, pgn, time_class, result, played_as,
    opponent, played_at, eco_code, opening_name, opening_url, analysis_status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chess_com_id) DO UPDATE SET
    time_class = excluded.time_class,
    result = excluded.result,
    played_as = excluded.played_as,
    opponent = excluded.opponent,
    played_at = excluded.played_at,
    eco_code = excluded.eco_code,
    opening_name = excluded.opening_name,
    opening_url = excluded.opening_url
`, g.ProfileID, g.ChessComID, g.PGN, g.TimeClass, g.Result, g.PlayedAs, g.Opponent, g.PlayedAt, g.ECOCode, g.OpeningName, g.OpeningURL, g.AnalysisStatus)
	if err != nil {
		log.Error("failed to insert game: %v", err)
		return 0, err
	}
	if id, err := res.LastInsertId(); err == nil && id != 0 {
		log.Debug("game inserted: id=%d", id)
		return id, nil
	}
	var id int64
	err = db.QueryRowContext(ctx, `SELECT id FROM games WHERE chess_com_id = ?`, g.ChessComID).Scan(&id)
	if err != nil {
		log.Error("failed to get game id: %v", err)
	} else {
		log.Debug("game exists: id=%d", id)
	}
	return id, err
}

// InsertGamesBatch inserts multiple games within a single transaction.
// It returns the IDs of newly inserted games (existing games are skipped).
func (db *DB) InsertGamesBatch(ctx context.Context, games []models.Game) ([]int64, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("batch inserting %d games", len(games))

	if len(games) == 0 {
		return nil, nil
	}

	var insertedIDs []int64
	err := tx(ctx, db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
INSERT INTO games (
    profile_id, chess_com_id, pgn, time_class, result, played_as,
    opponent, played_at, eco_code, opening_name, opening_url, analysis_status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chess_com_id) DO NOTHING
`)
		if err != nil {
			log.Error("failed to prepare batch insert: %v", err)
			return err
		}
		defer stmt.Close()

		for _, g := range games {
			res, err := stmt.ExecContext(ctx, g.ProfileID, g.ChessComID, g.PGN, g.TimeClass, g.Result, g.PlayedAs, g.Opponent, g.PlayedAt, g.ECOCode, g.OpeningName, g.OpeningURL, g.AnalysisStatus)
			if err != nil {
				log.Error("failed to insert game chess_com_id=%s: %v", g.ChessComID, err)
				return err
			}
			if id, err := res.LastInsertId(); err == nil && id != 0 {
				insertedIDs = append(insertedIDs, id)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Debug("batch insert completed, %d new games inserted", len(insertedIDs))
	return insertedIDs, nil
}

// GetExistingChessComIDs returns a set of chess.com IDs already stored for the profile.
func (db *DB) GetExistingChessComIDs(ctx context.Context, profileID int64) (map[string]bool, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("loading existing chess_com_ids for profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `SELECT chess_com_id FROM games WHERE profile_id = ?`, profileID)
	if err != nil {
		log.Error("failed to list chess_com_ids: %v", err)
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Error("failed to scan chess_com_id: %v", err)
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// UpdateGameOpening updates ECO/opening fields for a game.
func (db *DB) UpdateGameOpening(ctx context.Context, id int64, ecoCode, openingName string) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("updating game opening: game_id=%d, eco=%s, opening=%s", id, ecoCode, openingName)

	_, err := db.ExecContext(ctx, `
UPDATE games
SET eco_code = ?, opening_name = ?
WHERE id = ?
`, ecoCode, openingName, id)
	if err != nil {
		log.Error("failed to update game opening: %v", err)
	}
	return err
}

func (db *DB) ListGames(ctx context.Context, filter models.GameFilter) ([]models.Game, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("listing games with filter: profile_id=%d, time_class=%s, result=%s, opening=%s",
		filter.ProfileID, filter.TimeClass, filter.Result, filter.OpeningName)

	clauses := []string{}
	args := []any{}
	if filter.ProfileID != 0 {
		clauses = append(clauses, "profile_id = ?")
		args = append(args, filter.ProfileID)
	}
	if filter.TimeClass != "" {
		clauses = append(clauses, "time_class = ?")
		args = append(args, filter.TimeClass)
	}
	if filter.Result != "" {
		clauses = append(clauses, "result = ?")
		args = append(args, filter.Result)
	}
	if filter.OpeningName != "" {
		clauses = append(clauses, "opening_name = ?")
		args = append(args, filter.OpeningName)
	}
	where := whereParts(clauses)

	orderBy := "played_at"
	if filter.OrderBy == "played_at" {
		orderBy = filter.OrderBy
	}
	orderDir := "DESC"
	if filter.OrderDir == "ASC" || filter.OrderDir == "DESC" {
		orderDir = filter.OrderDir
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
SELECT id, profile_id, chess_com_id, pgn, time_class, result, played_as, opponent, played_at,
       eco_code, opening_name, opening_url, analysis_status, created_at
FROM games
` + where + `
ORDER BY ` + orderBy + ` ` + orderDir + `
LIMIT ? OFFSET ?
`
	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to list games: %v", err)
		return nil, err
	}
	defer rows.Close()
	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.ProfileID, &g.ChessComID, &g.PGN, &g.TimeClass, &g.Result, &g.PlayedAs, &g.Opponent, &g.PlayedAt, &g.ECOCode, &g.OpeningName, &g.OpeningURL, &g.AnalysisStatus, &g.CreatedAt); err != nil {
			log.Error("failed to scan game row: %v", err)
			return nil, err
		}
		games = append(games, g)
	}
	log.Debug("found %d games", len(games))
	return games, rows.Err()
}

// CountGames returns the number of games matching the given filter.
func (db *DB) CountGames(ctx context.Context, filter models.GameFilter) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("counting games with filter: profile_id=%d, time_class=%s, result=%s, opening=%s",
		filter.ProfileID, filter.TimeClass, filter.Result, filter.OpeningName)

	clauses := []string{}
	args := []any{}
	if filter.ProfileID != 0 {
		clauses = append(clauses, "profile_id = ?")
		args = append(args, filter.ProfileID)
	}
	if filter.TimeClass != "" {
		clauses = append(clauses, "time_class = ?")
		args = append(args, filter.TimeClass)
	}
	if filter.Result != "" {
		clauses = append(clauses, "result = ?")
		args = append(args, filter.Result)
	}
	if filter.OpeningName != "" {
		clauses = append(clauses, "opening_name = ?")
		args = append(args, filter.OpeningName)
	}
	where := whereParts(clauses)

	var count int
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM games
`+where, args...).Scan(&count)
	if err != nil {
		log.Error("failed to count games: %v", err)
		return 0, err
	}
	return count, nil
}

func (db *DB) GetGame(ctx context.Context, id int64) (*models.Game, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("getting game: id=%d", id)

	var g models.Game
	err := db.QueryRowContext(ctx, `
SELECT id, profile_id, chess_com_id, pgn, time_class, result, played_as, opponent, played_at,
       eco_code, opening_name, opening_url, analysis_status, created_at
FROM games
WHERE id = ?
`, id).Scan(&g.ID, &g.ProfileID, &g.ChessComID, &g.PGN, &g.TimeClass, &g.Result, &g.PlayedAs, &g.Opponent, &g.PlayedAt, &g.ECOCode, &g.OpeningName, &g.OpeningURL, &g.AnalysisStatus, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("game not found: id=%d", id)
		} else {
			log.Error("failed to get game: %v", err)
		}
		return nil, err
	}
	log.Debug("game found: opponent=%s, result=%s", g.Opponent, g.Result)
	return &g, nil
}

func (db *DB) UpdateGameStatus(ctx context.Context, id int64, status string) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("updating game status: game_id=%d, status=%s", id, status)

	_, err := db.ExecContext(ctx, `UPDATE games SET analysis_status = ? WHERE id = ?`, status, id)
	if err != nil {
		log.Error("failed to update game status: %v", err)
	}
	return err
}

// ResetProcessingToPending marks in-progress games back to pending for a profile.
func (db *DB) ResetProcessingToPending(ctx context.Context, profileID int64) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("resetting processing games to pending: profile_id=%d", profileID)

	_, err := db.ExecContext(ctx, `
UPDATE games
SET analysis_status = 'pending'
WHERE profile_id = ? AND analysis_status = 'processing'
`, profileID)
	if err != nil {
		log.Error("failed to reset processing games: %v", err)
	}
	return err
}

// GamesNeedingAnalysis returns games that are not completed.
func (db *DB) GamesNeedingAnalysis(ctx context.Context, profileID int64) ([]models.Game, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("listing games needing analysis: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT id, profile_id, chess_com_id, pgn, time_class, result, played_as, opponent, played_at,
       eco_code, opening_name, opening_url, analysis_status, created_at
FROM games
WHERE profile_id = ? AND analysis_status IN ('pending','processing','failed')
ORDER BY played_at DESC
`, profileID)
	if err != nil {
		log.Error("failed to list games needing analysis: %v", err)
		return nil, err
	}
	defer rows.Close()

	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.ProfileID, &g.ChessComID, &g.PGN, &g.TimeClass, &g.Result, &g.PlayedAs, &g.Opponent, &g.PlayedAt,
			&g.ECOCode, &g.OpeningName, &g.OpeningURL, &g.AnalysisStatus, &g.CreatedAt); err != nil {
			log.Error("failed to scan game row: %v", err)
			return nil, err
		}
		games = append(games, g)
	}
	log.Debug("found %d games needing analysis", len(games))
	return games, rows.Err()
}

// CountGamesNeedingAnalysis returns the number of incomplete games for a profile.
func (db *DB) CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("counting games needing analysis: profile_id=%d", profileID)

	var count int
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM games
WHERE profile_id = ? AND analysis_status IN ('pending','processing','failed')
`, profileID).Scan(&count)
	if err != nil {
		log.Error("failed to count games needing analysis: %v", err)
		return 0, err
	}
	return count, nil
}

func (db *DB) InsertPosition(ctx context.Context, p models.Position) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("inserting position: game_id=%d, move_number=%d, classification=%s",
		p.GameID, p.MoveNumber, p.Classification)

	res, err := db.ExecContext(ctx, `
INSERT INTO positions (game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, mate_before, mate_after, classification)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, p.GameID, p.MoveNumber, p.FEN, p.MovePlayed, p.BestMove, p.EvalBefore, p.EvalAfter, p.EvalDiff, p.MateBefore, p.MateAfter, p.Classification)
	if err != nil {
		log.Error("failed to insert position: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error("failed to get position id: %v", err)
		return 0, err
	}
	log.Debug("position inserted: id=%d", id)
	return id, nil
}

func (db *DB) PositionsForGame(ctx context.Context, gameID int64) ([]models.Position, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching positions for game: game_id=%d", gameID)

	rows, err := db.QueryContext(ctx, `
SELECT id, game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, mate_before, mate_after, classification, created_at
FROM positions
WHERE game_id = ?
ORDER BY move_number ASC
`, gameID)
	if err != nil {
		log.Error("failed to query positions: %v", err)
		return nil, err
	}
	defer rows.Close()
	var positions []models.Position
	for rows.Next() {
		var p models.Position
		if err := rows.Scan(&p.ID, &p.GameID, &p.MoveNumber, &p.FEN, &p.MovePlayed, &p.BestMove, &p.EvalBefore, &p.EvalAfter, &p.EvalDiff, &p.MateBefore, &p.MateAfter, &p.Classification, &p.CreatedAt); err != nil {
			log.Error("failed to scan position row: %v", err)
			return nil, err
		}
		positions = append(positions, p)
	}
	log.Debug("found %d positions", len(positions))
	return positions, rows.Err()
}

func (db *DB) InsertFlashcard(ctx context.Context, c models.Flashcard) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("inserting flashcard: position_id=%d", c.PositionID)

	res, err := db.ExecContext(ctx, `
INSERT INTO flashcards (position_id, due_at, interval_days, ease_factor, times_reviewed, times_correct)
VALUES (?, ?, ?, ?, ?, ?)
`, c.PositionID, c.DueAt, c.IntervalDays, c.EaseFactor, c.TimesReviewed, c.TimesCorrect)
	if err != nil {
		log.Error("failed to insert flashcard: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error("failed to get flashcard id: %v", err)
		return 0, err
	}
	log.Debug("flashcard inserted: id=%d", id)
	return id, nil
}

func (db *DB) UpdateFlashcard(ctx context.Context, c models.Flashcard) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("updating flashcard: id=%d, interval=%d, ease=%.2f", c.ID, c.IntervalDays, c.EaseFactor)

	_, err := db.ExecContext(ctx, `
UPDATE flashcards
SET due_at = ?, interval_days = ?, ease_factor = ?, times_reviewed = ?, times_correct = ?
WHERE id = ?
`, c.DueAt, c.IntervalDays, c.EaseFactor, c.TimesReviewed, c.TimesCorrect, c.ID)
	if err != nil {
		log.Error("failed to update flashcard: %v", err)
	}
	return err
}

func (db *DB) NextFlashcards(ctx context.Context, profileID int64, limit int) ([]models.Flashcard, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching next flashcards: profile_id=%d, limit=%d", profileID, limit)

	rows, err := db.QueryContext(ctx, `
SELECT id, position_id, due_at, interval_days, ease_factor, times_reviewed, times_correct, created_at
FROM flashcards
WHERE due_at <= CURRENT_TIMESTAMP
AND position_id IN (
    SELECT p.id FROM positions p
    JOIN games g ON g.id = p.game_id
    WHERE g.profile_id = ?
)
ORDER BY RANDOM()
LIMIT ?
`, profileID, limit)
	if err != nil {
		log.Error("failed to query flashcards: %v", err)
		return nil, err
	}
	defer rows.Close()
	var cards []models.Flashcard
	for rows.Next() {
		var c models.Flashcard
		if err := rows.Scan(&c.ID, &c.PositionID, &c.DueAt, &c.IntervalDays, &c.EaseFactor, &c.TimesReviewed, &c.TimesCorrect, &c.CreatedAt); err != nil {
			log.Error("failed to scan flashcard row: %v", err)
			return nil, err
		}
		cards = append(cards, c)
	}
	log.Debug("found %d due flashcards", len(cards))
	return cards, rows.Err()
}

func (db *DB) OpeningStats(ctx context.Context, profileID int64) ([]models.OpeningStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching opening stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT opening_name, eco_code, total_games, wins, draws, losses, win_rate, avg_blunders
FROM opening_stats
WHERE profile_id = ?
ORDER BY total_games DESC
LIMIT 100
`, profileID)
	if err != nil {
		log.Error("failed to query opening stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.OpeningStat
	for rows.Next() {
		var s models.OpeningStat
		if err := rows.Scan(&s.OpeningName, &s.ECOCode, &s.TotalGames, &s.Wins, &s.Draws, &s.Losses, &s.WinRate, &s.AvgBlunders); err != nil {
			log.Error("failed to scan opening stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	log.Debug("found %d opening stats", len(stats))
	return stats, rows.Err()
}

func (db *DB) FlashcardWithPosition(ctx context.Context, id int64, profileID int64) (*models.FlashcardWithPosition, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard with position: id=%d, profile_id=%d", id, profileID)

	var fp models.FlashcardWithPosition
	err := db.QueryRowContext(ctx, `
SELECT 
    f.id, f.position_id, f.due_at, f.interval_days, f.ease_factor, f.times_reviewed, f.times_correct, f.created_at,
    p.game_id, p.move_number, p.fen, p.move_played, p.best_move, p.eval_before, p.eval_after, p.eval_diff, p.mate_before, p.mate_after, p.classification,
    CASE WHEN g.played_as = 'white' THEN pr.username ELSE g.opponent END AS white_player,
    CASE WHEN g.played_as = 'black' THEN pr.username ELSE g.opponent END AS black_player
FROM flashcards f
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
JOIN profiles pr ON pr.id = g.profile_id
WHERE f.id = ? AND g.profile_id = ?
`, id, profileID).Scan(&fp.ID, &fp.PositionID, &fp.DueAt, &fp.IntervalDays, &fp.EaseFactor, &fp.TimesReviewed, &fp.TimesCorrect, &fp.CreatedAt,
		&fp.GameID, &fp.MoveNumber, &fp.FEN, &fp.MovePlayed, &fp.BestMove, &fp.EvalBefore, &fp.EvalAfter, &fp.EvalDiff, &fp.MateBefore, &fp.MateAfter, &fp.Classification,
		&fp.WhitePlayer, &fp.BlackPlayer)
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("flashcard not found: id=%d", id)
		return nil, nil
	}
	if err != nil {
		log.Error("failed to get flashcard with position: %v", err)
		return nil, err
	}
	log.Debug("flashcard found: position_id=%d, classification=%s", fp.PositionID, fp.Classification)
	return &fp, nil
}
