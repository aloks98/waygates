package handlers

import (
	"context"

	"github.com/aloks98/goauth/store"
	"github.com/aloks98/goauth/token"
)

// AuthProvider defines the interface for authentication operations
type AuthProvider interface {
	AssignRole(ctx context.Context, userID, role string) error
	GenerateTokenPair(ctx context.Context, userID string, metadata map[string]any) (*token.Pair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*token.Pair, error)
	RevokeAccessToken(ctx context.Context, accessToken string) error
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	GetUserPermissions(ctx context.Context, userID string) (*store.UserPermissions, error)
}
