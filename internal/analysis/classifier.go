package analysis

func ClassifyMove(evalBefore, evalAfter float64, isWhiteMove bool) string {
	diff := evalAfter - evalBefore

	// Stockfish evaluates from white's perspective
	// For white moves: negative diff = loss (e.g., +100 -> +50 means white lost 50)
	// For black moves: positive diff = loss (e.g., -100 -> -50 means black lost 50, but from white's perspective the eval improved)
	var loss float64
	// loss is the absolute value of the difference between the evaluations
	if isWhiteMove {
		loss = -diff // negative diff means loss for white
	} else {
		loss = diff // positive diff means loss for black
	}

	switch {
	case loss > 200:
		return "blunder"
	case loss > 100:
		return "mistake"
	case loss > 50:
		return "inaccuracy"
	default:
		return "good"
	}
}
