package services

// AnalysisConfig holds configuration for game analysis
type AnalysisConfig struct {
	StockfishPath   string
	StockfishDepth  int
	StockfishMaxTime int // milliseconds, 0 = no limit
}
