package worker

import "context"

// AnalysisServiceInterface defines the interface for game analysis
// This avoids import cycles by not importing the services package
type AnalysisServiceInterface interface {
	AnalyzeGame(ctx context.Context, gameID int64) error
}
