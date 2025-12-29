package models

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
