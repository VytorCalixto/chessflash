package analysis

// ClassificationThresholds defines the centipawn thresholds for move classification.
type ClassificationThresholds struct {
	BlunderCP    float64 // Default: 200
	MistakeCP    float64 // Default: 100
	InaccuracyCP float64 // Default: 50
}

// DefaultThresholds returns the standard classification thresholds.
func DefaultThresholds() ClassificationThresholds {
	return ClassificationThresholds{
		BlunderCP:    200,
		MistakeCP:    100,
		InaccuracyCP: 50,
	}
}

// ClassifyMove classifies a move using default thresholds.
func ClassifyMove(evalBefore, evalAfter float64, isWhiteMove bool, movePlayed, bestMove string) string {
	return ClassifyMoveWithThresholds(DefaultThresholds(), evalBefore, evalAfter, isWhiteMove, movePlayed, bestMove)
}

// ClassifyMoveWithThresholds classifies a move using custom thresholds.
func ClassifyMoveWithThresholds(thresholds ClassificationThresholds, evalBefore, evalAfter float64, isWhiteMove bool, movePlayed, bestMove string) string {
	// If the played move matches the best move, it's always "good"
	// This prevents classifying best moves as mistakes/blunders in losing positions
	if movePlayed != "" && bestMove != "" && movePlayed == bestMove {
		return "good"
	}

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
	case loss > thresholds.BlunderCP:
		return "blunder"
	case loss > thresholds.MistakeCP:
		return "mistake"
	case loss > thresholds.InaccuracyCP:
		return "inaccuracy"
	default:
		return "good"
	}
}
