package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aloks98/homelab-proxy/backend/internal/api/routes"
	"github.com/aloks98/homelab-proxy/backend/internal/config"
	"github.com/aloks98/homelab-proxy/backend/internal/database"
	"github.com/aloks98/homelab-proxy/backend/internal/repository"
	"github.com/aloks98/homelab-proxy/backend/internal/service"
	"go.uber.org/zap"
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
	defer logger.Sync()

	logger.Info("Starting Caddy Manager Backend",
		zap.String("version", "1.0.0"),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
	)

	// Run database migrations
	logger.Info("Running database migrations...")
	if err := database.RunMigrations(cfg.Database.Type, cfg.GetDatabaseDSN()); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}
	logger.Info("Database migrations completed successfully")

	// Connect to database
	logger.Info("Connecting to database...")
	// Use GORM logger Silent mode for now (we have zap for logging)
	db, err := database.Connect(cfg.Database.Type, cfg.GetDatabaseDSN(), 1) // 1 = Silent mode
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()
	logger.Info("Database connection established")

	// Create default user if needed
	userRepo := repository.NewUserRepository(db)
	tokenService := service.NewTokenService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenService, cfg.Security)
	userService := service.NewUserService(userRepo, authService, cfg, logger)
	userService.CreateDefaultUserIfNeeded()

	// Setup routes
	router := routes.SetupRoutes(cfg, db, logger)

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

	logger.Info("Server stopped")
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
