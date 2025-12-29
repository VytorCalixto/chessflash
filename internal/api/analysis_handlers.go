package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
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

	// Parse filters from query string (optional)
	var baseFilter models.AnalysisFilter
	hasFilters := false
	if len(r.URL.Query()) > 0 {
		baseFilter = parseAnalysisFilter(r, profile.ID)
		// Check if any filters were actually provided
		hasFilters = len(baseFilter.TimeClasses) > 0 || baseFilter.Result != "" ||
			baseFilter.Opponent != "" || baseFilter.OpeningName != "" ||
			baseFilter.StartDate != nil || baseFilter.EndDate != nil ||
			baseFilter.MinRating > 0 || baseFilter.MaxRating > 0 ||
			baseFilter.PlayedAs != ""
	}

	var pending, processing, completed, failed int
	var err error

	if hasFilters {
		// Use filtered counts
		baseFilter.ProfileID = profile.ID
		
		// For pending: status = 'pending'
		pending, err = s.GameService.CountGamesByStatusWithFilter(ctx, profile.ID, "pending", baseFilter)
		if err != nil {
			log.Warn("failed to count pending games: %v", err)
			pending = 0
		}

		// For processing: status = 'processing'
		processing, err = s.GameService.CountGamesByStatusWithFilter(ctx, profile.ID, "processing", baseFilter)
		if err != nil {
			log.Warn("failed to count processing games: %v", err)
			processing = 0
		}

		// For completed: status = 'completed'
		completed, err = s.GameService.CountGamesByStatusWithFilter(ctx, profile.ID, "completed", baseFilter)
		if err != nil {
			log.Warn("failed to count completed games: %v", err)
			completed = 0
		}

		// For failed: status = 'failed'
		failed, err = s.GameService.CountGamesByStatusWithFilter(ctx, profile.ID, "failed", baseFilter)
		if err != nil {
			log.Warn("failed to count failed games: %v", err)
			failed = 0
		}
	} else {
		// Use unfiltered counts (backward compatibility)
		pending, err = s.GameService.CountGamesByStatus(ctx, profile.ID, "pending")
		if err != nil {
			log.Warn("failed to count pending games: %v", err)
			pending = 0
		}

		processing, err = s.GameService.CountGamesByStatus(ctx, profile.ID, "processing")
		if err != nil {
			log.Warn("failed to count processing games: %v", err)
			processing = 0
		}

		completed, err = s.GameService.CountGamesByStatus(ctx, profile.ID, "completed")
		if err != nil {
			log.Warn("failed to count completed games: %v", err)
			completed = 0
		}

		failed, err = s.GameService.CountGamesByStatus(ctx, profile.ID, "failed")
		if err != nil {
			log.Warn("failed to count failed games: %v", err)
			failed = 0
		}
	}

	// Get queue size
	queueSize := s.AnalysisPool.QueueSize()

	// Get worker count
	workerCount := s.AnalysisPool.WorkerCount()
	if workerCount == 0 {
		workerCount = 1 // Avoid division by zero
	}

	// Check if pool is running
	isRunning := s.AnalysisPool.IsRunning()

	// Get average analysis time
	avgTime, err := s.GameService.GetAverageAnalysisTime(ctx, profile.ID)
	if err != nil {
		log.Warn("failed to get average analysis time: %v", err)
		avgTime = 30.0 // Default fallback
	}

	// Calculate estimated time to completion
	// Include queue_size in the count since those games will be processed
	totalGamesToProcess := pending + processing + queueSize
	var estimatedSeconds int64
	if avgTime > 0 && totalGamesToProcess > 0 {
		estimatedSeconds = int64((float64(totalGamesToProcess) * avgTime) / float64(workerCount))
	}

	status := map[string]interface{}{
		"pending":          pending,
		"processing":       processing,
		"completed":        completed,
		"failed":           failed,
		"queue_size":       queueSize,
		"is_running":        isRunning,
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

func (s *Server) handleAnalysisQueuePage(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	// Get available time classes and other filter options
	timeClasses := []string{"bullet", "blitz", "rapid", "daily"}
	results := []string{"win", "loss", "draw"}
	colors := []string{"white", "black"}

	s.render(w, r, "pages/analysis_queue.html", pageData{
		"profile":     profile,
		"time_classes": timeClasses,
		"results":     results,
		"colors":      colors,
	})
}

func (s *Server) handleAnalysisQueueCount(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		handleError(w, r, errors.NewBadRequestError("profile required"))
		return
	}

	filter := parseAnalysisFilter(r, profile.ID)
	count, err := s.GameService.CountGamesForAnalysis(r.Context(), filter)
	if err != nil {
		log.Error("failed to count games for analysis: %v", err)
		handleError(w, r, errors.NewInternalError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"count": count,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response: %v", err)
	}
}

func (s *Server) handleQueueAnalysis(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		handleError(w, r, errors.NewBadRequestError("method not allowed"))
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Warn("failed to parse form: %v", err)
		handleError(w, r, errors.NewBadRequestError("invalid form data"))
		return
	}

	filter := parseAnalysisFilter(r, profile.ID)
	
	// Get count first (quick operation)
	count, err := s.GameService.CountGamesForAnalysis(r.Context(), filter)
	if err != nil {
		log.Error("failed to count games for analysis: %v", err)
		handleError(w, r, err)
		return
	}

	// Clear the queue and restart the pool to ensure clean state with new filters
	// This prevents mixing games from different filter sets
	// Do this asynchronously to avoid blocking the HTTP response
	go func() {
		ctx := context.Background()
		ctx = logger.NewContext(ctx, log.WithField("async_queue", true))
		
		if s.AnalysisPool.IsRunning() {
			log.Info("clearing analysis queue and restarting pool for new filter set")
			s.AnalysisPool.Cancel()
			s.AnalysisPool.ClearQueue()
			// Restart with a new background context
			s.AnalysisPool.Restart(ctx)
		}
		
		// Queue games after clearing/restarting
		queuedCount, err := s.GameService.QueueGamesForAnalysis(ctx, filter)
		if err != nil {
			log.Error("failed to queue games for analysis: %v", err)
		} else {
			log.Info("queued %d games for analysis (async)", queuedCount)
		}
	}()

	log.Info("initiated queueing of %d games for analysis (async)", count)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func parseAnalysisFilter(r *http.Request, profileID int64) models.AnalysisFilter {
	filter := models.AnalysisFilter{
		ProfileID: profileID,
	}

	// Time class (can be multiple)
	timeClasses := r.URL.Query()["time_class"]
	if len(timeClasses) == 0 {
		timeClasses = r.Form["time_class"]
	}
	filter.TimeClasses = timeClasses

	// Result
	if result := r.URL.Query().Get("result"); result != "" {
		filter.Result = result
	} else if result := r.FormValue("result"); result != "" {
		filter.Result = result
	}

	// Opponent
	if opponent := r.URL.Query().Get("opponent"); opponent != "" {
		filter.Opponent = opponent
	} else if opponent := r.FormValue("opponent"); opponent != "" {
		filter.Opponent = opponent
	}

	// Opening
	if opening := r.URL.Query().Get("opening"); opening != "" {
		filter.OpeningName = opening
	} else if opening := r.FormValue("opening"); opening != "" {
		filter.OpeningName = opening
	}

	// Date preset (takes precedence over custom dates)
	if preset := r.URL.Query().Get("date_preset"); preset != "" {
		if startDate, endDate := applyDatePreset(preset); startDate != nil || endDate != nil {
			filter.StartDate = startDate
			filter.EndDate = endDate
		}
	} else if preset := r.FormValue("date_preset"); preset != "" {
		if startDate, endDate := applyDatePreset(preset); startDate != nil || endDate != nil {
			filter.StartDate = startDate
			filter.EndDate = endDate
		}
	} else {
		// Custom date range
		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
				filter.StartDate = &t
			}
		} else if startDateStr := r.FormValue("start_date"); startDateStr != "" {
			if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
				filter.StartDate = &t
			}
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
				// Set to end of day
				t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				filter.EndDate = &t
			}
		} else if endDateStr := r.FormValue("end_date"); endDateStr != "" {
			if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
				// Set to end of day
				t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				filter.EndDate = &t
			}
		}
	}

	// Limit (queue next N games)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	} else if limitStr := r.FormValue("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	// Rating range
	if minRatingStr := r.URL.Query().Get("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.Atoi(minRatingStr); err == nil && minRating > 0 {
			filter.MinRating = minRating
		}
	} else if minRatingStr := r.FormValue("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.Atoi(minRatingStr); err == nil && minRating > 0 {
			filter.MinRating = minRating
		}
	}

	if maxRatingStr := r.URL.Query().Get("max_rating"); maxRatingStr != "" {
		if maxRating, err := strconv.Atoi(maxRatingStr); err == nil && maxRating > 0 {
			filter.MaxRating = maxRating
		}
	} else if maxRatingStr := r.FormValue("max_rating"); maxRatingStr != "" {
		if maxRating, err := strconv.Atoi(maxRatingStr); err == nil && maxRating > 0 {
			filter.MaxRating = maxRating
		}
	}

	// Color
	if playedAs := r.URL.Query().Get("played_as"); playedAs != "" {
		filter.PlayedAs = playedAs
	} else if playedAs := r.FormValue("played_as"); playedAs != "" {
		filter.PlayedAs = playedAs
	}

	// Include failed
	if includeFailed := r.URL.Query().Get("include_failed"); includeFailed != "" {
		filter.IncludeFailed = strings.ToLower(includeFailed) == "true" || includeFailed == "1"
	} else if includeFailed := r.FormValue("include_failed"); includeFailed != "" {
		filter.IncludeFailed = strings.ToLower(includeFailed) == "true" || includeFailed == "1"
	}

	return filter
}

func applyDatePreset(preset string) (*time.Time, *time.Time) {
	now := time.Now()
	var startDate, endDate *time.Time

	switch preset {
	case "last_week":
		start := now.AddDate(0, 0, -7)
		startDate = &start
		endDate = &now
	case "last_month":
		start := now.AddDate(0, 0, -30)
		startDate = &start
		endDate = &now
	case "last_3_months":
		start := now.AddDate(0, 0, -90)
		startDate = &start
		endDate = &now
	case "last_year":
		start := now.AddDate(0, 0, -365)
		startDate = &start
		endDate = &now
	case "all_time":
		// nil, nil means no date filtering
		return nil, nil
	}

	return startDate, endDate
}
