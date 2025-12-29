package db

import (
	"context"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

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
