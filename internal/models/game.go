package models

import "time"

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

type AnalysisFilter struct {
	ProfileID     int64
	TimeClasses   []string // Support multiple time classes
	Result        string
	Opponent      string
	OpeningName   string
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int
	MinRating     int
	MaxRating     int
	PlayedAs      string
	IncludeFailed bool
}
