package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr                   string
	DBPath                 string
	StockfishPath          string
	StockfishDepth         int
	LogLevel               string
	AnalysisWorkerCount    int
	AnalysisQueueSize      int
	ImportWorkerCount      int
	ImportQueueSize        int
	ArchiveLimit           int
	MaxConcurrentArchive   int
}

// Load reads configuration from a .env file (if present) and environment variables,
// applying sensible defaults when values are missing or invalid.
func Load() Config {
	// Ignore error so the app still starts when .env is absent in production.
	_ = godotenv.Load()

	return Config{
		Addr:                   envOr("ADDR", ":8080"),
		DBPath:                 envOr("DB_PATH", "file:chessflash.db"),
		StockfishPath:          envOr("STOCKFISH_PATH", "stockfish"),
		StockfishDepth:         envIntOr("STOCKFISH_DEPTH", 18),
		LogLevel:               envOr("LOG_LEVEL", "INFO"),
		AnalysisWorkerCount:    envIntOr("ANALYSIS_WORKER_COUNT", 2),
		AnalysisQueueSize:      envIntOr("ANALYSIS_QUEUE_SIZE", 64),
		ImportWorkerCount:      envIntOr("IMPORT_WORKER_COUNT", 2),
		ImportQueueSize:        envIntOr("IMPORT_QUEUE_SIZE", 32),
		ArchiveLimit:           envIntOr("ARCHIVE_LIMIT", 0),
		MaxConcurrentArchive:   envIntOr("MAX_CONCURRENT_ARCHIVE", 10),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		log.Printf("invalid value for %s=%q, using default %d", key, v, def)
	}
	return def
}
