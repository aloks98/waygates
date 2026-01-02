package routes

import (
	"log"
	"time"

	"github.com/aloks98/goauth"
	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/aloks98/homelab-proxy/backend/internal/api/handlers"
	"github.com/aloks98/homelab-proxy/backend/internal/api/middleware"
	"github.com/aloks98/homelab-proxy/backend/internal/auth"
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
func SetupRoutes(cfg *config.Config, db *gorm.DB, logger *zap.Logger, goauthInstance *goauth.Auth[*auth.CustomClaims]) *chi.Mux {
	r := chi.NewRouter()

	// Validate CORS configuration - warn if wildcard with credentials
	corsOrigins := cfg.Security.CORSOrigins
	for _, origin := range corsOrigins {
		if origin == "*" {
			log.Println("[SECURITY WARNING] CORS wildcard '*' is configured. This is insecure when AllowCredentials is true.")
			// Replace wildcard with empty to prevent insecure configuration
			// In production, explicit origins should be configured
			corsOrigins = []string{}
			break
		}
	}

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(middleware.BodyLimit(middleware.DefaultBodyLimit)) // 1MB body limit

	// Initialize dependencies
	caddyClient := caddy.NewClient(cfg.Caddy.AdminURL, cfg.Caddy.Timeout)

	// Repositories
	proxyRepo := repository.NewProxyRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Services
	proxyService := service.NewProxyService(proxyRepo, caddyClient, logger)

	// Create auth adapter for middleware
	authAdapter := &auth.Adapter{}
	authAdapter.SetAuth(goauthInstance)

	// Middleware config with custom error handler
	mwConfig := chimw.DefaultConfig()
	mwConfig.ErrorHandler = auth.ErrorHandler()

	// Handlers
	healthHandler := handlers.NewHealthHandlerWithDB(db)
	authHandler := handlers.NewAuthHandler(goauthInstance, userRepo, cfg.Security.BcryptCost)
	proxyHandler := handlers.NewProxyHandler(proxyService)
	statusHandler := handlers.NewStatusHandler(caddyClient, userRepo)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/api/health", healthHandler.HealthCheck)
		r.Get("/api/status", statusHandler.GetStatus)
		r.Post("/api/auth/register", authHandler.Register)
		r.Post("/api/auth/login", authHandler.Login)
		r.Post("/api/auth/refresh", authHandler.RefreshToken)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		// Apply authentication middleware
		r.Use(chimw.Authenticate(authAdapter, authAdapter, mwConfig))

		// Auth routes (require authentication)
		r.Post("/api/auth/logout", authHandler.Logout)
		r.Get("/api/auth/me", authHandler.GetMe)

		// Proxy routes with permission checks
		r.Route("/api/proxies", func(r chi.Router) {
			// Read operations - require proxies:read
			r.With(chimw.RequirePermission(authAdapter, "proxies:read", mwConfig)).Get("/", proxyHandler.ListProxies)
			r.With(chimw.RequirePermission(authAdapter, "proxies:read", mwConfig)).Get("/{id}", proxyHandler.GetProxy)

			// Create operations - require proxies:create
			r.With(chimw.RequirePermission(authAdapter, "proxies:create", mwConfig)).Post("/", proxyHandler.CreateProxy)

			// Update operations - require proxies:update
			r.With(chimw.RequirePermission(authAdapter, "proxies:update", mwConfig)).Put("/{id}", proxyHandler.UpdateProxy)
			r.With(chimw.RequirePermission(authAdapter, "proxies:update", mwConfig)).Post("/{id}/enable", proxyHandler.EnableProxy)
			r.With(chimw.RequirePermission(authAdapter, "proxies:update", mwConfig)).Post("/{id}/disable", proxyHandler.DisableProxy)

			// Delete operations - require proxies:delete
			r.With(chimw.RequirePermission(authAdapter, "proxies:delete", mwConfig)).Delete("/{id}", proxyHandler.DeleteProxy)
		})
	})

	return r
}
