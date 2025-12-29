package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

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

// Validate checks that all configuration values are sensible.
// Returns an error describing all validation failures.
func (c Config) Validate() error {
	var errs []string

	// Validate address format
	if c.Addr == "" {
		errs = append(errs, "ADDR cannot be empty")
	}

	// Validate database path
	if c.DBPath == "" {
		errs = append(errs, "DB_PATH cannot be empty")
	}

	// Validate Stockfish path exists and is executable
	if c.StockfishPath != "" {
		if _, err := exec.LookPath(c.StockfishPath); err != nil {
			errs = append(errs, fmt.Sprintf("STOCKFISH_PATH %q not found or not executable", c.StockfishPath))
		}
	}

	// Validate numeric ranges
	if c.StockfishDepth < 1 || c.StockfishDepth > 30 {
		errs = append(errs, fmt.Sprintf("STOCKFISH_DEPTH must be 1-30, got %d", c.StockfishDepth))
	}

	if c.AnalysisWorkerCount < 1 {
		errs = append(errs, fmt.Sprintf("ANALYSIS_WORKER_COUNT must be >= 1, got %d", c.AnalysisWorkerCount))
	}

	if c.AnalysisQueueSize < 1 {
		errs = append(errs, fmt.Sprintf("ANALYSIS_QUEUE_SIZE must be >= 1, got %d", c.AnalysisQueueSize))
	}

	if c.ImportWorkerCount < 1 {
		errs = append(errs, fmt.Sprintf("IMPORT_WORKER_COUNT must be >= 1, got %d", c.ImportWorkerCount))
	}

	if c.ImportQueueSize < 1 {
		errs = append(errs, fmt.Sprintf("IMPORT_QUEUE_SIZE must be >= 1, got %d", c.ImportQueueSize))
	}

	if c.MaxConcurrentArchive < 1 {
		errs = append(errs, fmt.Sprintf("MAX_CONCURRENT_ARCHIVE must be >= 1, got %d", c.MaxConcurrentArchive))
	}

	// Validate log level
	validLogLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true}
	if !validLogLevels[strings.ToUpper(c.LogLevel)] {
		errs = append(errs, fmt.Sprintf("LOG_LEVEL must be DEBUG/INFO/WARN/ERROR, got %q", c.LogLevel))
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// MustLoad calls Load() then Validate(), panicking if validation fails.
func MustLoad() Config {
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration:\n%v", err))
	}
	return cfg
}
