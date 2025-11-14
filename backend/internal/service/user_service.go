package service

import (
	"github.com/aloks98/homelab-proxy/backend/internal/config"
	"github.com/aloks98/homelab-proxy/backend/internal/repository"
	"go.uber.org/zap"
)

// UserService handles user-related business logic
type UserService struct {
	userRepo *repository.UserRepository
	authSvc  *AuthService
	cfg      *config.Config
	logger   *zap.Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository, authSvc *AuthService, cfg *config.Config, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		authSvc:  authSvc,
		cfg:      cfg,
		logger:   logger,
	}
}

// CreateDefaultUserIfNeeded checks if any user exists and creates a default one if not
func (s *UserService) CreateDefaultUserIfNeeded() {
	// Only create a default user if the credentials are provided in the config
	if s.cfg.DefaultUser.Username == "" || s.cfg.DefaultUser.Email == "" || s.cfg.DefaultUser.Password == "" {
		s.logger.Info("Default user not fully configured, skipping creation.")
		return
	}

	count, err := s.userRepo.Count()
	if err != nil {
		s.logger.Error("Failed to count users", zap.Error(err))
		return
	}

	if count == 0 {
		s.logger.Info("No users found in database, creating default admin user")
		name := s.cfg.DefaultUser.Name
		if name == "" {
			name = s.cfg.DefaultUser.Username
		}
		_, err := s.authSvc.RegisterUser(
			name,
			s.cfg.DefaultUser.Username,
			s.cfg.DefaultUser.Email,
			s.cfg.DefaultUser.Password,
		)
		if err != nil {
			s.logger.Error("Failed to create default user", zap.Error(err))
		} else {
			s.logger.Info("Default user created successfully",
				zap.String("username", s.cfg.DefaultUser.Username),
				zap.String("email", s.cfg.DefaultUser.Email),
			)
		}
	}
}
