package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

func (db *DB) InsertPuzzleRushSession(ctx context.Context, s models.PuzzleRushSession) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("inserting puzzle rush session: profile_id=%d, difficulty=%s", s.ProfileID, s.Difficulty)

	var completedAt interface{}
	if s.CompletedAt != nil {
		completedAt = s.CompletedAt
	}

	res, err := db.ExecContext(ctx, `
INSERT INTO puzzle_rush_sessions (profile_id, difficulty, score, mistakes_made, mistakes_allowed, total_time_seconds, completed_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, s.ProfileID, s.Difficulty, s.Score, s.MistakesMade, s.MistakesAllowed, s.TotalTimeSeconds, completedAt)
	if err != nil {
		log.Error("failed to insert puzzle rush session: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error("failed to get puzzle rush session id: %v", err)
		return 0, err
	}
	log.Debug("puzzle rush session inserted: id=%d", id)
	return id, nil
}

func (db *DB) UpdatePuzzleRushSession(ctx context.Context, s models.PuzzleRushSession) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("updating puzzle rush session: id=%d, score=%d, mistakes=%d", s.ID, s.Score, s.MistakesMade)

	var completedAt interface{}
	if s.CompletedAt != nil {
		completedAt = s.CompletedAt
	}

	_, err := db.ExecContext(ctx, `
UPDATE puzzle_rush_sessions
SET score = ?, mistakes_made = ?, total_time_seconds = ?, completed_at = ?
WHERE id = ?
`, s.Score, s.MistakesMade, s.TotalTimeSeconds, completedAt, s.ID)
	if err != nil {
		log.Error("failed to update puzzle rush session: %v", err)
	}
	return err
}

func (db *DB) GetPuzzleRushSession(ctx context.Context, sessionID int64) (*models.PuzzleRushSession, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching puzzle rush session: id=%d", sessionID)

	var s models.PuzzleRushSession
	var completedAt sql.NullTime
	err := db.QueryRowContext(ctx, `
SELECT id, profile_id, difficulty, score, mistakes_made, mistakes_allowed, total_time_seconds, completed_at, created_at
FROM puzzle_rush_sessions
WHERE id = ?
`, sessionID).Scan(&s.ID, &s.ProfileID, &s.Difficulty, &s.Score, &s.MistakesMade, &s.MistakesAllowed, &s.TotalTimeSeconds, &completedAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("puzzle rush session not found: id=%d", sessionID)
		return nil, nil
	}
	if err != nil {
		log.Error("failed to get puzzle rush session: %v", err)
		return nil, err
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return &s, nil
}

func (db *DB) GetActivePuzzleRushSession(ctx context.Context, profileID int64) (*models.PuzzleRushSession, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching active puzzle rush session: profile_id=%d", profileID)

	var s models.PuzzleRushSession
	var completedAt sql.NullTime
	err := db.QueryRowContext(ctx, `
SELECT id, profile_id, difficulty, score, mistakes_made, mistakes_allowed, total_time_seconds, completed_at, created_at
FROM puzzle_rush_sessions
WHERE profile_id = ? AND completed_at IS NULL
ORDER BY created_at DESC
LIMIT 1
`, profileID).Scan(&s.ID, &s.ProfileID, &s.Difficulty, &s.Score, &s.MistakesMade, &s.MistakesAllowed, &s.TotalTimeSeconds, &completedAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("no active puzzle rush session found: profile_id=%d", profileID)
		return nil, nil
	}
	if err != nil {
		log.Error("failed to get active puzzle rush session: %v", err)
		return nil, err
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return &s, nil
}

func (db *DB) InsertPuzzleRushAttempt(ctx context.Context, a models.PuzzleRushAttempt) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("inserting puzzle rush attempt: session_id=%d, flashcard_id=%d", a.SessionID, a.FlashcardID)

	res, err := db.ExecContext(ctx, `
INSERT INTO puzzle_rush_attempts (session_id, flashcard_id, was_correct, time_seconds, attempt_number)
VALUES (?, ?, ?, ?, ?)
`, a.SessionID, a.FlashcardID, a.WasCorrect, a.TimeSeconds, a.AttemptNumber)
	if err != nil {
		log.Error("failed to insert puzzle rush attempt: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error("failed to get puzzle rush attempt id: %v", err)
		return 0, err
	}
	log.Debug("puzzle rush attempt inserted: id=%d", id)
	return id, nil
}

func (db *DB) GetPuzzleRushSessionAttempts(ctx context.Context, sessionID int64) ([]models.PuzzleRushAttempt, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching puzzle rush attempts: session_id=%d", sessionID)

	rows, err := db.QueryContext(ctx, `
SELECT id, session_id, flashcard_id, was_correct, time_seconds, attempt_number, created_at
FROM puzzle_rush_attempts
WHERE session_id = ?
ORDER BY attempt_number
`, sessionID)
	if err != nil {
		log.Error("failed to query puzzle rush attempts: %v", err)
		return nil, err
	}
	defer rows.Close()

	var attempts []models.PuzzleRushAttempt
	for rows.Next() {
		var a models.PuzzleRushAttempt
		if err := rows.Scan(&a.ID, &a.SessionID, &a.FlashcardID, &a.WasCorrect, &a.TimeSeconds, &a.AttemptNumber, &a.CreatedAt); err != nil {
			log.Error("failed to scan puzzle rush attempt: %v", err)
			return nil, err
		}
		attempts = append(attempts, a)
	}
	log.Debug("found %d attempts", len(attempts))
	return attempts, rows.Err()
}

func (db *DB) GetPuzzleRushUserStats(ctx context.Context, profileID int64) (*models.PuzzleRushStats, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching puzzle rush stats: profile_id=%d", profileID)

	stats := &models.PuzzleRushStats{
		BestScores:   make(map[string]int),
		AverageScores: make(map[string]float64),
	}

	// Get total attempts
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM puzzle_rush_sessions WHERE profile_id = ?
`, profileID).Scan(&stats.TotalAttempts)
	if err != nil {
		log.Error("failed to get total attempts: %v", err)
		return nil, err
	}

	if stats.TotalAttempts == 0 {
		return stats, nil
	}

	// Get best scores per difficulty
	rows, err := db.QueryContext(ctx, `
SELECT difficulty, MAX(score) as best_score
FROM puzzle_rush_sessions
WHERE profile_id = ? AND completed_at IS NOT NULL
GROUP BY difficulty
`, profileID)
	if err != nil {
		log.Error("failed to query best scores: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var difficulty string
		var bestScore int
		if err := rows.Scan(&difficulty, &bestScore); err != nil {
			log.Error("failed to scan best score: %v", err)
			continue
		}
		stats.BestScores[difficulty] = bestScore
	}

	// Get average scores per difficulty
	rows, err = db.QueryContext(ctx, `
SELECT difficulty, AVG(score) as avg_score
FROM puzzle_rush_sessions
WHERE profile_id = ? AND completed_at IS NOT NULL
GROUP BY difficulty
`, profileID)
	if err != nil {
		log.Error("failed to query average scores: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var difficulty string
		var avgScore float64
		if err := rows.Scan(&difficulty, &avgScore); err != nil {
			log.Error("failed to scan average score: %v", err)
			continue
		}
		stats.AverageScores[difficulty] = avgScore
	}

	// Get total correct and mistakes from attempts
	err = db.QueryRowContext(ctx, `
SELECT 
    SUM(CASE WHEN was_correct = 1 THEN 1 ELSE 0 END) as total_correct,
    SUM(CASE WHEN was_correct = 0 THEN 1 ELSE 0 END) as total_mistakes
FROM puzzle_rush_attempts pra
JOIN puzzle_rush_sessions prs ON prs.id = pra.session_id
WHERE prs.profile_id = ?
`, profileID).Scan(&stats.TotalCorrect, &stats.TotalMistakes)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Error("failed to get total correct/mistakes: %v", err)
		return nil, err
	}

	// Get success rate (completed sessions / total sessions)
	var completedCount int
	err = db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM puzzle_rush_sessions
WHERE profile_id = ? AND completed_at IS NOT NULL
`, profileID).Scan(&completedCount)
	if err != nil {
		log.Error("failed to get completed count: %v", err)
		return nil, err
	}
	if stats.TotalAttempts > 0 {
		stats.SuccessRate = float64(completedCount) / float64(stats.TotalAttempts) * 100
	}

	// Get average time
	err = db.QueryRowContext(ctx, `
SELECT AVG(total_time_seconds) FROM puzzle_rush_sessions
WHERE profile_id = ? AND completed_at IS NOT NULL
`, profileID).Scan(&stats.AverageTimeSeconds)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Error("failed to get average time: %v", err)
		return nil, err
	}

	return stats, nil
}

func (db *DB) GetPuzzleRushBestScores(ctx context.Context, profileID int64) ([]models.PuzzleRushBestScore, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("fetching puzzle rush best scores: profile_id=%d", profileID)

	rows, err := db.QueryContext(ctx, `
SELECT difficulty, MAX(score) as best_score, completed_at
FROM puzzle_rush_sessions
WHERE profile_id = ? AND completed_at IS NOT NULL
GROUP BY difficulty
ORDER BY difficulty
`, profileID)
	if err != nil {
		log.Error("failed to query best scores: %v", err)
		return nil, err
	}
	defer rows.Close()

	var bestScores []models.PuzzleRushBestScore
	for rows.Next() {
		var bs models.PuzzleRushBestScore
		var completedAt sql.NullTime
		if err := rows.Scan(&bs.Difficulty, &bs.Score, &completedAt); err != nil {
			log.Error("failed to scan best score: %v", err)
			continue
		}
		if completedAt.Valid {
			bs.CompletedAt = completedAt.Time
		} else {
			bs.CompletedAt = time.Time{}
		}
		bestScores = append(bestScores, bs)
	}
	return bestScores, rows.Err()
}
