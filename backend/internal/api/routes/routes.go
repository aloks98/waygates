package routes

import (
	"time"

	"github.com/aloks98/homelab-proxy/backend/internal/api/handlers"
	"github.com/aloks98/homelab-proxy/backend/internal/caddy"
	"github.com/aloks98/homelab-proxy/backend/internal/config"
	"github.com/aloks98/homelab-proxy/backend/internal/repository"
	"github.com/aloks98/homelab-proxy/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

// SetupRoutes configures all application routes
func SetupRoutes(cfg *config.Config, db *gorm.DB) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

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

	// Services
	proxyService := service.NewProxyService(proxyRepo, caddyClient)

	// Handlers
	healthHandler := handlers.NewHealthHandler()
	proxyHandler := handlers.NewProxyHandler(proxyService)

	// Public routes (no auth required)
	r.Group(func(r chi.Router) {
		r.Get("/api/health", healthHandler.HealthCheck)
	})

	// Proxy routes (temporarily public for testing, will add auth later)
	r.Route("/api/proxies", func(r chi.Router) {
		r.Get("/", proxyHandler.ListProxies)
		r.Get("/{id}", proxyHandler.GetProxy)
		r.Post("/", proxyHandler.CreateProxy)
		r.Put("/{id}", proxyHandler.UpdateProxy)
		r.Delete("/{id}", proxyHandler.DeleteProxy)
		r.Post("/{id}/enable", proxyHandler.EnableProxy)
		r.Post("/{id}/disable", proxyHandler.DisableProxy)
	})

	// Protected routes will be added here
	// r.Group(func(r chi.Router) {
	//     r.Use(authMiddleware)
	//     // Protected endpoints...
	// })

	return r
}
