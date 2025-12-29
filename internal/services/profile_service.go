package services

import (
	"context"
	"database/sql"

	"github.com/vytor/chessflash/internal/errors"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

// ProfileService handles profile-related business logic
type ProfileService interface {
	ListProfiles(ctx context.Context) ([]models.Profile, error)
	CreateProfile(ctx context.Context, username string) (*models.Profile, error)
	GetProfile(ctx context.Context, id int64) (*models.Profile, error)
	DeleteProfile(ctx context.Context, id int64) error
}

type profileService struct {
	profileRepo repository.ProfileRepository
}

// NewProfileService creates a new ProfileService
func NewProfileService(profileRepo repository.ProfileRepository) ProfileService {
	return &profileService{profileRepo: profileRepo}
}

func (s *profileService) ListProfiles(ctx context.Context) ([]models.Profile, error) {
	log := logger.FromContext(ctx)
	log.Debug("listing profiles")

	profiles, err := s.profileRepo.List(ctx)
	if err != nil {
		log.Error("failed to list profiles: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return profiles, nil
}

func (s *profileService) CreateProfile(ctx context.Context, username string) (*models.Profile, error) {
	log := logger.FromContext(ctx)
	log.Debug("creating profile: username=%s", username)

	if username == "" {
		return nil, errors.NewValidationError("username", "cannot be empty")
	}

	profile, err := s.profileRepo.Upsert(ctx, username)
	if err != nil {
		log.Error("failed to create profile: %v", err)
		return nil, errors.NewInternalError(err)
	}

	return profile, nil
}

func (s *profileService) GetProfile(ctx context.Context, id int64) (*models.Profile, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting profile: id=%d", id)

	profile, err := s.profileRepo.Get(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("profile", id)
		}
		log.Error("failed to get profile: %v", err)
		return nil, errors.NewInternalError(err)
	}

	if profile == nil {
		return nil, errors.NewNotFoundError("profile", id)
	}

	return profile, nil
}

func (s *profileService) DeleteProfile(ctx context.Context, id int64) error {
	log := logger.FromContext(ctx)
	log.Debug("deleting profile: id=%d", id)

	if err := s.profileRepo.Delete(ctx, id); err != nil {
		log.Error("failed to delete profile: %v", err)
		return errors.NewInternalError(err)
	}

	return nil
}
