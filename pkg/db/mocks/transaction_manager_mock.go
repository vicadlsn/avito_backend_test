package mocks

import (
	"context"
)

type MockTransactionManager struct{}

func NewMockTransactionManager() *MockTransactionManager {
	return &MockTransactionManager{}
}

func (m *MockTransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
