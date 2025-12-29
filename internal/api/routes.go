package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)
	r.Use(s.profileMiddleware)

	r.Get("/", s.handleHome)
	r.Post("/import", s.handleImport)
	r.Post("/resume-analysis", s.handleResumeAnalysis)
	r.Get("/games", s.handleGames)
	r.Get("/games/{id}", s.handleGameDetail)
	r.Post("/games/{id}/queue-analysis", s.handleQueueGameAnalysis)
	r.Get("/flashcards", s.handleFlashcards)
	r.Post("/flashcards/{id}/review", s.handleReviewFlashcard)
	r.Get("/openings", s.handleOpenings)
	r.Get("/profiles", s.handleProfiles)
	r.Post("/profiles", s.handleCreateProfile)
	r.Post("/profiles/{id}/select", s.handleSelectProfile)
	r.Post("/profiles/{id}/delete", s.handleDeleteProfile)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	return r
}
