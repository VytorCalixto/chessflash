package chesscom

import "strings"

// DeriveResult determines which color the user played, their opponent, and the result
func DeriveResult(username string, mg MonthlyGame) (playedAs, opponent, result string) {
	if strings.EqualFold(mg.White.Username, username) {
		playedAs = "white"
		opponent = mg.Black.Username
		result = NormalizeResult(mg.White.Result)
		return
	}
	playedAs = "black"
	opponent = mg.White.Username
	result = NormalizeResult(mg.Black.Result)
	return
}

// NormalizeResult converts chess.com result strings to standardized values
func NormalizeResult(res string) string {
	res = strings.ToLower(res)
	switch res {
	case "win":
		return "win"
	case "stalemate", "agreed", "repetition", "timevsinsufficient", "insufficient", "fiftymove", "draw":
		return "draw"
	case "checkmated", "resigned", "timeout", "abandoned", "kingofthehill", "threecheck", "bughousepartnerlose":
		return "loss"
	default:
		return "loss"
	}
}
