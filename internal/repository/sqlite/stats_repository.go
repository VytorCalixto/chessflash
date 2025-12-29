package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

type statsRepository struct {
	db *sql.DB
}

// NewStatsRepository creates a new StatsRepository implementation
func NewStatsRepository(db *sql.DB) repository.StatsRepository {
	return &statsRepository{db: db}
}

func (r *statsRepository) OpeningStats(ctx context.Context, profileID int64, limit, offset int) ([]models.OpeningStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching opening stats: profile_id=%d, limit=%d, offset=%d", profileID, limit, offset)

	rows, err := r.db.QueryContext(ctx, `
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

func (r *statsRepository) CountOpeningStats(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("counting opening stats: profile_id=%d", profileID)

	var count int
	err := r.db.QueryRowContext(ctx, `
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

func (r *statsRepository) OpponentStats(ctx context.Context, profileID int64, limit, offset int, orderBy, orderDir string) ([]models.OpponentStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching opponent stats: profile_id=%d, limit=%d, offset=%d, order_by=%s, order_dir=%s", profileID, limit, offset, orderBy, orderDir)

	// Validate and set default orderBy
	if orderBy != "total_games" && orderBy != "last_played_at" {
		orderBy = "total_games"
	}
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	rows, err := r.db.QueryContext(ctx, `
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

func (r *statsRepository) CountOpponentStats(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("counting opponent stats: profile_id=%d", profileID)

	var count int
	err := r.db.QueryRowContext(ctx, `
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

func (r *statsRepository) TimeClassStats(ctx context.Context, profileID int64, dateCutoff *time.Time) ([]models.TimeClassStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching time class stats: profile_id=%d, date_cutoff=%v", profileID, dateCutoff)

	var rows *sql.Rows
	var err error

	// If date filtering, query from games directly
	if dateCutoff != nil {
		rows, err = r.db.QueryContext(ctx, `
SELECT g.time_class,
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
WHERE g.profile_id = ? AND g.played_at >= ?
GROUP BY g.time_class
ORDER BY total_games DESC
`, profileID, dateCutoff)
	} else {
		// Use cache when not filtering
		rows, err = r.db.QueryContext(ctx, `
SELECT time_class, total_games, wins, draws, losses, win_rate, avg_blunders, avg_game_length
FROM time_class_stats_cache
WHERE profile_id = ?
ORDER BY total_games DESC
`, profileID)
	}

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

func (r *statsRepository) ColorStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.ColorStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching color stats: profile_id=%d, time_class=%s, date_cutoff=%v", profileID, timeClass, dateCutoff)

	var rows *sql.Rows
	var err error

	// If filtering, query from games directly
	if timeClass != "" || dateCutoff != nil {
		query := `
SELECT g.played_as,
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
WHERE g.profile_id = ?`
		args := []any{profileID}

		if timeClass != "" {
			query += " AND g.time_class = ?"
			args = append(args, timeClass)
		}
		if dateCutoff != nil {
			query += " AND g.played_at >= ?"
			args = append(args, dateCutoff)
		}

		query += " GROUP BY g.played_as ORDER BY total_games DESC"

		rows, err = r.db.QueryContext(ctx, query, args...)
	} else {
		// Use cache when not filtering
		rows, err = r.db.QueryContext(ctx, `
SELECT played_as, total_games, wins, draws, losses, win_rate, avg_blunders
FROM color_stats_cache
WHERE profile_id = ?
ORDER BY total_games DESC
`, profileID)
	}

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

func (r *statsRepository) MonthlyStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.MonthlyStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching monthly stats: profile_id=%d, time_class=%s, date_cutoff=%v", profileID, timeClass, dateCutoff)

	var rows *sql.Rows
	var err error

	// If filtering, query from games directly
	if timeClass != "" || dateCutoff != nil {
		query := `
SELECT strftime('%Y-%m', g.played_at) AS year_month,
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
WHERE g.profile_id = ?`
		args := []any{profileID}

		if timeClass != "" {
			query += " AND g.time_class = ?"
			args = append(args, timeClass)
		}
		if dateCutoff != nil {
			query += " AND g.played_at >= ?"
			args = append(args, dateCutoff)
		}

		query += " GROUP BY year_month ORDER BY year_month DESC"

		rows, err = r.db.QueryContext(ctx, query, args...)
	} else {
		// Use cache when not filtering
		rows, err = r.db.QueryContext(ctx, `
SELECT year_month, total_games, wins, draws, losses, win_rate, total_blunders, blunder_rate, avg_rating
FROM monthly_stats_cache
WHERE profile_id = ?
ORDER BY year_month DESC
`, profileID)
	}

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

func (r *statsRepository) MistakePhaseStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.MistakePhaseStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching mistake phase stats: profile_id=%d, time_class=%s, date_cutoff=%v", profileID, timeClass, dateCutoff)

	var rows *sql.Rows
	var err error

	// If filtering, query from positions/games directly
	if timeClass != "" || dateCutoff != nil {
		query := `
SELECT CASE
           WHEN p.move_number <= 15 THEN 'opening'
           WHEN p.move_number <= 35 THEN 'middlegame'
           ELSE 'endgame'
       END AS phase,
       p.classification,
       COUNT(*) AS count,
       AVG(CASE WHEN p.eval_diff < 0 THEN -p.eval_diff ELSE 0 END) AS avg_eval_loss
FROM positions p
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?`
		args := []any{profileID}

		if timeClass != "" {
			query += " AND g.time_class = ?"
			args = append(args, timeClass)
		}
		if dateCutoff != nil {
			query += " AND g.played_at >= ?"
			args = append(args, dateCutoff)
		}

		query += " GROUP BY phase, p.classification ORDER BY phase, p.classification"

		rows, err = r.db.QueryContext(ctx, query, args...)
	} else {
		// Use cache when not filtering
		rows, err = r.db.QueryContext(ctx, `
SELECT phase, classification, count, avg_eval_loss
FROM mistake_phase_cache
WHERE profile_id = ?
ORDER BY phase, classification
`, profileID)
	}

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

func (r *statsRepository) RatingStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) ([]models.RatingStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching rating stats: profile_id=%d, time_class=%s, date_cutoff=%v", profileID, timeClass, dateCutoff)

	var rows *sql.Rows
	var err error

	// If filtering, query from games directly with complex subqueries
	if timeClass != "" || dateCutoff != nil {
		// Build WHERE clause for main query
		whereClause := "WHERE g.profile_id = ? AND g.player_rating IS NOT NULL"
		args := []any{profileID}

		if timeClass != "" {
			whereClause += " AND g.time_class = ?"
			args = append(args, timeClass)
		}
		if dateCutoff != nil {
			whereClause += " AND g.played_at >= ?"
			args = append(args, dateCutoff)
		}

		// Build WHERE clause for subqueries (they reference outer query's time_class)
		subWhereClause := "WHERE g2.profile_id = ? AND g2.time_class = g.time_class AND g2.player_rating IS NOT NULL"
		if dateCutoff != nil {
			subWhereClause += " AND g2.played_at >= ?"
		}

		subWhereClauseAsc := "WHERE g3.profile_id = ? AND g3.time_class = g.time_class AND g3.player_rating IS NOT NULL"
		if dateCutoff != nil {
			subWhereClauseAsc += " AND g3.played_at >= ?"
		}

		query := `
SELECT g.time_class,
       MIN(g.player_rating) AS min_rating,
       MAX(g.player_rating) AS max_rating,
       AVG(g.player_rating) AS avg_rating,
       (
           SELECT g2.player_rating
           FROM games g2
           ` + subWhereClause + `
           ORDER BY g2.played_at DESC
           LIMIT 1
       ) AS current_rating,
       (
           COALESCE((
               SELECT g2.player_rating
               FROM games g2
               ` + subWhereClause + `
               ORDER BY g2.played_at DESC
               LIMIT 1
           ), 0) -
           COALESCE((
               SELECT g3.player_rating
               FROM games g3
               ` + subWhereClauseAsc + `
               ORDER BY g3.played_at ASC
               LIMIT 1
           ), 0)
       ) AS rating_change,
       COUNT(g.player_rating) AS games_tracked
FROM games g
` + whereClause + `
GROUP BY g.time_class
ORDER BY g.time_class`

		// Build args array - subqueries need profileID and optionally dateCutoff
		// SQLite will use the outer query's g.time_class, so we don't need to pass it
		finalArgs := make([]any, 0)
		finalArgs = append(finalArgs, args...) // Main query args
		// Subquery 1 args (for current_rating)
		finalArgs = append(finalArgs, profileID)
		if dateCutoff != nil {
			finalArgs = append(finalArgs, dateCutoff)
		}
		// Subquery 2 args (for rating_change - first subquery)
		finalArgs = append(finalArgs, profileID)
		if dateCutoff != nil {
			finalArgs = append(finalArgs, dateCutoff)
		}
		// Subquery 3 args (for rating_change - second subquery)
		finalArgs = append(finalArgs, profileID)
		if dateCutoff != nil {
			finalArgs = append(finalArgs, dateCutoff)
		}

		rows, err = r.db.QueryContext(ctx, query, finalArgs...)
	} else {
		// Use cache when not filtering
		rows, err = r.db.QueryContext(ctx, `
SELECT time_class, min_rating, max_rating, avg_rating, current_rating, rating_change, games_tracked
FROM rating_stats_cache
WHERE profile_id = ?
ORDER BY time_class
`, profileID)
	}

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

func (r *statsRepository) FlashcardStats(ctx context.Context, profileID int64) (*models.FlashcardStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching flashcard stats: profile_id=%d", profileID)

	var stat models.FlashcardStat
	err := r.db.QueryRowContext(ctx, `
SELECT 
    COUNT(DISTINCT f.id) AS total_cards,
    COALESCE(SUM(f.times_reviewed), 0) AS total_reviews,
    COUNT(DISTINCT CASE WHEN f.ease_factor > 2.5 AND f.interval_days > 30 THEN f.id END) AS cards_mastered,
    COUNT(DISTINCT CASE WHEN f.ease_factor < 2.0 AND f.times_reviewed > 3 THEN f.id END) AS cards_struggling,
    COUNT(DISTINCT CASE WHEN f.due_at <= CURRENT_TIMESTAMP THEN f.id END) AS cards_due,
    COUNT(DISTINCT CASE WHEN f.due_at <= datetime('now', '+7 days') AND f.due_at > CURRENT_TIMESTAMP THEN f.id END) AS cards_due_soon,
    CASE 
        WHEN SUM(f.times_reviewed) > 0 
        THEN ROUND(100.0 * SUM(f.times_correct) / SUM(f.times_reviewed), 1)
        ELSE 0 
    END AS overall_accuracy,
    COALESCE(AVG(f.ease_factor), 0) AS avg_ease_factor,
    COALESCE(AVG(f.interval_days), 0) AS avg_interval_days
FROM flashcards f
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
`, profileID).Scan(
		&stat.TotalCards,
		&stat.TotalReviews,
		&stat.CardsMastered,
		&stat.CardsStruggling,
		&stat.CardsDue,
		&stat.CardsDueSoon,
		&stat.OverallAccuracy,
		&stat.AvgEaseFactor,
		&stat.AvgIntervalDays,
	)
	if err != nil {
		log.Error("failed to get flashcard stats: %v", err)
		return nil, err
	}
	return &stat, nil
}

func (r *statsRepository) FlashcardClassificationStats(ctx context.Context, profileID int64) ([]models.FlashcardClassificationStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching flashcard classification stats: profile_id=%d", profileID)

	rows, err := r.db.QueryContext(ctx, `
SELECT 
    p.classification,
    COUNT(DISTINCT f.id) AS total_cards,
    COALESCE(SUM(f.times_reviewed), 0) AS total_reviews,
    CASE 
        WHEN SUM(f.times_reviewed) > 0 
        THEN ROUND(100.0 * SUM(f.times_correct) / SUM(f.times_reviewed), 1)
        ELSE 0 
    END AS avg_accuracy,
    COALESCE(AVG(f.ease_factor), 0) AS avg_ease_factor,
    COALESCE(AVG(f.times_reviewed), 0) AS avg_reviews_needed
FROM flashcards f
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
GROUP BY p.classification
ORDER BY total_cards DESC
`, profileID)
	if err != nil {
		log.Error("failed to query classification stats: %v", err)
		return nil, err
	}
	defer rows.Close()

	var stats []models.FlashcardClassificationStat
	for rows.Next() {
		var s models.FlashcardClassificationStat
		if err := rows.Scan(&s.Classification, &s.TotalCards, &s.TotalReviews, &s.AvgAccuracy, &s.AvgEaseFactor, &s.AvgReviewsNeeded); err != nil {
			log.Error("failed to scan classification stat: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *statsRepository) FlashcardPhaseStats(ctx context.Context, profileID int64) ([]models.FlashcardPhaseStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching flashcard phase stats: profile_id=%d", profileID)

	rows, err := r.db.QueryContext(ctx, `
SELECT 
    CASE
        WHEN p.move_number <= 15 THEN 'opening'
        WHEN p.move_number <= 35 THEN 'middlegame'
        ELSE 'endgame'
    END AS phase,
    COUNT(DISTINCT f.id) AS total_cards,
    COALESCE(SUM(f.times_reviewed), 0) AS total_reviews,
    CASE 
        WHEN SUM(f.times_reviewed) > 0 
        THEN ROUND(100.0 * SUM(f.times_correct) / SUM(f.times_reviewed), 1)
        ELSE 0 
    END AS avg_accuracy,
    COALESCE(AVG(f.ease_factor), 0) AS avg_ease_factor
FROM flashcards f
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
GROUP BY phase
ORDER BY phase
`, profileID)
	if err != nil {
		log.Error("failed to query phase stats: %v", err)
		return nil, err
	}
	defer rows.Close()

	var stats []models.FlashcardPhaseStat
	for rows.Next() {
		var s models.FlashcardPhaseStat
		if err := rows.Scan(&s.Phase, &s.TotalCards, &s.TotalReviews, &s.AvgAccuracy, &s.AvgEaseFactor); err != nil {
			log.Error("failed to scan phase stat: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *statsRepository) FlashcardOpeningStats(ctx context.Context, profileID int64, limit int) ([]models.FlashcardOpeningStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching flashcard opening stats: profile_id=%d", profileID)

	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT 
    COALESCE(g.opening_name, 'Unknown') AS opening_name,
    COALESCE(g.eco_code, '') AS eco_code,
    COUNT(DISTINCT f.id) AS total_cards,
    COALESCE(SUM(f.times_reviewed), 0) AS total_reviews,
    CASE 
        WHEN SUM(f.times_reviewed) > 0 
        THEN ROUND(100.0 * SUM(f.times_correct) / SUM(f.times_reviewed), 1)
        ELSE 0 
    END AS avg_accuracy,
    COALESCE(AVG(f.ease_factor), 0) AS avg_ease_factor
FROM flashcards f
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ? AND g.opening_name IS NOT NULL AND g.opening_name != ''
GROUP BY g.opening_name, g.eco_code
ORDER BY total_cards DESC
LIMIT ?
`, profileID, limit)
	if err != nil {
		log.Error("failed to query opening stats: %v", err)
		return nil, err
	}
	defer rows.Close()

	var stats []models.FlashcardOpeningStat
	for rows.Next() {
		var s models.FlashcardOpeningStat
		if err := rows.Scan(&s.OpeningName, &s.ECOCode, &s.TotalCards, &s.TotalReviews, &s.AvgAccuracy, &s.AvgEaseFactor); err != nil {
			log.Error("failed to scan opening stat: %v", err)
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *statsRepository) FlashcardTimeStats(ctx context.Context, profileID int64) (*models.FlashcardTimeStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching flashcard time stats: profile_id=%d", profileID)

	var avgTime, fastestTime, slowestTime float64
	err := r.db.QueryRowContext(ctx, `
SELECT 
    COALESCE(AVG(rh.time_seconds), 0) AS avg_time_seconds,
    COALESCE(MIN(rh.time_seconds), 0) AS fastest_time,
    COALESCE(MAX(rh.time_seconds), 0) AS slowest_time
FROM review_history rh
JOIN flashcards f ON f.id = rh.flashcard_id
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
`, profileID).Scan(&avgTime, &fastestTime, &slowestTime)
	if err != nil {
		log.Error("failed to get time stats: %v", err)
		return nil, err
	}

	// Calculate median separately (simpler approach)
	var medianTime float64
	var count int
	err = r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM review_history rh
JOIN flashcards f ON f.id = rh.flashcard_id
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
`, profileID).Scan(&count)
	if err == nil && count > 0 {
		// Get median by ordering and taking middle value
		offset := count / 2
		err = r.db.QueryRowContext(ctx, `
SELECT time_seconds FROM review_history rh
JOIN flashcards f ON f.id = rh.flashcard_id
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
ORDER BY rh.time_seconds
LIMIT 1 OFFSET ?
`, profileID, offset).Scan(&medianTime)
		if err != nil {
			medianTime = avgTime // Fallback to average if median fails
		}
	}

	// Get time by quality
	rows, err := r.db.QueryContext(ctx, `
SELECT 
    rh.quality,
    COALESCE(AVG(rh.time_seconds), 0) AS avg_time
FROM review_history rh
JOIN flashcards f ON f.id = rh.flashcard_id
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
GROUP BY rh.quality
`, profileID)
	if err != nil {
		log.Error("failed to query time by quality: %v", err)
		return nil, err
	}
	defer rows.Close()

	timeByQuality := make(map[int]float64)
	for rows.Next() {
		var quality int
		var avgTime float64
		if err := rows.Scan(&quality, &avgTime); err != nil {
			log.Error("failed to scan time by quality: %v", err)
			continue
		}
		timeByQuality[quality] = avgTime
	}

	return &models.FlashcardTimeStat{
		AvgTimeSeconds:    avgTime,
		MedianTimeSeconds: medianTime,
		FastestTime:       fastestTime,
		SlowestTime:       slowestTime,
		TimeByQuality:     timeByQuality,
	}, rows.Err()
}

func (r *statsRepository) SummaryStats(ctx context.Context, profileID int64, timeClass string, dateCutoff *time.Time) (*models.SummaryStat, error) {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("fetching summary stats: profile_id=%d, time_class=%s, date_cutoff=%v", profileID, timeClass, dateCutoff)

	var stat models.SummaryStat

	// If filtering, query from games table directly
	if dateCutoff != nil || timeClass != "" {
		gameQuery := `
SELECT 
    COUNT(*) AS total_games,
    CASE 
        WHEN COUNT(*) > 0 
        THEN ROUND(100.0 * SUM(CASE WHEN result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1)
        ELSE 0 
    END AS overall_win_rate
FROM games
WHERE profile_id = ?`
		gameArgs := []any{profileID}

		if timeClass != "" {
			gameQuery += " AND time_class = ?"
			gameArgs = append(gameArgs, timeClass)
		}
		if dateCutoff != nil {
			gameQuery += " AND played_at >= ?"
			gameArgs = append(gameArgs, dateCutoff)
		}

		err := r.db.QueryRowContext(ctx, gameQuery, gameArgs...).Scan(&stat.TotalGames, &stat.OverallWinRate)
		if err != nil {
			log.Error("failed to get games/wins with filters: %v", err)
			return nil, err
		}

		// Get current highest rating from games
		ratingQuery := `
SELECT COALESCE(MAX(player_rating), 0)
FROM games
WHERE profile_id = ? AND player_rating IS NOT NULL`
		ratingArgs := []any{profileID}
		if timeClass != "" {
			ratingQuery += " AND time_class = ?"
			ratingArgs = append(ratingArgs, timeClass)
		}
		if dateCutoff != nil {
			ratingQuery += " AND played_at >= ?"
			ratingArgs = append(ratingArgs, dateCutoff)
		}

		err = r.db.QueryRowContext(ctx, ratingQuery, ratingArgs...).Scan(&stat.CurrentRating)
		if err != nil {
			log.Error("failed to get current rating with filters: %v", err)
			return nil, err
		}

		// Get total blunders from positions
		blunderQuery := `
SELECT COALESCE(COUNT(*), 0)
FROM positions p
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ? AND p.classification = 'blunder'`
		blunderArgs := []any{profileID}
		if timeClass != "" {
			blunderQuery += " AND g.time_class = ?"
			blunderArgs = append(blunderArgs, timeClass)
		}
		if dateCutoff != nil {
			blunderQuery += " AND g.played_at >= ?"
			blunderArgs = append(blunderArgs, dateCutoff)
		}

		err = r.db.QueryRowContext(ctx, blunderQuery, blunderArgs...).Scan(&stat.TotalBlunders)
		if err != nil {
			log.Error("failed to get total blunders with filters: %v", err)
			return nil, err
		}
	} else {
		// Use cache when not filtering
		err := r.db.QueryRowContext(ctx, `
SELECT 
    COALESCE(SUM(total_games), 0) AS total_games,
    CASE 
        WHEN SUM(total_games) > 0 
        THEN ROUND(100.0 * SUM(wins) / SUM(total_games), 1)
        ELSE 0 
    END AS overall_win_rate
FROM time_class_stats_cache
WHERE profile_id = ?
`, profileID).Scan(&stat.TotalGames, &stat.OverallWinRate)
		if err != nil {
			log.Error("failed to get games/wins from summary stats: %v", err)
			return nil, err
		}

		err = r.db.QueryRowContext(ctx, `
SELECT COALESCE(MAX(current_rating), 0)
FROM rating_stats_cache
WHERE profile_id = ?
`, profileID).Scan(&stat.CurrentRating)
		if err != nil {
			log.Error("failed to get current rating from summary stats: %v", err)
			return nil, err
		}

		err = r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(total_blunders), 0)
FROM monthly_stats_cache
WHERE profile_id = ?
`, profileID).Scan(&stat.TotalBlunders)
		if err != nil {
			log.Error("failed to get total blunders from summary stats: %v", err)
			return nil, err
		}
	}

	// Calculate average blunders per game
	if stat.TotalGames > 0 {
		stat.AvgBlundersPerGame = float64(stat.TotalBlunders) / float64(stat.TotalGames)
	} else {
		stat.AvgBlundersPerGame = 0
	}

	return &stat, nil
}

func (r *statsRepository) RefreshProfileStats(ctx context.Context, profileID int64) error {
	log := logger.FromContext(ctx).WithPrefix("stats_repo")
	log.Debug("refreshing cached stats: profile_id=%d", profileID)

	if err := r.refreshOpeningStats(ctx, profileID); err != nil {
		return err
	}
	if err := r.refreshOpponentStats(ctx, profileID); err != nil {
		return err
	}
	if err := r.refreshTimeClassStats(ctx, profileID); err != nil {
		return err
	}
	if err := r.refreshColorStats(ctx, profileID); err != nil {
		return err
	}
	if err := r.refreshMonthlyStats(ctx, profileID); err != nil {
		return err
	}
	if err := r.refreshMistakePhaseStats(ctx, profileID); err != nil {
		return err
	}
	if err := r.refreshRatingStats(ctx, profileID); err != nil {
		return err
	}
	return nil
}

func (r *statsRepository) refreshOpeningStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}

func (r *statsRepository) refreshOpponentStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}

func (r *statsRepository) refreshTimeClassStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}

func (r *statsRepository) refreshColorStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}

func (r *statsRepository) refreshMonthlyStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}

func (r *statsRepository) refreshMistakePhaseStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}

func (r *statsRepository) refreshRatingStats(ctx context.Context, profileID int64) error {
	return tx(ctx, r.db, func(tx *sql.Tx) error {
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
	})
}
