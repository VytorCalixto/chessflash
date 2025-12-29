package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

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

func (db *DB) FlashcardWithPosition(ctx context.Context, id int64, profileID int64) (*models.FlashcardWithPosition, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard with position: id=%d, profile_id=%d", id, profileID)

	var fp models.FlashcardWithPosition
	var prevMovePlayed sql.NullString
	err := db.QueryRowContext(ctx, `
SELECT 
    f.id, f.position_id, f.due_at, f.interval_days, f.ease_factor, f.times_reviewed, f.times_correct, f.created_at,
    p.game_id, p.move_number, p.fen, p.move_played, p.best_move, p.eval_before, p.eval_after, p.eval_diff, p.mate_before, p.mate_after, p.classification,
    CASE WHEN g.played_as = 'white' THEN pr.username ELSE g.opponent END AS white_player,
    CASE WHEN g.played_as = 'black' THEN pr.username ELSE g.opponent END AS black_player,
    prev_p.move_played AS prev_move_played,
    g.player_rating, g.opponent_rating, g.played_at, g.time_class
FROM flashcards f
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
JOIN profiles pr ON pr.id = g.profile_id
LEFT JOIN positions prev_p ON prev_p.game_id = p.game_id AND prev_p.move_number = p.move_number - 1
WHERE f.id = ? AND g.profile_id = ?
`, id, profileID).Scan(&fp.ID, &fp.PositionID, &fp.DueAt, &fp.IntervalDays, &fp.EaseFactor, &fp.TimesReviewed, &fp.TimesCorrect, &fp.CreatedAt,
		&fp.GameID, &fp.MoveNumber, &fp.FEN, &fp.MovePlayed, &fp.BestMove, &fp.EvalBefore, &fp.EvalAfter, &fp.EvalDiff, &fp.MateBefore, &fp.MateAfter, &fp.Classification,
		&fp.WhitePlayer, &fp.BlackPlayer, &prevMovePlayed,
		&fp.PlayerRating, &fp.OpponentRating, &fp.PlayedAt, &fp.TimeClass)
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("flashcard not found: id=%d", id)
		return nil, nil
	}
	if err != nil {
		log.Error("failed to get flashcard with position: %v", err)
		return nil, err
	}
	if prevMovePlayed.Valid {
		fp.PrevMovePlayed = prevMovePlayed.String
	}
	log.Debug("flashcard found: position_id=%d, classification=%s", fp.PositionID, fp.Classification)
	return &fp, nil
}

func (db *DB) InsertReviewHistory(ctx context.Context, flashcardID int64, quality int, timeSeconds float64) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("inserting review history: flashcard_id=%d, quality=%d, time=%.2fs", flashcardID, quality, timeSeconds)

	_, err := db.ExecContext(ctx, `
		INSERT INTO review_history (flashcard_id, quality, time_seconds)
		VALUES (?, ?, ?)
	`, flashcardID, quality, timeSeconds)
	if err != nil {
		log.Error("failed to insert review history: %v", err)
	}
	return err
}

func (db *DB) FlashcardStats(ctx context.Context, profileID int64) (*models.FlashcardStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard stats: profile_id=%d", profileID)

	var stat models.FlashcardStat
	err := db.QueryRowContext(ctx, `
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

func (db *DB) FlashcardClassificationStats(ctx context.Context, profileID int64) ([]models.FlashcardClassificationStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard classification stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
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

func (db *DB) FlashcardPhaseStats(ctx context.Context, profileID int64) ([]models.FlashcardPhaseStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard phase stats: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
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

func (db *DB) FlashcardOpeningStats(ctx context.Context, profileID int64, limit int) ([]models.FlashcardOpeningStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard opening stats: profile_id=%d", profileID)

	if limit <= 0 {
		limit = 20
	}

	rows, err := db.QueryContext(ctx, `
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

func (db *DB) FlashcardTimeStats(ctx context.Context, profileID int64) (*models.FlashcardTimeStat, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching flashcard time stats: profile_id=%d", profileID)

	var avgTime, fastestTime, slowestTime float64
	err := db.QueryRowContext(ctx, `
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
	err = db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM review_history rh
JOIN flashcards f ON f.id = rh.flashcard_id
JOIN positions p ON p.id = f.position_id
JOIN games g ON g.id = p.game_id
WHERE g.profile_id = ?
`, profileID).Scan(&count)
	if err == nil && count > 0 {
		// Get median by ordering and taking middle value
		offset := count / 2
		err = db.QueryRowContext(ctx, `
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
	rows, err := db.QueryContext(ctx, `
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
