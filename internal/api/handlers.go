package api

import (
	"html/template"
	"net/http"

	"github.com/vytor/chessflash/internal/chesscom"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/services"
	"github.com/vytor/chessflash/internal/worker"
)

type Server struct {
	ProfileService       services.ProfileService
	GameService          services.GameService
	FlashcardService     services.FlashcardService
	StatsService         services.StatsService
	ImportService        services.ImportService
	AnalysisService      services.AnalysisService
	AnalysisPool         *worker.Pool
	ImportPool           *worker.Pool
	ChessClient          *chesscom.Client
	Templates            *template.Template
	StockfishPath        string
	StockfishDepth       int
	ArchiveLimit         int
	MaxConcurrentArchive int
}

type pageData map[string]any

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data pageData) {
	if data == nil {
		data = pageData{}
	}
	if _, ok := data["profile"]; !ok {
		data["profile"] = profileFromContext(r.Context())
	}

	log := logger.FromContext(r.Context())
	if err := s.Templates.ExecuteTemplate(w, name, data); err != nil {
		log.Error("failed to render template %s: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
