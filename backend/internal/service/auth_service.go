package service

import (
	"errors"
	"fmt"

	"github.com/aloks98/homelab-proxy/backend/internal/config"
	"github.com/aloks98/homelab-proxy/backend/internal/models"
	"github.com/aloks98/homelab-proxy/backend/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user with this email or username already exists")
)

// AuthService handles authentication logic
type AuthService struct {
	userRepo     *repository.UserRepository
	tokenService *TokenService
	cfg          config.SecurityConfig
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, tokenService *TokenService, cfg config.SecurityConfig) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenService: tokenService,
		cfg:          cfg,
	}
}

// RegisterUser creates a new user
func (s *AuthService) RegisterUser(name, username, email, password string) (*models.User, error) {
	// Check if user already exists
	_, err := s.userRepo.GetByUsernameOrEmail(email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	_, err = s.userRepo.GetByUsernameOrEmail(username)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check for existing user: %w", err)
	}

	// Create new user
	user := &models.User{
		Name:     name,
		Username: username,
		Email:    email,
	}

	// Hash password
	if err := user.SetPassword(password, s.cfg.BcryptCost); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Save user to database
	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// LoginUser authenticates a user and returns tokens
func (s *AuthService) LoginUser(identifier, password string) (string, string, error) {
	// Get user by username or email
	user, err := s.userRepo.GetByUsernameOrEmail(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}

	// Check password
	if !user.CheckPassword(password) {
		return "", "", ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, refreshToken, err := s.tokenService.GenerateTokens(user.ID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	return accessToken, refreshToken, nil
}
