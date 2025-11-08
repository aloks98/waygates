package routes

import (
	"time"

	"github.com/aloks98/homelab-proxy/backend/internal/api/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// SetupRoutes configures all application routes
func SetupRoutes(corsOrigins []string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health handler
	healthHandler := handlers.NewHealthHandler()

	// Public routes (no auth required)
	r.Group(func(r chi.Router) {
		r.Get("/api/health", healthHandler.HealthCheck)
	})

	// Protected routes will be added here
	// r.Group(func(r chi.Router) {
	//     r.Use(authMiddleware)
	//     // Protected endpoints...
	// })

	return r
}
