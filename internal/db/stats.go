package db

import (
	"context"
	"database/sql"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

func (db *DB) OpeningStats(ctx context.Context, profileID int64, limit, offset int) ([]models.OpeningStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching opening stats: profile_id=%d, limit=%d, offset=%d", profileID, limit, offset)

	rows, err := db.QueryContext(ctx, `
SELECT opening_name, eco_code, total_games, wins, draws, losses, win_rate, avg_blunders
FROM opening_stats_cache
WHERE profile_id = ?
ORDER BY total_games DESC
LIMIT ? OFFSET ?
`, profileID, limit, offset)
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

func (db *DB) CountOpeningStats(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("counting opening stats: profile_id=%d", profileID)

	var count int
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM opening_stats_cache
WHERE profile_id = ?
`, profileID).Scan(&count)
	if err != nil {
		log.Error("failed to count opening stats: %v", err)
		return 0, err
	}
	return count, nil
}

// RefreshProfileStats recomputes all cached stats for a profile.
func (db *DB) RefreshProfileStats(ctx context.Context, profileID int64) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("refreshing cached stats: profile_id=%d", profileID)

	return tx(ctx, db, func(tx *sql.Tx) error {
		if err := db.refreshOpeningStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		if err := db.refreshOpponentStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		if err := db.refreshTimeClassStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		if err := db.refreshColorStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		if err := db.refreshMonthlyStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		if err := db.refreshMistakePhaseStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		if err := db.refreshRatingStatsTx(ctx, tx, profileID); err != nil {
			return err
		}
		return nil
	})
}

func (db *DB) refreshOpeningStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM opening_stats_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO opening_stats_cache (profile_id, opening_name, eco_code, total_games, wins, draws, losses, win_rate, avg_blunders)
SELECT g.profile_id,
       g.opening_name,
       MAX(g.eco_code) AS eco_code,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(AVG(COALESCE(b.blunder_count, 0)), 0) AS avg_blunders
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
WHERE g.profile_id = ? AND g.opening_name IS NOT NULL AND g.opening_name != ''
GROUP BY g.profile_id, g.opening_name
`, profileID)
	return err
}

func (db *DB) refreshOpponentStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM opponent_stats_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO opponent_stats_cache (profile_id, opponent, total_games, wins, draws, losses, win_rate, avg_opponent_rating, last_played_at)
SELECT g.profile_id,
       g.opponent,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       AVG(g.opponent_rating) AS avg_opponent_rating,
       MAX(g.played_at) AS last_played_at
FROM games g
WHERE g.profile_id = ?
GROUP BY g.profile_id, g.opponent
`, profileID)
	return err
}

func (db *DB) refreshTimeClassStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM time_class_stats_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO time_class_stats_cache (profile_id, time_class, total_games, wins, draws, losses, win_rate, avg_blunders, avg_game_length)
SELECT g.profile_id,
       g.time_class,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(AVG(COALESCE(b.blunder_count, 0)), 0) AS avg_blunders,
       COALESCE(AVG(COALESCE(m.moves_played, 0)), 0) AS avg_game_length
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
LEFT JOIN (
    SELECT game_id, MAX(move_number) AS moves_played
    FROM positions
    GROUP BY game_id
) m ON m.game_id = g.id
WHERE g.profile_id = ?
GROUP BY g.profile_id, g.time_class
`, profileID)
	return err
}

func (db *DB) refreshColorStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM color_stats_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO color_stats_cache (profile_id, played_as, total_games, wins, draws, losses, win_rate, avg_blunders)
SELECT g.profile_id,
       g.played_as,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(AVG(COALESCE(b.blunder_count, 0)), 0) AS avg_blunders
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
WHERE g.profile_id = ?
GROUP BY g.profile_id, g.played_as
`, profileID)
	return err
}

func (db *DB) refreshMonthlyStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM monthly_stats_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO monthly_stats_cache (profile_id, year_month, total_games, wins, draws, losses, win_rate, total_blunders, blunder_rate, avg_rating)
SELECT g.profile_id,
       strftime('%Y-%m', g.played_at) AS year_month,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(SUM(COALESCE(b.blunder_count, 0)), 0) AS total_blunders,
       CASE WHEN COUNT(*) > 0 THEN ROUND(1.0 * COALESCE(SUM(COALESCE(b.blunder_count, 0)), 0) / COUNT(*), 3) ELSE 0 END AS blunder_rate,
       AVG(g.player_rating) AS avg_rating
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
WHERE g.profile_id = ?
GROUP BY g.profile_id, year_month
`, profileID)
	return err
}

func (db *DB) refreshMistakePhaseStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM mistake_phase_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO mistake_phase_cache (profile_id, phase, classification, count, avg_eval_loss)
SELECT g.profile_id,
       CASE
           WHEN p.move_number <= 15 THEN 'opening'
           WHEN p.move_number <= 35 THEN 'middlegame'
           ELSE 'endgame'
       END AS phase,
       p.classification,
       COUNT(*) AS count,
       AVG(CASE WHEN p.eval_diff < 0 THEN -p.eval_diff ELSE 0 END) AS avg_eval_loss
FROM positions p
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
GROUP BY g.profile_id, phase, p.classification
`, profileID)
	return err
}

func (db *DB) refreshRatingStatsTx(ctx context.Context, tx *sql.Tx, profileID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM rating_stats_cache WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO rating_stats_cache (profile_id, time_class, min_rating, max_rating, avg_rating, current_rating, rating_change, games_tracked)
SELECT g.profile_id,
       g.time_class,
       MIN(g.player_rating) AS min_rating,
       MAX(g.player_rating) AS max_rating,
       AVG(g.player_rating) AS avg_rating,
       (
           SELECT g2.player_rating
           FROM games g2
           WHERE g2.profile_id = g.profile_id AND g2.time_class = g.time_class AND g2.player_rating IS NOT NULL
           ORDER BY g2.played_at DESC
           LIMIT 1
       ) AS current_rating,
       (
           COALESCE((
               SELECT g2.player_rating
               FROM games g2
               WHERE g2.profile_id = g.profile_id AND g2.time_class = g.time_class AND g2.player_rating IS NOT NULL
               ORDER BY g2.played_at DESC
               LIMIT 1
           ), 0) -
           COALESCE((
               SELECT g3.player_rating
               FROM games g3
               WHERE g3.profile_id = g.profile_id AND g3.time_class = g.time_class AND g3.player_rating IS NOT NULL
               ORDER BY g3.played_at ASC
               LIMIT 1
           ), 0)
       ) AS rating_change,
       COUNT(g.player_rating) AS games_tracked
FROM games g
WHERE g.profile_id = ? AND g.player_rating IS NOT NULL
GROUP BY g.profile_id, g.time_class
`, profileID)
	return err
}

func (db *DB) OpponentStats(ctx context.Context, profileID int64, limit, offset int, orderBy, orderDir string) ([]models.OpponentStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching opponent stats: profile_id=%d, limit=%d, offset=%d, order_by=%s, order_dir=%s", profileID, limit, offset, orderBy, orderDir)

	// Validate and set default orderBy
	if orderBy != "total_games" && orderBy != "last_played_at" {
		orderBy = "total_games"
	}
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	rows, err := db.QueryContext(ctx, `
SELECT opponent, total_games, wins, draws, losses, win_rate, avg_opponent_rating, last_played_at
FROM opponent_stats_cache
WHERE profile_id = ?
ORDER BY `+orderBy+` `+orderDir+`
LIMIT ? OFFSET ?
`, profileID, limit, offset)
	if err != nil {
		log.Error("failed to query opponent stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.OpponentStat
	for rows.Next() {
		var s models.OpponentStat
		if err := rows.Scan(&s.Opponent, &s.TotalGames, &s.Wins, &s.Draws, &s.Losses, &s.WinRate, &s.AvgOpponentRating, &s.LastPlayedAt); err != nil {
			log.Error("failed to scan opponent stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (db *DB) CountOpponentStats(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("counting opponent stats: profile_id=%d", profileID)

	var count int
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM opponent_stats_cache
WHERE profile_id = ?
`, profileID).Scan(&count)
	if err != nil {
		log.Error("failed to count opponent stats: %v", err)
		return 0, err
	}
	return count, nil
}

func (db *DB) TimeClassStats(ctx context.Context, profileID int64) ([]models.TimeClassStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching time class stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT time_class, total_games, wins, draws, losses, win_rate, avg_blunders, avg_game_length
FROM time_class_stats_cache
WHERE profile_id = ?
ORDER BY total_games DESC
`, profileID)
	if err != nil {
		log.Error("failed to query time class stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.TimeClassStat
	for rows.Next() {
		var s models.TimeClassStat
		if err := rows.Scan(&s.TimeClass, &s.TotalGames, &s.Wins, &s.Draws, &s.Losses, &s.WinRate, &s.AvgBlunders, &s.AvgGameLength); err != nil {
			log.Error("failed to scan time class stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (db *DB) ColorStats(ctx context.Context, profileID int64) ([]models.ColorStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching color stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT played_as, total_games, wins, draws, losses, win_rate, avg_blunders
FROM color_stats_cache
WHERE profile_id = ?
ORDER BY total_games DESC
`, profileID)
	if err != nil {
		log.Error("failed to query color stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.ColorStat
	for rows.Next() {
		var s models.ColorStat
		if err := rows.Scan(&s.PlayedAs, &s.TotalGames, &s.Wins, &s.Draws, &s.Losses, &s.WinRate, &s.AvgBlunders); err != nil {
			log.Error("failed to scan color stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (db *DB) MonthlyStats(ctx context.Context, profileID int64) ([]models.MonthlyStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching monthly stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT year_month, total_games, wins, draws, losses, win_rate, total_blunders, blunder_rate, avg_rating
FROM monthly_stats_cache
WHERE profile_id = ?
ORDER BY year_month DESC
`, profileID)
	if err != nil {
		log.Error("failed to query monthly stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.MonthlyStat
	for rows.Next() {
		var s models.MonthlyStat
		if err := rows.Scan(&s.YearMonth, &s.TotalGames, &s.Wins, &s.Draws, &s.Losses, &s.WinRate, &s.TotalBlunders, &s.BlunderRate, &s.AvgRating); err != nil {
			log.Error("failed to scan monthly stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (db *DB) MistakePhaseStats(ctx context.Context, profileID int64) ([]models.MistakePhaseStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching mistake phase stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT phase, classification, count, avg_eval_loss
FROM mistake_phase_cache
WHERE profile_id = ?
ORDER BY phase, classification
`, profileID)
	if err != nil {
		log.Error("failed to query mistake phase stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.MistakePhaseStat
	for rows.Next() {
		var s models.MistakePhaseStat
		if err := rows.Scan(&s.Phase, &s.Classification, &s.Count, &s.AvgEvalLoss); err != nil {
			log.Error("failed to scan mistake phase stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (db *DB) RatingStats(ctx context.Context, profileID int64) ([]models.RatingStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching rating stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT time_class, min_rating, max_rating, avg_rating, current_rating, rating_change, games_tracked
FROM rating_stats_cache
WHERE profile_id = ?
ORDER BY time_class
`, profileID)
	if err != nil {
		log.Error("failed to query rating stats: %v", err)
		return nil, err
	}
	defer rows.Close()
	var stats []models.RatingStat
	for rows.Next() {
		var s models.RatingStat
		if err := rows.Scan(&s.TimeClass, &s.MinRating, &s.MaxRating, &s.AvgRating, &s.CurrentRating, &s.RatingChange, &s.GamesTracked); err != nil {
			log.Error("failed to scan rating stat row: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
