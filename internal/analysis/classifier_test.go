package analysis_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vytor/chessflash/internal/analysis"
)

func TestClassifyMove_Blunder(t *testing.T) {
	tests := []struct {
		name        string
		evalBefore  float64
		evalAfter   float64
		isWhiteMove bool
		movePlayed  string
		bestMove    string
		expected    string
	}{
		{
			name:        "white blunder - loses 250 cp",
			evalBefore:  100,
			evalAfter:  -150,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "blunder",
		},
		{
			name:        "black blunder - loses 250 cp",
			evalBefore:  -100,
			evalAfter:   150,
			isWhiteMove: false,
			movePlayed:  "e7e5",
			bestMove:    "d7d5",
			expected:    "blunder",
		},
		{
			name:        "white mistake - loses 150 cp",
			evalBefore:  100,
			evalAfter:  -50,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "mistake",
		},
		{
			name:        "black mistake - loses 150 cp",
			evalBefore:  -100,
			evalAfter:   50,
			isWhiteMove: false,
			movePlayed:  "e7e5",
			bestMove:    "d7d5",
			expected:    "mistake",
		},
		{
			name:        "white inaccuracy - loses 75 cp",
			evalBefore:  100,
			evalAfter:   25,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "inaccuracy",
		},
		{
			name:        "black inaccuracy - loses 75 cp",
			evalBefore:  -100,
			evalAfter:  -25,
			isWhiteMove: false,
			movePlayed:  "e7e5",
			bestMove:    "d7d5",
			expected:    "inaccuracy",
		},
		{
			name:        "white good move - loses 30 cp",
			evalBefore:  100,
			evalAfter:   70,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "good",
		},
		{
			name:        "black good move - loses 30 cp",
			evalBefore:  -100,
			evalAfter:  -70,
			isWhiteMove: false,
			movePlayed:  "e7e5",
			bestMove:    "d7d5",
			expected:    "good",
		},
		{
			name:        "white improves position",
			evalBefore:  100,
			evalAfter:   150,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "good",
		},
		{
			name:        "black improves position",
			evalBefore:  -100,
			evalAfter:  -150,
			isWhiteMove: false,
			movePlayed:  "e7e5",
			bestMove:    "d7d5",
			expected:    "good",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analysis.ClassifyMove(tt.evalBefore, tt.evalAfter, tt.isWhiteMove, tt.movePlayed, tt.bestMove)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyMove_BestMove(t *testing.T) {
	tests := []struct {
		name        string
		evalBefore  float64
		evalAfter   float64
		isWhiteMove bool
		movePlayed  string
		bestMove    string
	}{
		{
			name:        "best move in losing position - should be good",
			evalBefore:  -500,
			evalAfter:   -600,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "e2e4",
		},
		{
			name:        "best move in winning position - should be good",
			evalBefore:  500,
			evalAfter:   600,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "e2e4",
		},
		{
			name:        "best move even if eval drops",
			evalBefore:  100,
			evalAfter:   -100,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "e2e4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analysis.ClassifyMove(tt.evalBefore, tt.evalAfter, tt.isWhiteMove, tt.movePlayed, tt.bestMove)
			assert.Equal(t, "good", result, "best move should always be classified as 'good'")
		})
	}
}

func TestClassifyMove_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		evalBefore  float64
		evalAfter   float64
		isWhiteMove bool
		movePlayed  string
		bestMove    string
		expected    string
	}{
		{
			name:        "exactly 200 cp loss - should be mistake (threshold is exclusive)",
			evalBefore:  100,
			evalAfter:  -100,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "mistake",
		},
		{
			name:        "exactly 100 cp loss - should be inaccuracy (threshold is exclusive)",
			evalBefore:  100,
			evalAfter:   0,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "inaccuracy",
		},
		{
			name:        "exactly 50 cp loss - should be good (threshold is exclusive)",
			evalBefore:  100,
			evalAfter:   50,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "d2d4",
			expected:    "good",
		},
		{
			name:        "empty best move - should classify normally",
			evalBefore:  100,
			evalAfter:  -150,
			isWhiteMove: true,
			movePlayed:  "e2e4",
			bestMove:    "",
			expected:    "blunder",
		},
		{
			name:        "empty move played - should classify normally",
			evalBefore:  100,
			evalAfter:  -150,
			isWhiteMove: true,
			movePlayed:  "",
			bestMove:    "d2d4",
			expected:    "blunder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analysis.ClassifyMove(tt.evalBefore, tt.evalAfter, tt.isWhiteMove, tt.movePlayed, tt.bestMove)
			assert.Equal(t, tt.expected, result)
		})
	}
}
