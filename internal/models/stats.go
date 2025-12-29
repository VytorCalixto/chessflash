package models

import "time"

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

type SummaryStat struct {
	TotalGames         int     `json:"total_games"`
	OverallWinRate     float64 `json:"overall_win_rate"`
	CurrentRating      int     `json:"current_rating"`
	TotalBlunders      int     `json:"total_blunders"`
	AvgBlundersPerGame float64 `json:"avg_blunders_per_game"`
}
