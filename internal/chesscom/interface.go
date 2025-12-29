package chesscom

import "context"

// ClientInterface defines the interface for Chess.com API operations.
// This interface enables testability by allowing mock implementations.
type ClientInterface interface {
	FetchArchives(ctx context.Context, username string) ([]string, error)
	FetchMonthly(ctx context.Context, archiveURL string) ([]MonthlyGame, error)
}

// Ensure Client implements the interface
var _ ClientInterface = (*Client)(nil)
