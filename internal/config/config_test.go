package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vytor/chessflash/internal/config"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := config.Config{
		Addr:                 ":8080",
		DBPath:               "test.db",
		StockfishPath:        "", // Empty path skips validation (will use default)
		StockfishDepth:       18,
		StockfishMaxTime:     0,
		LogLevel:             "INFO",
		AnalysisWorkerCount:  2,
		AnalysisQueueSize:    64,
		ImportWorkerCount:    2,
		ImportQueueSize:      32,
		ArchiveLimit:         0,
		MaxConcurrentArchive: 10,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidate_EmptyAddr(t *testing.T) {
	cfg := config.Config{
		Addr:        "",
		DBPath:      "test.db",
		StockfishPath: "stockfish",
		StockfishDepth: 18,
		LogLevel:    "INFO",
		AnalysisWorkerCount: 2,
		AnalysisQueueSize: 64,
		ImportWorkerCount: 2,
		ImportQueueSize: 32,
		MaxConcurrentArchive: 10,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ADDR cannot be empty")
}

func TestValidate_EmptyDBPath(t *testing.T) {
	cfg := config.Config{
		Addr:        ":8080",
		DBPath:      "",
		StockfishPath: "stockfish",
		StockfishDepth: 18,
		LogLevel:    "INFO",
		AnalysisWorkerCount: 2,
		AnalysisQueueSize: 64,
		ImportWorkerCount: 2,
		ImportQueueSize: 32,
		MaxConcurrentArchive: 10,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DB_PATH cannot be empty")
}

func TestValidate_InvalidStockfishDepth(t *testing.T) {
	tests := []struct {
		name  string
		depth int
	}{
		{
			name:  "depth too low",
			depth: 0,
		},
		{
			name:  "depth too high",
			depth: 31,
		},
		{
			name:  "negative depth",
			depth: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Addr:        ":8080",
				DBPath:      "test.db",
				StockfishPath: "stockfish",
				StockfishDepth: tt.depth,
				LogLevel:    "INFO",
				AnalysisWorkerCount: 2,
				AnalysisQueueSize: 64,
				ImportWorkerCount: 2,
				ImportQueueSize: 32,
				MaxConcurrentArchive: 10,
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "STOCKFISH_DEPTH")
		})
	}
}

func TestValidate_ValidStockfishDepth(t *testing.T) {
	tests := []struct {
		name  string
		depth int
	}{
		{
			name:  "minimum depth",
			depth: 1,
		},
		{
			name:  "maximum depth",
			depth: 30,
		},
		{
			name:  "middle depth",
			depth: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Addr:        ":8080",
				DBPath:      "test.db",
				StockfishPath: "", // Empty path skips validation
				StockfishDepth: tt.depth,
				LogLevel:    "INFO",
				AnalysisWorkerCount: 2,
				AnalysisQueueSize: 64,
				ImportWorkerCount: 2,
				ImportQueueSize: 32,
				MaxConcurrentArchive: 10,
			}

			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestValidate_InvalidWorkerCounts(t *testing.T) {
	tests := []struct {
		name            string
		analysisWorkers int
		importWorkers   int
		expectedError   string
	}{
		{
			name:            "zero analysis workers",
			analysisWorkers: 0,
			importWorkers:   2,
			expectedError:   "ANALYSIS_WORKER_COUNT",
		},
		{
			name:            "zero import workers",
			analysisWorkers: 2,
			importWorkers:   0,
			expectedError:   "IMPORT_WORKER_COUNT",
		},
		{
			name:            "negative analysis workers",
			analysisWorkers: -1,
			importWorkers:   2,
			expectedError:   "ANALYSIS_WORKER_COUNT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Addr:        ":8080",
				DBPath:      "test.db",
				StockfishPath: "stockfish",
				StockfishDepth: 18,
				LogLevel:    "INFO",
				AnalysisWorkerCount: tt.analysisWorkers,
				AnalysisQueueSize: 64,
				ImportWorkerCount: tt.importWorkers,
				ImportQueueSize: 32,
				MaxConcurrentArchive: 10,
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestValidate_InvalidQueueSizes(t *testing.T) {
	tests := []struct {
		name          string
		analysisQueue int
		importQueue   int
		expectedError string
	}{
		{
			name:          "zero analysis queue",
			analysisQueue: 0,
			importQueue:   32,
			expectedError:  "ANALYSIS_QUEUE_SIZE",
		},
		{
			name:          "zero import queue",
			analysisQueue: 64,
			importQueue:   0,
			expectedError: "IMPORT_QUEUE_SIZE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Addr:        ":8080",
				DBPath:      "test.db",
				StockfishPath: "stockfish",
				StockfishDepth: 18,
				LogLevel:    "INFO",
				AnalysisWorkerCount: 2,
				AnalysisQueueSize: tt.analysisQueue,
				ImportWorkerCount: 2,
				ImportQueueSize: tt.importQueue,
				MaxConcurrentArchive: 10,
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{
			name:  "invalid level",
			level: "INVALID",
		},
		{
			name:  "empty level",
			level: "",
		},
		{
			name:  "lowercase valid level",
			level: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Addr:        ":8080",
				DBPath:      "test.db",
				StockfishPath: "", // Empty path skips validation
				StockfishDepth: 18,
				LogLevel:    tt.level,
				AnalysisWorkerCount: 2,
				AnalysisQueueSize: 64,
				ImportWorkerCount: 2,
				ImportQueueSize: 32,
				MaxConcurrentArchive: 10,
			}

			err := cfg.Validate()
			if tt.level == "debug" {
				// Lowercase should be accepted (converted to uppercase)
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "LOG_LEVEL")
			}
		})
	}
}

func TestValidate_ValidLogLevels(t *testing.T) {
	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			cfg := config.Config{
				Addr:        ":8080",
				DBPath:      "test.db",
				StockfishPath: "", // Empty path skips validation
				StockfishDepth: 18,
				LogLevel:    level,
				AnalysisWorkerCount: 2,
				AnalysisQueueSize: 64,
				ImportWorkerCount: 2,
				ImportQueueSize: 32,
				MaxConcurrentArchive: 10,
			}

			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestValidate_InvalidMaxConcurrentArchive(t *testing.T) {
	cfg := config.Config{
		Addr:        ":8080",
		DBPath:      "test.db",
		StockfishPath: "stockfish",
		StockfishDepth: 18,
		LogLevel:    "INFO",
		AnalysisWorkerCount: 2,
		AnalysisQueueSize: 64,
		ImportWorkerCount: 2,
		ImportQueueSize: 32,
		MaxConcurrentArchive: 0,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MAX_CONCURRENT_ARCHIVE")
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := config.Config{
		Addr:        "",
		DBPath:      "",
		StockfishPath: "stockfish",
		StockfishDepth: 50,
		LogLevel:    "INVALID",
		AnalysisWorkerCount: 0,
		AnalysisQueueSize: 0,
		ImportWorkerCount: 0,
		ImportQueueSize: 0,
		MaxConcurrentArchive: 0,
	}

	err := cfg.Validate()
	require.Error(t, err)
	
	errStr := err.Error()
	assert.Contains(t, errStr, "ADDR cannot be empty")
	assert.Contains(t, errStr, "DB_PATH cannot be empty")
	assert.Contains(t, errStr, "STOCKFISH_DEPTH")
	assert.Contains(t, errStr, "LOG_LEVEL")
	assert.Contains(t, errStr, "ANALYSIS_WORKER_COUNT")
	assert.Contains(t, errStr, "ANALYSIS_QUEUE_SIZE")
	assert.Contains(t, errStr, "IMPORT_WORKER_COUNT")
	assert.Contains(t, errStr, "IMPORT_QUEUE_SIZE")
	assert.Contains(t, errStr, "MAX_CONCURRENT_ARCHIVE")
}

func TestValidate_StockfishPathNotFound(t *testing.T) {
	cfg := config.Config{
		Addr:        ":8080",
		DBPath:      "test.db",
		StockfishPath: "nonexistent-stockfish-binary-12345",
		StockfishDepth: 18,
		LogLevel:    "INFO",
		AnalysisWorkerCount: 2,
		AnalysisQueueSize: 64,
		ImportWorkerCount: 2,
		ImportQueueSize: 32,
		MaxConcurrentArchive: 10,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "STOCKFISH_PATH")
}

func TestValidate_EmptyStockfishPath(t *testing.T) {
	cfg := config.Config{
		Addr:        ":8080",
		DBPath:      "test.db",
		StockfishPath: "",
		StockfishDepth: 18,
		LogLevel:    "INFO",
		AnalysisWorkerCount: 2,
		AnalysisQueueSize: 64,
		ImportWorkerCount: 2,
		ImportQueueSize: 32,
		MaxConcurrentArchive: 10,
	}

	err := cfg.Validate()
	// Empty path should be valid (will use default "stockfish")
	assert.NoError(t, err)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Save original values
	originalAddr := os.Getenv("ADDR")
	originalDBPath := os.Getenv("DB_PATH")
	
	defer func() {
		if originalAddr != "" {
			os.Setenv("ADDR", originalAddr)
		} else {
			os.Unsetenv("ADDR")
		}
		if originalDBPath != "" {
			os.Setenv("DB_PATH", originalDBPath)
		} else {
			os.Unsetenv("DB_PATH")
		}
	}()

	os.Setenv("ADDR", ":9090")
	os.Setenv("DB_PATH", "custom.db")

	cfg := config.Load()

	assert.Equal(t, ":9090", cfg.Addr)
	assert.Equal(t, "custom.db", cfg.DBPath)
}
