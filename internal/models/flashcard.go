package models

import "time"

type Flashcard struct {
	ID            int64     `json:"id"`
	PositionID    int64     `json:"position_id"`
	DueAt         time.Time `json:"due_at"`
	IntervalDays  int       `json:"interval_days"`
	EaseFactor    float64   `json:"ease_factor"`
	TimesReviewed int       `json:"times_reviewed"`
	TimesCorrect  int       `json:"times_correct"`
	CreatedAt     time.Time `json:"created_at"`
}

type FlashcardWithPosition struct {
	Flashcard
	GameID         int64     `json:"game_id"`
	MoveNumber     int       `json:"move_number"`
	FEN            string    `json:"fen"`
	MovePlayed     string    `json:"move_played"`
	PrevMovePlayed string    `json:"prev_move_played"`
	BestMove       string    `json:"best_move"`
	EvalBefore     float64   `json:"eval_before"`
	EvalAfter      float64   `json:"eval_after"`
	EvalDiff       float64   `json:"eval_diff"`
	MateBefore     *int      `json:"mate_before"`
	MateAfter      *int      `json:"mate_after"`
	Classification string    `json:"classification"`
	WhitePlayer    string    `json:"white_player"`
	BlackPlayer    string    `json:"black_player"`
	PlayerRating   int       `json:"player_rating"`
	OpponentRating int       `json:"opponent_rating"`
	PlayedAt       time.Time `json:"played_at"`
	TimeClass      string    `json:"time_class"`
}

type ReviewHistory struct {
	ID          int64     `json:"id"`
	FlashcardID int64     `json:"flashcard_id"`
	Quality     int       `json:"quality"`
	TimeSeconds float64   `json:"time_seconds"`
	ReviewedAt  time.Time `json:"reviewed_at"`
}
