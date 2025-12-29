package flashcard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vytor/chessflash/internal/flashcard"
	"github.com/vytor/chessflash/internal/models"
)

func TestApplyReview_PerfectScore(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   2.5,
		IntervalDays: 1,
		DueAt:        time.Now(),
	}

	updated := flashcard.ApplyReview(card, 5)

	require.NotNil(t, updated)
	assert.Greater(t, updated.IntervalDays, card.IntervalDays, "interval should increase with perfect score")
	assert.GreaterOrEqual(t, updated.EaseFactor, card.EaseFactor, "ease factor should increase or stay same")
	assert.Equal(t, 1, updated.TimesReviewed, "times reviewed should increment")
	assert.Equal(t, 1, updated.TimesCorrect, "times correct should increment")
	assert.True(t, updated.DueAt.After(time.Now()), "due date should be in the future")
}

func TestApplyReview_Again(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   2.5,
		IntervalDays: 10,
		DueAt:        time.Now(),
	}

	updated := flashcard.ApplyReview(card, 0)

	assert.Equal(t, 1, updated.IntervalDays, "interval should reset to 1 for 'again'")
	assert.Less(t, updated.EaseFactor, card.EaseFactor, "ease factor should decrease")
	assert.Equal(t, 1, updated.TimesReviewed, "times reviewed should increment")
	assert.Equal(t, 0, updated.TimesCorrect, "times correct should reset to 0")
}

func TestApplyReview_Hard(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   2.5,
		IntervalDays: 10,
		DueAt:        time.Now(),
	}

	updated := flashcard.ApplyReview(card, 1)

	assert.Equal(t, 1, updated.IntervalDays, "interval should reset to 1 for 'hard'")
	assert.Less(t, updated.EaseFactor, card.EaseFactor, "ease factor should decrease")
	assert.Equal(t, 1, updated.TimesReviewed, "times reviewed should increment")
	assert.Equal(t, 0, updated.TimesCorrect, "times correct should not increment")
}

func TestApplyReview_Good(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   2.5,
		IntervalDays: 1,
		DueAt:        time.Now(),
	}

	updated := flashcard.ApplyReview(card, 2)

	assert.Equal(t, 6, updated.IntervalDays, "interval should be 6 when previous was 1")
	assert.GreaterOrEqual(t, updated.EaseFactor, card.EaseFactor, "ease factor should increase or stay same")
	assert.Equal(t, 1, updated.TimesReviewed, "times reviewed should increment")
	assert.Equal(t, 1, updated.TimesCorrect, "times correct should increment")
}

func TestApplyReview_Easy(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   2.5,
		IntervalDays: 10,
		DueAt:        time.Now(),
	}

	updated := flashcard.ApplyReview(card, 3)

	assert.Greater(t, updated.IntervalDays, card.IntervalDays, "interval should increase significantly")
	assert.Greater(t, updated.EaseFactor, card.EaseFactor, "ease factor should increase")
	assert.Equal(t, 1, updated.TimesReviewed, "times reviewed should increment")
	assert.Equal(t, 1, updated.TimesCorrect, "times correct should increment")
}

func TestApplyReview_FirstReview(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:    2.5,
		IntervalDays:  0,
		DueAt:         time.Now(),
		TimesReviewed: 0,
		TimesCorrect:  0,
	}

	updated := flashcard.ApplyReview(card, 2)

	assert.Equal(t, 1, updated.IntervalDays, "first review should set interval to 1")
	assert.Equal(t, 1, updated.TimesReviewed, "times reviewed should be 1")
	assert.Equal(t, 1, updated.TimesCorrect, "times correct should be 1")
}

func TestApplyReview_IntervalCalculation(t *testing.T) {
	tests := []struct {
		name         string
		quality      int
		intervalDays int
		easeFactor   float64
		expected     int
	}{
		{
			name:         "interval 1 with good review becomes 6",
			quality:      2,
			intervalDays: 1,
			easeFactor:   2.5,
			expected:     6,
		},
		{
			name:         "interval 6 with good review multiplies by ease factor",
			quality:      2,
			intervalDays: 6,
			easeFactor:   2.5,
			expected:     15, // 6 * 2.5 = 15
		},
		{
			name:         "interval 10 with easy review multiplies by higher ease factor",
			quality:      3,
			intervalDays: 10,
			easeFactor:   2.5,
			expected:     26, // 10 * 2.6 (approx)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := models.Flashcard{
				EaseFactor:   tt.easeFactor,
				IntervalDays: tt.intervalDays,
				DueAt:        time.Now(),
			}

			updated := flashcard.ApplyReview(card, tt.quality)

			assert.Equal(t, tt.expected, updated.IntervalDays)
		})
	}
}

func TestApplyReview_MinEaseFactor(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   1.3,
		IntervalDays: 10,
		DueAt:        time.Now(),
	}

	// Multiple "again" reviews should not drop below 1.3
	for i := 0; i < 10; i++ {
		card = flashcard.ApplyReview(card, 0)
		assert.GreaterOrEqual(t, card.EaseFactor, 1.3, "ease factor should not drop below 1.3")
	}
}

func TestApplyReview_TimesCorrectReset(t *testing.T) {
	card := models.Flashcard{
		EaseFactor:   2.5,
		IntervalDays: 10,
		TimesCorrect: 5,
		DueAt:        time.Now(),
	}

	// Good review should increment
	card = flashcard.ApplyReview(card, 2)
	assert.Equal(t, 6, card.TimesCorrect)

	// "Again" review should reset
	card = flashcard.ApplyReview(card, 0)
	assert.Equal(t, 0, card.TimesCorrect)
}
