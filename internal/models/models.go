package models

import "time"

type Profile struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	CreatedAt time.Time  `json:"created_at"`
	LastSyncAt *time.Time `json:"last_sync_at"`
}

type Game struct {
	ID             int64     `json:"id"`
	ProfileID      int64     `json:"profile_id"`
	ChessComID     string    `json:"chess_com_id"`
	PGN            string    `json:"pgn"`
	TimeClass      string    `json:"time_class"`
	Result         string    `json:"result"`
	PlayedAs       string    `json:"played_as"`
	Opponent       string    `json:"opponent"`
	PlayedAt       time.Time `json:"played_at"`
	ECOCode        string    `json:"eco_code"`
	OpeningName    string    `json:"opening_name"`
	OpeningURL     string    `json:"opening_url"`
	AnalysisStatus string    `json:"analysis_status"`
	CreatedAt      time.Time `json:"created_at"`
}

type GameFilter struct {
	ProfileID   int64
	TimeClass   string
	Result      string
	OpeningName string
	Limit       int
	Offset      int
	OrderBy     string
	OrderDir    string
}

type Position struct {
	ID            int64     `json:"id"`
	GameID        int64     `json:"game_id"`
	MoveNumber    int       `json:"move_number"`
	FEN           string    `json:"fen"`
	MovePlayed    string    `json:"move_played"`
	BestMove      string    `json:"best_move"`
	EvalBefore    float64   `json:"eval_before"`
	EvalAfter     float64   `json:"eval_after"`
	EvalDiff      float64   `json:"eval_diff"`
	Classification string   `json:"classification"`
	CreatedAt     time.Time `json:"created_at"`
}

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
	GameID         int64   `json:"game_id"`
	MoveNumber     int     `json:"move_number"`
	FEN            string  `json:"fen"`
	MovePlayed     string  `json:"move_played"`
	BestMove       string  `json:"best_move"`
	EvalBefore     float64 `json:"eval_before"`
	EvalAfter      float64 `json:"eval_after"`
	EvalDiff       float64 `json:"eval_diff"`
	Classification string  `json:"classification"`
	WhitePlayer    string  `json:"white_player"`
	BlackPlayer    string  `json:"black_player"`
}

type OpeningStat struct {
	OpeningName string  `json:"opening_name"`
	ECOCode     string  `json:"eco_code"`
	TotalGames  int     `json:"total_games"`
	Wins        int     `json:"wins"`
	Draws       int     `json:"draws"`
	Losses      int     `json:"losses"`
	WinRate     float64 `json:"win_rate"`
	AvgBlunders float64 `json:"avg_blunders"`
}

