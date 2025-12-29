package models

import "time"

type Position struct {
	ID             int64     `json:"id"`
	GameID         int64     `json:"game_id"`
	MoveNumber     int       `json:"move_number"`
	FEN            string    `json:"fen"`
	MovePlayed     string    `json:"move_played"`
	BestMove       string    `json:"best_move"`
	EvalBefore     float64   `json:"eval_before"`
	EvalAfter      float64   `json:"eval_after"`
	EvalDiff       float64   `json:"eval_diff"`
	MateBefore     *int      `json:"mate_before"`
	MateAfter      *int      `json:"mate_after"`
	Classification string    `json:"classification"`
	CreatedAt      time.Time `json:"created_at"`
}
