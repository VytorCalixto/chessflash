package analysis

import (
	"fmt"

	"github.com/corentings/chess/v2"
)

// MoveToUCI converts a chess Move to UCI format (e.g., "e2e4", "e7e8q")
func MoveToUCI(move *chess.Move) string {
	if move == nil {
		return ""
	}

	s1 := move.S1()
	s2 := move.S2()
	promo := move.Promo()

	// Convert squares to algebraic notation
	from := squareToString(s1)
	to := squareToString(s2)

	uci := from + to

	// Add promotion piece if present
	if promo != chess.NoPieceType {
		switch promo {
		case chess.Queen:
			uci += "q"
		case chess.Rook:
			uci += "r"
		case chess.Bishop:
			uci += "b"
		case chess.Knight:
			uci += "n"
		}
	}

	return uci
}

// squareToString converts a Square to algebraic notation (e.g., "e2", "a8")
func squareToString(sq chess.Square) string {
	file := sq.File()
	rank := sq.Rank()

	fileChar := 'a' + file
	rankChar := '1' + rank

	return fmt.Sprintf("%c%c", fileChar, rankChar)
}
