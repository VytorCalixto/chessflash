package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/vytor/chessflash/internal/models"
)

// MockProfileRepository is a mock implementation of repository.ProfileRepository
type MockProfileRepository struct {
	mock.Mock
}

func (m *MockProfileRepository) Get(ctx context.Context, id int64) (*models.Profile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileRepository) List(ctx context.Context) ([]models.Profile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Profile), args.Error(1)
}

func (m *MockProfileRepository) Upsert(ctx context.Context, username string) (*models.Profile, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileRepository) UpdateSync(ctx context.Context, id int64, t time.Time) error {
	args := m.Called(ctx, id, t)
	return args.Error(0)
}

func (m *MockProfileRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
