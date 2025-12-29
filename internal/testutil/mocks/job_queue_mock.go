package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockJobQueue is a mock implementation of jobs.JobQueue
type MockJobQueue struct {
	mock.Mock
}

func (m *MockJobQueue) EnqueueAnalysis(gameID int64) error {
	args := m.Called(gameID)
	return args.Error(0)
}

func (m *MockJobQueue) EnqueueImport(profileID int64, username string) error {
	args := m.Called(profileID, username)
	return args.Error(0)
}
