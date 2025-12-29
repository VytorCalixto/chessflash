package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/vytor/chessflash/internal/chesscom"
)

// MockChessClient is a mock implementation of chesscom.Client
type MockChessClient struct {
	mock.Mock
}

func (m *MockChessClient) FetchArchives(ctx context.Context, username string) ([]string, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockChessClient) FetchMonthly(ctx context.Context, archiveURL string) ([]chesscom.MonthlyGame, error) {
	args := m.Called(ctx, archiveURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]chesscom.MonthlyGame), args.Error(1)
}
