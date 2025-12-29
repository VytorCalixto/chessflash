package jobs

// JobQueue provides an abstraction for enqueueing background jobs
type JobQueue interface {
	EnqueueAnalysis(gameID int64) error
	EnqueueImport(profileID int64, username string) error
}
