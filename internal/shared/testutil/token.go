package testutil

import (
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/token"
)

// MockTokenManager is a mock implementation of token.Manager for testing
type MockTokenManager struct {
	GenerateAccessTokenFunc  func(memberID, email string) (string, error)
	GenerateRefreshTokenFunc func(memberID, email string) (string, error)
	ValidateTokenFunc        func(tokenString string) (*token.Claims, error)
}

func (m *MockTokenManager) GenerateAccessToken(memberID, email string) (string, error) {
	if m.GenerateAccessTokenFunc != nil {
		return m.GenerateAccessTokenFunc(memberID, email)
	}
	return "mock-access-token", nil
}

func (m *MockTokenManager) GenerateRefreshToken(memberID, email string) (string, error) {
	if m.GenerateRefreshTokenFunc != nil {
		return m.GenerateRefreshTokenFunc(memberID, email)
	}
	return "mock-refresh-token", nil
}

func (m *MockTokenManager) ValidateToken(tokenString string) (*token.Claims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(tokenString)
	}
	return nil, nil
}

// Ensure MockTokenManager implements token.Manager
var _ token.Manager = (*MockTokenManager)(nil)

// NewMockTokenManager creates a new mock token manager with default behavior
func NewMockTokenManager() *MockTokenManager {
	return &MockTokenManager{}
}
