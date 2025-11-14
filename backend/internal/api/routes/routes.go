package routes

import (
	"time"

	"github.com/aloks98/homelab-proxy/backend/internal/api/handlers"
	"github.com/aloks98/homelab-proxy/backend/internal/api/middleware"
	"github.com/aloks98/homelab-proxy/backend/internal/caddy"
	"github.com/aloks98/homelab-proxy/backend/internal/config"
	"github.com/aloks98/homelab-proxy/backend/internal/repository"
	"github.com/aloks98/homelab-proxy/backend/internal/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SetupRoutes configures all application routes
func SetupRoutes(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Security.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize dependencies
	caddyClient := caddy.NewClient(cfg.Caddy.AdminURL, cfg.Caddy.Timeout)

	// Repositories
	proxyRepo := repository.NewProxyRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Services
	tokenService := service.NewTokenService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenService, cfg.Security)
	proxyService := service.NewProxyService(proxyRepo, caddyClient)

	// Handlers
	healthHandler := handlers.NewHealthHandler()
	authHandler := handlers.NewAuthHandler(authService)
	proxyHandler := handlers.NewProxyHandler(proxyService)
	statusHandler := handlers.NewStatusHandler(caddyClient, userRepo)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/api/health", healthHandler.HealthCheck)
		r.Get("/api/status", statusHandler.GetStatus)
		r.Post("/api/auth/register", authHandler.Register)
		r.Post("/api/auth/login", authHandler.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(tokenService, userRepo))

		// Proxy routes
		r.Route("/api/proxies", func(r chi.Router) {
			r.Get("/", proxyHandler.ListProxies)
			r.Get("/{id}", proxyHandler.GetProxy)
			r.Post("/", proxyHandler.CreateProxy)
			r.Put("/{id}", proxyHandler.UpdateProxy)
			r.Delete("/{id}", proxyHandler.DeleteProxy)
			r.Post("/{id}/enable", proxyHandler.EnableProxy)
			r.Post("/{id}/disable", proxyHandler.DisableProxy)
		})
	})

	return r
}
