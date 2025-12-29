package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

type flashcardRepository struct {
	db *sql.DB
}

// NewFlashcardRepository creates a new FlashcardRepository implementation
func NewFlashcardRepository(db *sql.DB) repository.FlashcardRepository {
	return &flashcardRepository{db: db}
}

func (r *flashcardRepository) Insert(ctx context.Context, c models.Flashcard) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("flashcard_repo")
	log.Debug("inserting flashcard: position_id=%d", c.PositionID)

	res, err := r.db.ExecContext(ctx, `
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

func (r *flashcardRepository) Update(ctx context.Context, c models.Flashcard) error {
	log := logger.FromContext(ctx).WithPrefix("flashcard_repo")
	log.Debug("updating flashcard: id=%d, interval=%d, ease=%.2f", c.ID, c.IntervalDays, c.EaseFactor)

	_, err := r.db.ExecContext(ctx, `
UPDATE flashcards
SET due_at = ?, interval_days = ?, ease_factor = ?, times_reviewed = ?, times_correct = ?
WHERE id = ?
`, c.DueAt, c.IntervalDays, c.EaseFactor, c.TimesReviewed, c.TimesCorrect, c.ID)
	if err != nil {
		log.Error("failed to update flashcard: %v", err)
	}
	return err
}

func (r *flashcardRepository) NextFlashcards(ctx context.Context, profileID int64, limit int) ([]models.Flashcard, error) {
	log := logger.FromContext(ctx).WithPrefix("flashcard_repo")
	log.Debug("fetching next flashcards: profile_id=%d, limit=%d", profileID, limit)

	rows, err := r.db.QueryContext(ctx, `
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

func (r *flashcardRepository) FlashcardWithPosition(ctx context.Context, id int64, profileID int64) (*models.FlashcardWithPosition, error) {
	log := logger.FromContext(ctx).WithPrefix("flashcard_repo")
	log.Debug("fetching flashcard with position: id=%d, profile_id=%d", id, profileID)

	var fp models.FlashcardWithPosition
	var prevMovePlayed sql.NullString
	var playerRating, opponentRating sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
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
		&playerRating, &opponentRating, &fp.PlayedAt, &fp.TimeClass)
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
	if playerRating.Valid {
		fp.PlayerRating = int(playerRating.Int64)
	}
	if opponentRating.Valid {
		fp.OpponentRating = int(opponentRating.Int64)
	}
	log.Debug("flashcard found: position_id=%d, classification=%s", fp.PositionID, fp.Classification)
	return &fp, nil
}

func (r *flashcardRepository) InsertReviewHistory(ctx context.Context, flashcardID int64, quality int, timeSeconds float64) error {
	log := logger.FromContext(ctx).WithPrefix("flashcard_repo")
	log.Debug("inserting review history: flashcard_id=%d, quality=%d, time=%.2fs", flashcardID, quality, timeSeconds)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO review_history (flashcard_id, quality, time_seconds)
		VALUES (?, ?, ?)
	`, flashcardID, quality, timeSeconds)
	if err != nil {
		log.Error("failed to insert review history: %v", err)
	}
	return err
}
