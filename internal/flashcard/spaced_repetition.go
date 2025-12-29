package flashcard

import (
	"time"

	"github.com/vytor/chessflash/internal/models"
)

// ApplyReview updates flashcard scheduling using SM-2 variant.
// quality: 0=Again, 1=Hard, 2=Good, 3=Easy
func ApplyReview(card models.Flashcard, quality int) models.Flashcard {
	const minEase = 1.3
	ef := card.EaseFactor
	ef = ef + 0.1 - float64(3-quality)*(0.08+float64(3-quality)*0.02)
	if ef < minEase {
		ef = minEase
	}

	interval := 1
	switch {
	case quality < 2:
		interval = 1
	case card.IntervalDays == 0:
		interval = 1
	case card.IntervalDays == 1:
		interval = 6
	default:
		interval = int(float64(card.IntervalDays) * ef)
	}

	card.TimesReviewed++
	if quality >= 2 {
		card.TimesCorrect++
	} else {
		card.TimesCorrect = 0
	}
	card.IntervalDays = interval
	card.EaseFactor = ef
	card.DueAt = time.Now().Add(time.Duration(interval) * 24 * time.Hour)
	return card
}

