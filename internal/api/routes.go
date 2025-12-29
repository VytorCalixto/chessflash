package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(recoveryMiddleware)
	r.Use(securityHeadersMiddleware)
	r.Use(loggingMiddleware)
	r.Use(s.profileMiddleware)

	r.Get("/", s.handleHome)
	r.Post("/import", s.handleImport)
	r.Post("/resume-analysis", s.handleResumeAnalysis)
	r.Post("/stop-analysis", s.handleStopAnalysis)
	r.Get("/analysis/queue", s.handleAnalysisQueuePage)
	r.Post("/analysis/queue", s.handleQueueAnalysis)
	r.Get("/api/analysis/queue/count", s.handleAnalysisQueueCount)
	r.Get("/games", s.handleGames)
	r.Get("/games/{id}", s.handleGameDetail)
	r.Post("/games/{id}/queue-analysis", s.handleQueueGameAnalysis)
	r.Get("/flashcards", s.handleFlashcards)
	r.Post("/flashcards/{id}/review", s.handleReviewFlashcard)
	r.Get("/flashcards/analytics", s.handleFlashcardAnalytics)
	r.Get("/puzzle-rush", s.handlePuzzleRushPage)
	r.Post("/puzzle-rush/start", s.handlePuzzleRushStart)
	r.Post("/puzzle-rush/answer", s.handlePuzzleRushAnswer)
	r.Get("/puzzle-rush/current", s.handlePuzzleRushCurrent)
	r.Get("/puzzle-rush/stats", s.handlePuzzleRushStats)
	r.Get("/api/evaluate", s.handleEvaluatePosition)
	r.Get("/api/analysis/status", s.handleAnalysisStatus)
	r.Get("/analytics", s.handleAnalytics)
	r.Post("/analytics/refresh", s.handleRefreshStats)
	r.Get("/openings", s.handleOpenings)
	r.Get("/opponents", s.handleOpponents)
	r.Get("/stats", s.handleStats)
	r.Get("/profiles", s.handleProfiles)
	r.Post("/profiles", s.handleCreateProfile)
	r.Post("/profiles/{id}/select", s.handleSelectProfile)
	r.Post("/profiles/{id}/delete", s.handleDeleteProfile)

	// Health check endpoints
	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	return r
}
