package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// PostgreSQL driver
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/api/routes"
	"github.com/aloks98/waygates/backend/internal/auth"
	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/database"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("Starting Caddy Manager Backend",
		zap.String("version", "1.0.0"),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
	)

	// Run database migrations
	logger.Info("Running database migrations...")
	if err := database.RunMigrations(cfg.GetDatabaseURL()); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}
	logger.Info("Database migrations completed successfully")

	// Connect to database (GORM)
	logger.Info("Connecting to PostgreSQL...")
	gormDB, err := database.Connect(cfg.GetDatabaseDSN(), 1) // 1 = Silent mode
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}()
	logger.Info("Database connection established")

	// Get the underlying *sql.DB for goauth
	sqlDB, err := getSQLDB(gormDB)
	if err != nil {
		logger.Fatal("Failed to get SQL DB connection", zap.Error(err))
	}

	// Initialize goauth
	logger.Info("Initializing authentication system...")
	goauthInstance, err := auth.NewAuth(cfg, sqlDB)
	if err != nil {
		logger.Fatal("Failed to initialize goauth", zap.Error(err))
	}
	defer func() {
		if err := goauthInstance.Close(); err != nil {
			logger.Error("Failed to close auth instance", zap.Error(err))
		}
	}()
	logger.Info("Authentication system initialized")

	// Create default user if needed
	userRepo := repository.NewUserRepository(gormDB)
	createDefaultUserIfNeeded(cfg, userRepo, goauthInstance, logger)

	// Setup routes (also starts the background sync and metrics-scraper services)
	router, syncService, metricsScraper := routes.SetupRoutes(cfg, gormDB, logger, goauthInstance.Auth)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.GetServerAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// Stop the background sync loop and wait for any in-flight sync to finish.
	syncService.Stop()
	// Stop the metrics scraper.
	metricsScraper.Stop()

	logger.Info("Server stopped")
}

// getSQLDB extracts the underlying *sql.DB from a GORM DB
func getSQLDB(gormDB *gorm.DB) (*sql.DB, error) {
	return gormDB.DB()
}

// createDefaultUserIfNeeded creates the default admin user if no users exist
func createDefaultUserIfNeeded(cfg *config.Config, userRepo *repository.UserRepository, goauthInstance *auth.Auth, logger *zap.Logger) {
	// Only create a default user if the credentials are provided in the config
	if cfg.DefaultUser.Username == "" || cfg.DefaultUser.Email == "" || cfg.DefaultUser.Password == "" {
		logger.Info("Default user not fully configured, skipping creation.")
		return
	}

	count, err := userRepo.Count()
	if err != nil {
		logger.Error("Failed to count users", zap.Error(err))
		return
	}

	if count == 0 {
		logger.Info("No users found in database, creating default admin user")
		name := cfg.DefaultUser.Name
		if name == "" {
			name = cfg.DefaultUser.Username
		}

		user := &models.User{
			Name:     name,
			Username: cfg.DefaultUser.Username,
			Email:    cfg.DefaultUser.Email,
		}

		if err := user.SetPassword(cfg.DefaultUser.Password, cfg.Security.BcryptCost); err != nil {
			logger.Error("Failed to hash password for default user", zap.Error(err))
			return
		}

		if err := userRepo.Create(user); err != nil {
			logger.Error("Failed to create default user", zap.Error(err))
			return
		}

		// Assign admin role to the default user
		ctx := context.Background()
		userIDStr := fmt.Sprintf("%d", user.ID)
		if err := goauthInstance.AssignRole(ctx, userIDStr, "admin"); err != nil {
			logger.Error("Failed to assign admin role to default user", zap.Error(err))
		}

		logger.Info("Default user created successfully",
			zap.String("username", cfg.DefaultUser.Username),
			zap.String("email", cfg.DefaultUser.Email),
			zap.String("role", "admin"),
		)
	}
}

// initLogger creates and configures a logger based on config
func initLogger(logCfg config.LoggingConfig) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if logCfg.Format == "console" {
		config := zap.NewDevelopmentConfig()
		config.Level = parseLogLevel(logCfg.Level)
		logger, err = config.Build()
	} else {
		config := zap.NewProductionConfig()
		config.Level = parseLogLevel(logCfg.Level)
		logger, err = config.Build()
	}

	if err != nil {
		return nil, err
	}

	return logger, nil
}

// parseLogLevel converts string log level to zapcore.Level
func parseLogLevel(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}
