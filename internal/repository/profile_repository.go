package repository

import (
	"context"
	"time"

	"github.com/vytor/chessflash/internal/models"
)

// ProfileRepository handles profile data access
type ProfileRepository interface {
	Get(ctx context.Context, id int64) (*models.Profile, error)
	List(ctx context.Context) ([]models.Profile, error)
	Upsert(ctx context.Context, username string) (*models.Profile, error)
	UpdateSync(ctx context.Context, id int64, t time.Time) error
	Delete(ctx context.Context, id int64) error
}
