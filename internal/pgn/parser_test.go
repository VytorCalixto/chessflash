package pgn_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vytor/chessflash/internal/pgn"
)

func TestParsePGNHeaders_ValidHeaders(t *testing.T) {
	pgnText := `[Event "Live Chess"]
[Site "Chess.com"]
[Date "2024.01.15"]
[Round "-"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]
[WhiteElo "1500"]
[BlackElo "1600"]
[TimeControl "600+0"]
[ECO "B20"]
[Opening "Sicilian Defense"]

1. e4 c5 2. Nf3 d6`

	headers := pgn.ParsePGNHeaders(pgnText)

	assert.Equal(t, "Live Chess", headers["Event"])
	assert.Equal(t, "Chess.com", headers["Site"])
	assert.Equal(t, "2024.01.15", headers["Date"])
	assert.Equal(t, "Player1", headers["White"])
	assert.Equal(t, "Player2", headers["Black"])
	assert.Equal(t, "1-0", headers["Result"])
	assert.Equal(t, "1500", headers["WhiteElo"])
	assert.Equal(t, "1600", headers["BlackElo"])
	assert.Equal(t, "B20", headers["ECO"])
}

func TestParsePGNHeaders_EmptyPGN(t *testing.T) {
	pgnText := ""
	headers := pgn.ParsePGNHeaders(pgnText)
	assert.Empty(t, headers)
}

func TestParsePGNHeaders_NoHeaders(t *testing.T) {
	pgnText := `1. e4 e5 2. Nf3 Nc6`
	headers := pgn.ParsePGNHeaders(pgnText)
	assert.Empty(t, headers)
}

func TestParsePGNHeaders_MalformedHeaders(t *testing.T) {
	pgnText := `[Event Live Chess]
[Site Chess.com]
[Invalid header]
1. e4 e5`

	headers := pgn.ParsePGNHeaders(pgnText)
	assert.Empty(t, headers, "malformed headers should be ignored")
}

func TestParsePGNHeaders_HeadersWithQuotes(t *testing.T) {
	pgnText := `[Event "Live Chess Tournament"]
[Site "Chess.com"]
[Opening "King's Gambit"]`

	headers := pgn.ParsePGNHeaders(pgnText)
	assert.Equal(t, "Live Chess Tournament", headers["Event"])
	assert.Equal(t, "King's Gambit", headers["Opening"])
}

func TestExtractGameID_ValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard chess.com URL",
			url:      "https://www.chess.com/game/live/12345678",
			expected: "12345678",
		},
		{
			name:     "URL with path",
			url:      "https://www.chess.com/game/live/98765432/analysis",
			expected: "98765432",
		},
		{
			name:     "URL with username - regex doesn't match this pattern",
			url:      "https://www.chess.com/game/live/player1/12345678",
			expected: "https://www.chess.com/game/live/player1/12345678", // Returns original when pattern doesn't match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pgn.ExtractGameID(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractGameID_InvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "non-chess.com URL",
			url:      "https://example.com/game/123",
			expected: "https://example.com/game/123",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "URL without game ID",
			url:      "https://www.chess.com/game/live",
			expected: "https://www.chess.com/game/live",
		},
		{
			name:     "plain text",
			url:      "just some text",
			expected: "just some text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pgn.ExtractGameID(tt.url)
			assert.Equal(t, tt.expected, result, "invalid URLs should return the original string")
		})
	}
}

func TestExtractGameID_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "very long game ID",
			url:      "https://www.chess.com/game/live/12345678901234567890",
			expected: "12345678901234567890",
		},
		{
			name:     "game ID with leading zeros",
			url:      "https://www.chess.com/game/live/00012345",
			expected: "00012345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pgn.ExtractGameID(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
