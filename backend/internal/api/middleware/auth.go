package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/aloks98/homelab-proxy/backend/internal/models"
	"github.com/aloks98/homelab-proxy/backend/internal/repository"
	"github.com/aloks98/homelab-proxy/backend/internal/service"
	"github.com/aloks98/homelab-proxy/backend/internal/utils"
)

// UserContextKey is the key for storing user in context
type UserContextKey string

const userKey UserContextKey = "user"

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(tokenService *service.TokenService, userRepo *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.Unauthorized(w, "Authorization header is required")
				return
			}

			// Check if token is in "Bearer <token>" format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				utils.Unauthorized(w, "Invalid Authorization header format")
				return
			}
			tokenString := parts[1]

			// Validate token
			claims, err := tokenService.ValidateToken(tokenString)
			if err != nil {
				utils.Unauthorized(w, "Invalid or expired token")
				return
			}

			// Get user from database
			user, err := userRepo.GetByID(claims.UserID)
			if err != nil {
				utils.Unauthorized(w, "User not found")
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey).(*models.User)
	return user, ok
}
