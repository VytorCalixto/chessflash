package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

func (s *Server) handleOpenings(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	pageParam := r.URL.Query().Get("page")
	perPageParam := r.URL.Query().Get("per_page")

	page := 1
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}

	perPage := 25
	switch perPageParam {
	case "10":
		perPage = 10
	case "25":
		perPage = 25
	case "50":
		perPage = 50
	case "100":
		perPage = 100
	}

	offset := (page - 1) * perPage

	log = log.WithFields(map[string]any{
		"username": profile.Username,
		"page":     page,
		"per_page": perPage,
	})
	log.Debug("fetching opening stats")

	stats, totalCount, err := s.StatsService.GetOpeningStats(r.Context(), profile.ID, perPage, offset)
	if err != nil {
		handleError(w, r, err)
		return
	}

	totalPages := totalCount / perPage
	if totalCount%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	log.Debug("found %d opening stats (page %d of %d)", len(stats), page, totalPages)
	s.render(w, r, "pages/openings.html", pageData{
		"stats":       stats,
		"profile":     profile,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
		"total_count": totalCount,
	})
}

func (s *Server) handleOpponents(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	pageParam := r.URL.Query().Get("page")
	perPageParam := r.URL.Query().Get("per_page")
	orderBy := r.URL.Query().Get("order_by")
	orderDir := strings.ToUpper(r.URL.Query().Get("order_dir"))

	page := 1
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}

	perPage := 25
	switch perPageParam {
	case "10":
		perPage = 10
	case "25":
		perPage = 25
	case "50":
		perPage = 50
	case "100":
		perPage = 100
	}

	if orderBy != "total_games" && orderBy != "last_played_at" {
		orderBy = "total_games"
	}
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	offset := (page - 1) * perPage

	log = log.WithFields(map[string]any{
		"username":  profile.Username,
		"page":      page,
		"per_page":  perPage,
		"order_by":  orderBy,
		"order_dir": orderDir,
	})
	log.Debug("fetching opponent stats")

	stats, totalCount, err := s.StatsService.GetOpponentStats(r.Context(), profile.ID, perPage, offset, orderBy, orderDir)
	if err != nil {
		handleError(w, r, err)
		return
	}

	totalPages := totalCount / perPage
	if totalCount%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	log.Debug("found %d opponent stats (page %d of %d)", len(stats), page, totalPages)
	s.render(w, r, "pages/opponents.html", pageData{
		"stats":       stats,
		"profile":     profile,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
		"total_count": totalCount,
		"order_by":    orderBy,
		"order_dir":   orderDir,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log = log.WithField("username", profile.Username)
	log.Debug("fetching aggregate stats")

	summaryStats, err := s.StatsService.GetSummaryStats(r.Context(), profile.ID)
	if err != nil {
		log.Warn("failed to get summary stats, continuing without them: %v", err)
		summaryStats = nil
	}
	timeStats, err := s.StatsService.GetTimeClassStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	colorStats, err := s.StatsService.GetColorStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	monthlyStats, err := s.StatsService.GetMonthlyStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	mistakeStats, err := s.StatsService.GetMistakePhaseStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}
	ratingStats, err := s.StatsService.GetRatingStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	s.render(w, r, "pages/stats.html", pageData{
		"summary_stats": summaryStats,
		"time_stats":    timeStats,
		"color_stats":   colorStats,
		"monthly_stats": monthlyStats,
		"mistake_stats": mistakeStats,
		"rating_stats":  ratingStats,
		"profile":       profile,
	})
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log.Debug("rendering analytics hub page")
	s.render(w, r, "pages/analytics.html", pageData{
		"profile": profile,
	})
}

func (s *Server) handleFlashcardAnalytics(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context, redirecting to /profiles")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	log = log.WithField("username", profile.Username)
	log.Debug("fetching flashcard analytics")

	overallStats, err := s.StatsService.GetFlashcardStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	classificationStats, err := s.StatsService.GetFlashcardClassificationStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	phaseStats, err := s.StatsService.GetFlashcardPhaseStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	openingStats, err := s.StatsService.GetFlashcardOpeningStats(r.Context(), profile.ID, 20)
	if err != nil {
		handleError(w, r, err)
		return
	}

	timeStats, err := s.StatsService.GetFlashcardTimeStats(r.Context(), profile.ID)
	if err != nil {
		handleError(w, r, err)
		return
	}

	s.render(w, r, "pages/flashcard_analytics.html", pageData{
		"overall_stats":        overallStats,
		"classification_stats":  classificationStats,
		"phase_stats":          phaseStats,
		"opening_stats":        openingStats,
		"time_stats":           timeStats,
		"profile":              profile,
	})
}

func (s *Server) handleRefreshStats(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile := profileFromContext(r.Context())
	if profile == nil {
		log.Warn("no profile in context during refresh")
		handleError(w, r, errors.NewBadRequestError("no profile selected"))
		return
	}

	log = log.WithField("profile_id", profile.ID)
	log.Info("manually refreshing cached stats")

	if err := s.StatsService.RefreshStats(r.Context(), profile.ID); err != nil {
		handleError(w, r, err)
		return
	}

	log.Info("stats refreshed successfully")

	// Redirect back to referrer or analytics page
	redirectTo := r.FormValue("redirect")
	if redirectTo == "" {
		redirectTo = r.Header.Get("Referer")
	}
	if redirectTo == "" {
		redirectTo = "/analytics"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}
