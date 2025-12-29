package api

import (
	"encoding/json"
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
