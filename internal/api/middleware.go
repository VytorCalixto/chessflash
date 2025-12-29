package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

type contextKey string

const (
	profileContextKey contextKey = "profile"
	profileCookieName            = "profile_id"
)

func profileFromContext(ctx context.Context) *models.Profile {
	if v := ctx.Value(profileContextKey); v != nil {
		if p, ok := v.(*models.Profile); ok {
			return p
		}
	}
	return nil
}

func (s *Server) profileMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Allow profile selection and static assets without an active profile.
		if strings.HasPrefix(path, "/profiles") || strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		log := logger.FromContext(r.Context())
		cookie, err := r.Cookie(profileCookieName)
		if err != nil || cookie.Value == "" {
			log.Debug("no profile cookie, redirecting to /profiles")
			http.Redirect(w, r, "/profiles", http.StatusSeeOther)
			return
		}

		profileID, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			log.Warn("invalid profile cookie, clearing")
			clearProfileCookie(w)
			http.Redirect(w, r, "/profiles", http.StatusSeeOther)
			return
		}

		profile, err := s.ProfileService.GetProfile(r.Context(), profileID)
		if err != nil {
			log.Error("failed to load profile: %v", err)
			clearProfileCookie(w)
			http.Redirect(w, r, "/profiles", http.StatusSeeOther)
			return
		}
		if profile == nil {
			log.Warn("profile not found for cookie, redirecting")
			clearProfileCookie(w)
			http.Redirect(w, r, "/profiles", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), profileContextKey, profile)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func clearProfileCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    profileCookieName,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
	})
}

func setProfileCookie(w http.ResponseWriter, id int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     profileCookieName,
		Value:    strconv.FormatInt(id, 10),
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true when behind HTTPS (set via environment/config)
	})
}

// generateRequestID creates a random request ID.
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// loggingMiddleware logs HTTP requests with timing, status codes, and request IDs.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Create a request-scoped logger with the request ID
		log := logger.Default().WithFields(map[string]any{
			"request_id": requestID,
			"method":     r.Method,
			"path":       r.URL.Path,
		})

		// Add remote address if available
		if r.RemoteAddr != "" {
			log = log.WithField("remote_addr", r.RemoteAddr)
		}

		// Store logger in context
		ctx := logger.NewContext(r.Context(), log)
		r = r.WithContext(ctx)

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		log.Debug("request started")

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log the completed request
		duration := time.Since(start)
		log = log.WithFields(map[string]any{
			"status":      wrapped.status,
			"size":        wrapped.size,
			"duration_ms": duration.Milliseconds(),
		})

		if wrapped.status >= 500 {
			log.Error("request completed with server error")
		} else if wrapped.status >= 400 {
			log.Warn("request completed with client error")
		} else {
			log.Info("request completed")
		}
	})
}

// recoveryMiddleware recovers from panics and logs them.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log := logger.FromContext(r.Context())
				log.Error("panic recovered: %v", rec)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds security headers to responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// timeoutMiddleware wraps a handler with a timeout.
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}
