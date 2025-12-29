package api

import (
	"encoding/json"
	"net/http"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

// handleError centralizes error handling for HTTP responses
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	log := logger.FromContext(r.Context())

	// Check if it's already an AppError
	appErr, ok := err.(*errors.AppError)
	if !ok {
		// Wrap unknown errors as internal errors
		appErr = errors.NewInternalError(err)
	}

	// Log based on status code
	if appErr.Status >= 500 {
		log.Error("server error: %v", appErr)
	} else if appErr.Status >= 400 {
		log.Warn("client error: %v", appErr)
	} else {
		log.Debug("error: %v", appErr)
	}

	// Check if this is a JSON endpoint (API routes)
	if r.URL.Path == "/api/evaluate" || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.Status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		})
		return
	}

	// For HTML endpoints, redirect or show error
	// For now, use http.Error for simplicity
	// In the future, could render an error page
	http.Error(w, appErr.Message, appErr.Status)
}
