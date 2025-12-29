package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vytor/chessflash/internal/api"
	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/config"
	"github.com/vytor/chessflash/internal/db"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/services"
	"github.com/vytor/chessflash/internal/worker"
)

func main() {
	cfg := config.Load()

	// Initialize logger
	log := logger.New(
		logger.WithLevel(logger.ParseLevel(cfg.LogLevel)),
		logger.WithColors(true),
	)
	logger.SetDefault(log)

	log.Info("===========================================")
	log.Info("ChessFlash Server Starting")
	log.Info("===========================================")
	log.Info("configuration loaded")
	log.Debug("addr=%s", cfg.Addr)
	log.Debug("db_path=%s", cfg.DBPath)
	log.Debug("stockfish_path=%s", cfg.StockfishPath)
	log.Debug("stockfish_depth=%d", cfg.StockfishDepth)
	log.Debug("log_level=%s", cfg.LogLevel)
	log.Debug("analysis_worker_count=%d", cfg.AnalysisWorkerCount)
	log.Debug("analysis_queue_size=%d", cfg.AnalysisQueueSize)
	log.Debug("import_worker_count=%d", cfg.ImportWorkerCount)
	log.Debug("import_queue_size=%d", cfg.ImportQueueSize)
	log.Debug("archive_limit=%d", cfg.ArchiveLimit)
	log.Debug("max_concurrent_archive=%d", cfg.MaxConcurrentArchive)

	// Open database
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Error("failed to open database: %v", err)
		os.Exit(1)
	}
	defer func() {
		log.Debug("closing database connection")
		database.Close()
	}()

	// Load templates
	log.Debug("loading templates")
	tmpl, err := api.LoadTemplates()
	if err != nil {
		log.Error("failed to load templates: %v", err)
		os.Exit(1)
	}
	log.Debug("templates loaded successfully")

	// Initialize worker pools
	analysisPool := worker.NewPool(cfg.AnalysisWorkerCount, cfg.AnalysisQueueSize)
	importPool := worker.NewPool(cfg.ImportWorkerCount, cfg.ImportQueueSize)

	// Initialize services
	profileService := services.NewProfileService(database)
	gameService := services.NewGameService(database)
	flashcardService := services.NewFlashcardService(database)
	statsService := services.NewStatsService(database)
	importService := services.NewImportService(database)
	analysisService := services.NewAnalysisService()

	srv := &api.Server{
		ProfileService:       profileService,
		GameService:          gameService,
		FlashcardService:     flashcardService,
		StatsService:         statsService,
		ImportService:        importService,
		AnalysisService:      analysisService,
		AnalysisPool:         analysisPool,
		ImportPool:           importPool,
		ChessClient:          chesscom.New(),
		Templates:            tmpl,
		StockfishPath:        cfg.StockfishPath,
		StockfishDepth:       cfg.StockfishDepth,
		ArchiveLimit:         cfg.ArchiveLimit,
		MaxConcurrentArchive: cfg.MaxConcurrentArchive,
	}

	ctx, cancel := context.WithCancel(context.Background())
	analysisPool.Start(ctx)
	importPool.Start(ctx)

	// Configure HTTP server
	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		log.Info("HTTP server listening on %s", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop

	log.Info("received signal %v, initiating graceful shutdown", sig)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel worker context
	log.Debug("stopping worker pools")
	cancel()

	// Shutdown HTTP server
	log.Debug("shutting down HTTP server")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error: %v", err)
	}

	// Wait for workers to finish
	log.Debug("stopping analysis pool")
	analysisPool.Stop()
	log.Debug("stopping import pool")
	importPool.Stop()

	log.Info("===========================================")
	log.Info("ChessFlash Server Stopped")
	log.Info("===========================================")
}
