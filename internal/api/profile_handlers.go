package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
)

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Debug("rendering profiles page")

	profiles, err := s.ProfileService.ListProfiles(r.Context())
	if err != nil {
		handleError(w, r, err)
		return
	}

	s.render(w, r, "pages/profiles.html", pageData{
		"profiles": profiles,
		"current":  profileFromContext(r.Context()),
	})
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	if username == "" {
		log.Warn("create profile with empty username")
		handleError(w, r, errors.NewBadRequestError("username required"))
		return
	}

	profile, err := s.ProfileService.CreateProfile(r.Context(), username)
	if err != nil {
		handleError(w, r, err)
		return
	}

	setProfileCookie(w, profile.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleSelectProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid profile id: %s", idStr)
		handleError(w, r, errors.NewBadRequestError("invalid profile id"))
		return
	}

	profile, err := s.ProfileService.GetProfile(r.Context(), id)
	if err != nil {
		handleError(w, r, err)
		return
	}

	setProfileCookie(w, profile.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid profile id for delete: %s", idStr)
		handleError(w, r, errors.NewBadRequestError("invalid profile id"))
		return
	}

	if err := s.ProfileService.DeleteProfile(r.Context(), id); err != nil {
		handleError(w, r, err)
		return
	}

	if current := profileFromContext(r.Context()); current != nil && current.ID == id {
		clearProfileCookie(w)
	}
	http.Redirect(w, r, "/profiles", http.StatusSeeOther)
}
