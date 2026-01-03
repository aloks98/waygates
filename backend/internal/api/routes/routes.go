package routes

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aloks98/goauth"
	chimw "github.com/aloks98/goauth/middleware/chi"
	"github.com/aloks98/waygates/backend/internal/api/handlers"
	"github.com/aloks98/waygates/backend/internal/api/middleware"
	"github.com/aloks98/waygates/backend/internal/auth"
	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
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

	// Serve UI static files if enabled
	if cfg.UI.Enabled && cfg.UI.Path != "" {
		setupStaticFileServer(r, cfg.UI.Path, logger)
	}

	return r
}

// setupStaticFileServer configures static file serving for the SPA
func setupStaticFileServer(r *chi.Mux, uiPath string, logger *zap.Logger) {
	// Check if UI directory exists
	if _, err := os.Stat(uiPath); os.IsNotExist(err) {
		logger.Warn("UI directory not found, static file serving disabled", zap.String("path", uiPath))
		return
	}

	// Get absolute path
	absPath, err := filepath.Abs(uiPath)
	if err != nil {
		logger.Error("Failed to get absolute path for UI", zap.Error(err))
		return
	}

	logger.Info("Serving UI static files", zap.String("path", absPath))

	// Create file server
	fileServer := http.FileServer(http.Dir(absPath))

	// Serve static files with SPA fallback
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// Don't serve API routes
		if strings.HasPrefix(req.URL.Path, "/api") {
			http.NotFound(w, req)
			return
		}

		// Try to serve the file
		path := filepath.Join(absPath, req.URL.Path)

		// Check if file exists
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			// File doesn't exist, serve index.html for SPA routing
			http.ServeFile(w, req, filepath.Join(absPath, "index.html"))
			return
		}

		// Check if it's a directory
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Check for index.html in directory
			indexPath := filepath.Join(path, "index.html")
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				// No index.html, serve root index.html
				http.ServeFile(w, req, filepath.Join(absPath, "index.html"))
				return
			}
		}

		// Serve the file
		fileServer.ServeHTTP(w, req)
	})
}

// spaFileSystem wraps http.FileSystem to serve index.html for missing files
type spaFileSystem struct {
	fs       http.FileSystem
	fallback string
}

func (s spaFileSystem) Open(name string) (http.File, error) {
	f, err := s.fs.Open(name)
	if os.IsNotExist(err) {
		return s.fs.Open(s.fallback)
	}
	return f, err
}

// Readdir implements fs.ReadDirFile for directory listing (disabled for security)
func (s spaFileSystem) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, nil
}
