package models

import "time"

type PuzzleRushSession struct {
	ID              int64      `json:"id"`
	ProfileID       int64      `json:"profile_id"`
	Difficulty      string     `json:"difficulty"` // "easy", "medium", "hard"
	Score           int        `json:"score"`      // number of correct answers
	MistakesMade    int        `json:"mistakes_made"`
	MistakesAllowed int        `json:"mistakes_allowed"` // 5, 3, or 1
	TotalTimeSeconds float64   `json:"total_time_seconds"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type PuzzleRushAttempt struct {
	ID            int64     `json:"id"`
	SessionID     int64     `json:"session_id"`
	FlashcardID   int64     `json:"flashcard_id"`
	WasCorrect    bool      `json:"was_correct"`
	TimeSeconds   float64   `json:"time_seconds"`
	AttemptNumber int       `json:"attempt_number"`
	CreatedAt     time.Time `json:"created_at"`
}

type PuzzleRushStats struct {
	TotalAttempts      int                    `json:"total_attempts"`
	BestScores         map[string]int         `json:"best_scores"` // key: difficulty, value: best score
	AverageScores       map[string]float64     `json:"average_scores"`
	TotalCorrect        int                    `json:"total_correct"`
	TotalMistakes       int                    `json:"total_mistakes"`
	SuccessRate         float64                `json:"success_rate"` // percentage of completed sessions
	AverageTimeSeconds  float64                `json:"average_time_seconds"`
}

type PuzzleRushBestScore struct {
	Difficulty string    `json:"difficulty"`
	Score      int       `json:"score"`
	CompletedAt time.Time `json:"completed_at"`
}
