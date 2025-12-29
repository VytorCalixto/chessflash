package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

func (s *Server) handleEvaluatePosition(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	// Get FEN from query parameter
	fen := r.URL.Query().Get("fen")
	if fen == "" {
		handleError(w, r, errors.NewBadRequestError("fen parameter required"))
		return
	}

	result, err := s.AnalysisService.EvaluatePosition(r.Context(), fen, s.StockfishPath, s.StockfishDepth)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"cp":   result.CP,
		"mate": result.Mate,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response: %v", err)
	}
}

func (s *Server) handleAnalysisStatus(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		handleError(w, r, errors.NewBadRequestError("profile required"))
		return
	}

	ctx := r.Context()

	// Get counts for each status
	pending, err := s.GameService.CountGamesByStatus(ctx, profile.ID, "pending")
	if err != nil {
		log.Warn("failed to count pending games: %v", err)
		pending = 0
	}

	processing, err := s.GameService.CountGamesByStatus(ctx, profile.ID, "processing")
	if err != nil {
		log.Warn("failed to count processing games: %v", err)
		processing = 0
	}

	completed, err := s.GameService.CountGamesByStatus(ctx, profile.ID, "completed")
	if err != nil {
		log.Warn("failed to count completed games: %v", err)
		completed = 0
	}

	failed, err := s.GameService.CountGamesByStatus(ctx, profile.ID, "failed")
	if err != nil {
		log.Warn("failed to count failed games: %v", err)
		failed = 0
	}

	// Get queue size
	queueSize := s.AnalysisPool.QueueSize()

	// Get worker count
	workerCount := s.AnalysisPool.WorkerCount()
	if workerCount == 0 {
		workerCount = 1 // Avoid division by zero
	}

	// Get average analysis time
	avgTime, err := s.GameService.GetAverageAnalysisTime(ctx, profile.ID)
	if err != nil {
		log.Warn("failed to get average analysis time: %v", err)
		avgTime = 30.0 // Default fallback
	}

	// Calculate estimated time to completion
	var estimatedSeconds int64
	if avgTime > 0 && (pending+processing) > 0 {
		estimatedSeconds = int64((float64(pending+processing) * avgTime) / float64(workerCount))
	}

	status := map[string]interface{}{
		"pending":          pending,
		"processing":       processing,
		"completed":        completed,
		"failed":           failed,
		"queue_size":       queueSize,
		"estimated_seconds": estimatedSeconds,
		"estimated_time":   formatDuration(estimatedSeconds),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Error("failed to encode response: %v", err)
	}
}

func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return "N/A"
	}
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := minutes / 60
	minutes = minutes % 60
	if minutes == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, minutes)
}
