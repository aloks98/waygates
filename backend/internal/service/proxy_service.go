package service

import (
	"errors"
	"fmt"
	"math"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// ProxyService handles business logic for proxies
type ProxyService struct {
	repo        *repository.ProxyRepository
	caddyClient *caddy.Client
	logger      *zap.Logger
}

// NewProxyService creates a new proxy service
// Note: Startup sync is handled by SyncService, not here
func NewProxyService(
	repo *repository.ProxyRepository,
	caddyClient *caddy.Client,
	logger *zap.Logger,
) *ProxyService {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ProxyService{
		repo:        repo,
		caddyClient: caddyClient,
		logger:      logger.Named("proxy-service"),
	}
}

// ListProxiesRequest holds parameters for listing proxies
type ListProxiesRequest struct {
	Page   int
	Limit  int
	Search string
	Type   string
	Status string
	Sort   string
	Order  string
}

// ListProxies returns a paginated list of proxies
func (s *ProxyService) ListProxies(req ListProxiesRequest) (*models.ProxyListResponse, error) {
	// Validate and set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Get proxies from database
	proxies, total, err := s.repo.List(repository.ProxyListParams{
		Page:   req.Page,
		Limit:  req.Limit,
		Search: req.Search,
		Type:   req.Type,
		Status: req.Status,
		Sort:   req.Sort,
		Order:  req.Order,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list proxies: %w", err)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))

	return &models.ProxyListResponse{
		Items:      proxies,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetProxyByID retrieves a proxy by ID
func (s *ProxyService) GetProxyByID(id int) (*models.Proxy, error) {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProxyNotFound
		}
		return nil, fmt.Errorf("failed to get proxy: %w", err)
	}
	return proxy, nil
}

// CreateProxy creates a new proxy
func (s *ProxyService) CreateProxy(proxy *models.Proxy, userID int) error {
	// Validate
	if err := proxy.Validate(); err != nil {
		return err
	}

	// Check if hostname already exists
	exists, err := s.repo.HostnameExists(proxy.Hostname, 0)
	if err != nil {
		return fmt.Errorf("failed to check hostname: %w", err)
	}
	if exists {
		return ErrHostnameConflict
	}

	// Set creator
	proxy.CreatedBy = userID

	// Create in database first to get ID
	if err := s.repo.Create(proxy); err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	// Apply configuration to Caddy if active
	if proxy.IsActive {
		if err := s.applyCaddyConfig(proxy); err != nil {
			// Rollback database entry if Caddy config fails
			if delErr := s.repo.Delete(proxy.ID); delErr != nil {
				return fmt.Errorf("failed to apply Caddy config and rollback failed: caddy: %w, rollback: %w", err, delErr)
			}
			return fmt.Errorf("failed to apply Caddy configuration: %w", err)
		}
	}

	return nil
}

// UpdateProxy updates an existing proxy
func (s *ProxyService) UpdateProxy(id int, proxy *models.Proxy) error {
	// Get existing proxy
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProxyNotFound
		}
		return fmt.Errorf("failed to get proxy: %w", err)
	}

	// Validate
	if err := proxy.Validate(); err != nil {
		return err
	}

	// Check hostname conflict (excluding current proxy)
	if proxy.Hostname != existing.Hostname {
		exists, err := s.repo.HostnameExists(proxy.Hostname, id)
		if err != nil {
			return fmt.Errorf("failed to check hostname: %w", err)
		}
		if exists {
			return ErrHostnameConflict
		}
	}

	// Preserve fields that shouldn't be changed via update
	proxy.ID = id
	proxy.IsActive = existing.IsActive
	proxy.SSLEnabled = existing.SSLEnabled
	proxy.SSLForced = existing.SSLForced
	proxy.CreatedBy = existing.CreatedBy
	proxy.CreatedAt = existing.CreatedAt

	// Clean up configs that don't apply to the new type
	if proxy.Type != models.ProxyTypeReverseProxy {
		proxy.Upstreams = nil
		proxy.LoadBalancing = nil
		proxy.CustomHeaders = nil
	}
	if proxy.Type != models.ProxyTypeRedirect {
		proxy.RedirectConfig = nil
	}
	if proxy.Type != models.ProxyTypeStatic {
		proxy.StaticConfig = nil
	}

	// Update in database
	if err := s.repo.Update(proxy); err != nil {
		return fmt.Errorf("failed to update proxy: %w", err)
	}

	// Update Caddy configuration
	// Remove old config if hostname changed or was active
	if existing.IsActive {
		if err := s.removeCaddyConfig(existing.ID); err != nil {
			s.logger.Warn("Failed to remove old Caddy config", zap.Int("proxy_id", existing.ID), zap.Error(err))
		}
	}

	// Apply new config if active
	if proxy.IsActive {
		if err := s.applyCaddyConfig(proxy); err != nil {
			return fmt.Errorf("failed to apply Caddy configuration: %w", err)
		}
	}

	return nil
}

// DeleteProxy deletes a proxy
func (s *ProxyService) DeleteProxy(id int) error {
	// Check if proxy exists
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProxyNotFound
		}
		return fmt.Errorf("failed to get proxy: %w", err)
	}

	// Remove configuration from Caddy if it was active
	if proxy.IsActive {
		if err := s.removeCaddyConfig(id); err != nil {
			s.logger.Warn("Failed to remove Caddy config during delete", zap.Int("proxy_id", id), zap.Error(err))
		}
	}

	// Delete from database
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete proxy: %w", err)
	}

	return nil
}

// EnableProxy enables a proxy
func (s *ProxyService) EnableProxy(id int) error {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProxyNotFound
		}
		return fmt.Errorf("failed to get proxy: %w", err)
	}

	if proxy.IsActive {
		return ErrProxyAlreadyEnabled
	}

	// Update database status
	if err := s.repo.UpdateStatus(id, true); err != nil {
		return fmt.Errorf("failed to enable proxy: %w", err)
	}

	// Get updated proxy with active status
	proxy.IsActive = true

	// Apply configuration to Caddy
	if err := s.applyCaddyConfig(proxy); err != nil {
		// Rollback status update
		if rollbackErr := s.repo.UpdateStatus(id, false); rollbackErr != nil {
			return fmt.Errorf("failed to apply Caddy config and rollback failed: caddy: %w, rollback: %w", err, rollbackErr)
		}
		return fmt.Errorf("failed to apply Caddy configuration: %w", err)
	}

	return nil
}

// DisableProxy disables a proxy
func (s *ProxyService) DisableProxy(id int) error {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProxyNotFound
		}
		return fmt.Errorf("failed to get proxy: %w", err)
	}

	if !proxy.IsActive {
		return ErrProxyAlreadyDisabled
	}

	// Update database status
	if err := s.repo.UpdateStatus(id, false); err != nil {
		return fmt.Errorf("failed to disable proxy: %w", err)
	}

	// Remove configuration from Caddy
	if err := s.removeCaddyConfig(id); err != nil {
		s.logger.Warn("Failed to remove Caddy config during disable", zap.Int("proxy_id", id), zap.Error(err))
	}

	return nil
}

// GetStats returns statistics about proxies
func (s *ProxyService) GetStats() (*repository.ProxyStats, error) {
	return s.repo.GetStats()
}

// applyCaddyConfig applies a proxy configuration to Caddy
func (s *ProxyService) applyCaddyConfig(proxy *models.Proxy) error {
	// Ensure HTTP server structure exists
	if err := s.ensureHTTPServerExists(); err != nil {
		return NewCaddyError(fmt.Sprintf("failed to initialize Caddy HTTP server: %v", err))
	}

	// Build Caddy route configuration for this proxy
	newRoute, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		return fmt.Errorf("failed to build Caddy config: %w", err)
	}

	// Get current routes from Caddy
	config, err := s.caddyClient.GetConfig()
	if err != nil {
		return NewCaddyError(fmt.Sprintf("failed to get Caddy config: %v", err))
	}

	// Extract current routes
	routes := s.extractRoutes(config)

	// Remove existing route with same ID and catch-all route (we'll re-add catch-all at end)
	targetID := fmt.Sprintf("proxy_%d", proxy.ID)
	filteredRoutes := make([]interface{}, 0)
	for _, r := range routes {
		if routeMap, ok := r.(map[string]interface{}); ok {
			if id, ok := routeMap["@id"].(string); ok {
				if id == targetID || id == caddy.CatchAllRouteID {
					continue // Skip existing route and catch-all
				}
			}
		}
		filteredRoutes = append(filteredRoutes, r)
	}

	// Add new route
	filteredRoutes = append(filteredRoutes, newRoute)

	// Add catch-all route at the end (always last)
	// Note: SyncService will apply proper 404 settings during periodic sync
	catchAllRoute := caddy.BuildCatchAllRouteSimple()
	filteredRoutes = append(filteredRoutes, catchAllRoute)

	// Update all routes at once
	path := caddy.RoutesPath
	if err := s.caddyClient.PATCH(path, filteredRoutes); err != nil {
		return NewCaddyError(fmt.Sprintf("failed to update routes in Caddy: %v", err))
	}

	return nil
}

// extractRoutes extracts the routes array from Caddy config
func (s *ProxyService) extractRoutes(config map[string]interface{}) []interface{} {
	apps, ok := config["apps"].(map[string]interface{})
	if !ok {
		return []interface{}{}
	}

	httpApp, ok := apps["http"].(map[string]interface{})
	if !ok {
		return []interface{}{}
	}

	servers, ok := httpApp["servers"].(map[string]interface{})
	if !ok {
		return []interface{}{}
	}

	srv0, ok := servers["srv0"].(map[string]interface{})
	if !ok {
		return []interface{}{}
	}

	routes, ok := srv0["routes"].([]interface{})
	if !ok {
		// Routes key doesn't exist, return empty array
		// This will be created when we PATCH
		return []interface{}{}
	}

	return routes
}

// ensureHTTPServerExists ensures the HTTP app and server structure exists in Caddy
// The srv0 server should already exist from the Caddyfile (:443 block)
// This function verifies srv0 exists and initializes routes array if needed
func (s *ProxyService) ensureHTTPServerExists() error {
	config, err := s.caddyClient.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get Caddy config: %w", err)
	}

	// Navigate to srv0 server
	apps, ok := config["apps"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no apps in Caddy config")
	}

	httpApp, ok := apps["http"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no http app in Caddy config - check Caddyfile")
	}

	servers, ok := httpApp["servers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no servers in http app")
	}

	srv0, exists := servers["srv0"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("srv0 server not found - check Caddyfile has :443 block")
	}

	// Initialize routes array if it doesn't exist
	if _, hasRoutes := srv0["routes"]; !hasRoutes {
		// Create empty routes array
		if err := s.caddyClient.PUT(caddy.RoutesPath, []interface{}{}); err != nil {
			return fmt.Errorf("failed to initialize routes array: %w", err)
		}
	}

	return nil
}

// removeCaddyConfig removes a proxy configuration from Caddy
func (s *ProxyService) removeCaddyConfig(proxyID int) error {
	// Get current routes from Caddy
	config, err := s.caddyClient.GetConfig()
	if err != nil {
		// If we can't get config, route might not exist, which is okay
		s.logger.Debug("Could not get Caddy config for removal", zap.Int("proxy_id", proxyID), zap.Error(err))
		return nil //nolint:nilerr // intentional: config might not exist
	}

	// Extract current routes
	routes := s.extractRoutes(config)
	if len(routes) == 0 {
		// No routes to remove
		return nil
	}

	// Filter out the route with matching ID and catch-all (we'll re-add catch-all at end)
	targetID := fmt.Sprintf("proxy_%d", proxyID)
	filteredRoutes := make([]interface{}, 0)
	found := false
	for _, r := range routes {
		if routeMap, ok := r.(map[string]interface{}); ok {
			if id, ok := routeMap["@id"].(string); ok {
				if id == targetID {
					found = true
					continue // Skip this route
				}
				if id == caddy.CatchAllRouteID {
					continue // Skip catch-all, we'll re-add at end
				}
			}
		}
		filteredRoutes = append(filteredRoutes, r)
	}

	if !found {
		// Route doesn't exist, nothing to remove
		return nil
	}

	// Re-add catch-all route at the end (always last)
	// Note: SyncService will apply proper 404 settings during periodic sync
	catchAllRoute := caddy.BuildCatchAllRouteSimple()
	filteredRoutes = append(filteredRoutes, catchAllRoute)

	// Update routes array
	path := caddy.RoutesPath
	if err := s.caddyClient.PATCH(path, filteredRoutes); err != nil {
		return fmt.Errorf("failed to update routes in Caddy: %w", err)
	}

	return nil
}

// Service errors
var (
	ErrProxyNotFound        = fmt.Errorf("proxy not found")
	ErrHostnameConflict     = fmt.Errorf("hostname already exists")
	ErrProxyAlreadyEnabled  = fmt.Errorf("proxy is already enabled")
	ErrProxyAlreadyDisabled = fmt.Errorf("proxy is already disabled")
)

// CaddyError represents an error from Caddy Admin API
type CaddyError struct {
	Message string
}

func (e *CaddyError) Error() string {
	return e.Message
}

// NewCaddyError creates a new CaddyError
func NewCaddyError(message string) error {
	return &CaddyError{Message: message}
}

// IsCaddyError checks if an error is a CaddyError
func IsCaddyError(err error) bool {
	var caddyErr *CaddyError
	return errors.As(err, &caddyErr)
}
