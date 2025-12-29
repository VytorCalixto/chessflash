package models

import "time"

type Profile struct {
	ID         int64      `json:"id"`
	Username   string     `json:"username"`
	CreatedAt  time.Time  `json:"created_at"`
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
	PlayerRating   int       `json:"player_rating"`
	OpponentRating int       `json:"opponent_rating"`
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
	Opponent    string
	Limit       int
	Offset      int
	OrderBy     string
	OrderDir    string
}

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

type OpponentStat struct {
	Opponent          string    `json:"opponent"`
	TotalGames        int       `json:"total_games"`
	Wins              int       `json:"wins"`
	Draws             int       `json:"draws"`
	Losses            int       `json:"losses"`
	WinRate           float64   `json:"win_rate"`
	AvgOpponentRating float64   `json:"avg_opponent_rating"`
	LastPlayedAt      time.Time `json:"last_played_at"`
}

type TimeClassStat struct {
	TimeClass     string  `json:"time_class"`
	TotalGames    int     `json:"total_games"`
	Wins          int     `json:"wins"`
	Draws         int     `json:"draws"`
	Losses        int     `json:"losses"`
	WinRate       float64 `json:"win_rate"`
	AvgBlunders   float64 `json:"avg_blunders"`
	AvgGameLength float64 `json:"avg_game_length"`
}

type ColorStat struct {
	PlayedAs    string  `json:"played_as"`
	TotalGames  int     `json:"total_games"`
	Wins        int     `json:"wins"`
	Draws       int     `json:"draws"`
	Losses      int     `json:"losses"`
	WinRate     float64 `json:"win_rate"`
	AvgBlunders float64 `json:"avg_blunders"`
}

type MonthlyStat struct {
	YearMonth     string  `json:"year_month"`
	TotalGames    int     `json:"total_games"`
	Wins          int     `json:"wins"`
	Draws         int     `json:"draws"`
	Losses        int     `json:"losses"`
	WinRate       float64 `json:"win_rate"`
	TotalBlunders int     `json:"total_blunders"`
	BlunderRate   float64 `json:"blunder_rate"`
	AvgRating     float64 `json:"avg_rating"`
}

type MistakePhaseStat struct {
	Phase          string  `json:"phase"`
	Classification string  `json:"classification"`
	Count          int     `json:"count"`
	AvgEvalLoss    float64 `json:"avg_eval_loss"`
}

type RatingStat struct {
	TimeClass     string  `json:"time_class"`
	MinRating     int     `json:"min_rating"`
	MaxRating     int     `json:"max_rating"`
	AvgRating     float64 `json:"avg_rating"`
	CurrentRating int     `json:"current_rating"`
	RatingChange  int     `json:"rating_change"`
	GamesTracked  int     `json:"games_tracked"`
}

type ReviewHistory struct {
	ID          int64     `json:"id"`
	FlashcardID int64     `json:"flashcard_id"`
	Quality     int       `json:"quality"`
	TimeSeconds float64   `json:"time_seconds"`
	ReviewedAt  time.Time `json:"reviewed_at"`
}

type FlashcardStat struct {
	TotalCards      int     `json:"total_cards"`
	TotalReviews    int     `json:"total_reviews"`
	CardsMastered   int     `json:"cards_mastered"`
	CardsStruggling int     `json:"cards_struggling"`
	CardsDue        int     `json:"cards_due"`
	CardsDueSoon    int     `json:"cards_due_soon"`
	OverallAccuracy float64 `json:"overall_accuracy"`
	AvgEaseFactor   float64 `json:"avg_ease_factor"`
	AvgIntervalDays float64 `json:"avg_interval_days"`
}

type FlashcardClassificationStat struct {
	Classification   string  `json:"classification"`
	TotalCards       int     `json:"total_cards"`
	TotalReviews     int     `json:"total_reviews"`
	AvgAccuracy      float64 `json:"avg_accuracy"`
	AvgEaseFactor    float64 `json:"avg_ease_factor"`
	AvgReviewsNeeded float64 `json:"avg_reviews_needed"`
}

type FlashcardPhaseStat struct {
	Phase         string  `json:"phase"`
	TotalCards    int     `json:"total_cards"`
	TotalReviews  int     `json:"total_reviews"`
	AvgAccuracy   float64 `json:"avg_accuracy"`
	AvgEaseFactor float64 `json:"avg_ease_factor"`
}

type FlashcardOpeningStat struct {
	OpeningName   string  `json:"opening_name"`
	ECOCode       string  `json:"eco_code"`
	TotalCards    int     `json:"total_cards"`
	TotalReviews  int     `json:"total_reviews"`
	AvgAccuracy   float64 `json:"avg_accuracy"`
	AvgEaseFactor float64 `json:"avg_ease_factor"`
}

type FlashcardTimeStat struct {
	AvgTimeSeconds    float64         `json:"avg_time_seconds"`
	MedianTimeSeconds float64         `json:"median_time_seconds"`
	FastestTime       float64         `json:"fastest_time"`
	SlowestTime       float64         `json:"slowest_time"`
	TimeByQuality     map[int]float64 `json:"time_by_quality"`
}

type SummaryStat struct {
	TotalGames        int     `json:"total_games"`
	OverallWinRate    float64 `json:"overall_win_rate"`
	CurrentRating     int     `json:"current_rating"`
	TotalBlunders     int     `json:"total_blunders"`
	AvgBlundersPerGame float64 `json:"avg_blunders_per_game"`
}
