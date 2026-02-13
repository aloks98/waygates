package service

import (
	"errors"
	"fmt"
	"math"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// L4ProxyService handles business logic for L4 proxies
type L4ProxyService struct {
	repo   repository.L4ProxyRepositoryInterface
	logger *zap.Logger
}

// NewL4ProxyService creates a new L4 proxy service
func NewL4ProxyService(repo repository.L4ProxyRepositoryInterface, logger *zap.Logger) *L4ProxyService {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &L4ProxyService{
		repo:   repo,
		logger: logger.Named("l4-proxy-service"),
	}
}

// CreateL4ProxyRequest holds the data for creating an L4 proxy
type CreateL4ProxyRequest struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	ListenPort  int                    `json:"listen_port"`
	Protocol    string                 `json:"protocol"`
	IsActive    bool                   `json:"is_active"`
	Routes      []CreateL4RouteRequest `json:"routes,omitempty"`
}

// UpdateL4ProxyRequest holds the data for updating an L4 proxy
type UpdateL4ProxyRequest struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	ListenPort  int                    `json:"listen_port"`
	Protocol    string                 `json:"protocol"`
	IsActive    bool                   `json:"is_active"`
	Routes      []CreateL4RouteRequest `json:"routes,omitempty"`
}

// ListL4ProxiesRequest holds parameters for listing L4 proxies
type ListL4ProxiesRequest struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	Search   string `json:"search"`
	Protocol string `json:"protocol"`
	IsActive *bool  `json:"is_active"`
	Sort     string `json:"sort"`
	Order    string `json:"order"`
}

// CreateL4RouteRequest holds the data for creating an L4 route
type CreateL4RouteRequest struct {
	Priority             int                 `json:"priority"`
	MatcherType          string              `json:"matcher_type"`
	SNIHostnames         []string            `json:"sni_hostnames,omitempty"`
	AllowedIPRanges      []string            `json:"allowed_ip_ranges,omitempty"`
	RegexPattern         *string             `json:"regex_pattern,omitempty"`
	Upstreams            []L4UpstreamRequest `json:"upstreams"`
	LoadBalancingPolicy  string              `json:"load_balancing_policy"`
	TLSTerminate         bool                `json:"tls_terminate"`
	TLSPassthrough       bool                `json:"tls_passthrough"`
	ProxyProtocolVersion *string             `json:"proxy_protocol_version,omitempty"`
}

// L4UpstreamRequest holds the data for an L4 upstream
type L4UpstreamRequest struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Weight *int   `json:"weight,omitempty"`
}

// L4 proxy service errors
var (
	ErrL4ProxyNotFound        = errors.New("l4 proxy not found")
	ErrL4ProxyPortConflict    = errors.New("port and protocol combination already in use")
	ErrL4ProxyAlreadyActive   = errors.New("l4 proxy is already active")
	ErrL4ProxyAlreadyInactive = errors.New("l4 proxy is already inactive")
)

// Create creates a new L4 proxy
func (s *L4ProxyService) Create(req *CreateL4ProxyRequest, createdBy int) (*models.L4Proxy, error) {
	// Convert request to model
	proxy := s.requestToModel(req)
	proxy.CreatedBy = createdBy

	// Validate the model
	if err := proxy.Validate(); err != nil {
		return nil, err
	}

	// Check for port conflict
	if err := s.checkPortConflict(proxy.ListenPort, proxy.Protocol, 0); err != nil {
		return nil, err
	}

	// Create in database
	if err := s.repo.Create(proxy); err != nil {
		return nil, fmt.Errorf("failed to create l4 proxy: %w", err)
	}

	s.logger.Info("l4 proxy created",
		zap.Int("id", proxy.ID),
		zap.String("name", proxy.Name),
		zap.Int("listen_port", proxy.ListenPort),
		zap.String("protocol", proxy.Protocol),
		zap.Int("created_by", createdBy),
	)

	return proxy, nil
}

// GetByID retrieves an L4 proxy by ID
func (s *L4ProxyService) GetByID(id int) (*models.L4Proxy, error) {
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrL4ProxyNotFound
		}
		return nil, fmt.Errorf("failed to get l4 proxy: %w", err)
	}
	return proxy, nil
}

// List returns a paginated list of L4 proxies
func (s *L4ProxyService) List(req *ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	// Get proxies from database
	proxies, total, err := s.repo.List(repository.L4ProxyListParams{
		Page:     req.Page,
		Limit:    req.Limit,
		Search:   req.Search,
		Protocol: req.Protocol,
		IsActive: req.IsActive,
		Sort:     req.Sort,
		Order:    req.Order,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list l4 proxies: %w", err)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))

	return &models.L4ProxyListResponse{
		Items:      proxies,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing L4 proxy
func (s *L4ProxyService) Update(id int, req *UpdateL4ProxyRequest) (*models.L4Proxy, error) {
	// Get existing proxy
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrL4ProxyNotFound
		}
		return nil, fmt.Errorf("failed to get l4 proxy: %w", err)
	}

	// Convert request to model
	proxy := s.updateRequestToModel(req)
	proxy.ID = id
	proxy.CreatedBy = existing.CreatedBy
	proxy.CreatedAt = existing.CreatedAt

	// Validate the model
	if err := proxy.Validate(); err != nil {
		return nil, err
	}

	// Check for port conflict (excluding current proxy)
	portChanged := existing.ListenPort != proxy.ListenPort || existing.Protocol != proxy.Protocol
	if portChanged {
		if err := s.checkPortConflict(proxy.ListenPort, proxy.Protocol, id); err != nil {
			return nil, err
		}
	}

	// Update in database
	if err := s.repo.Update(proxy); err != nil {
		return nil, fmt.Errorf("failed to update l4 proxy: %w", err)
	}

	s.logger.Info("l4 proxy updated",
		zap.Int("id", proxy.ID),
		zap.String("name", proxy.Name),
		zap.Int("listen_port", proxy.ListenPort),
		zap.String("protocol", proxy.Protocol),
	)

	// Fetch the updated proxy with relations
	return s.repo.GetByID(id)
}

// Delete deletes an L4 proxy
func (s *L4ProxyService) Delete(id int) error {
	// Check if proxy exists
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrL4ProxyNotFound
		}
		return fmt.Errorf("failed to get l4 proxy: %w", err)
	}

	// Delete from database
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete l4 proxy: %w", err)
	}

	s.logger.Info("l4 proxy deleted",
		zap.Int("id", id),
		zap.String("name", proxy.Name),
	)

	return nil
}

// ToggleActive toggles the active status of an L4 proxy
func (s *L4ProxyService) ToggleActive(id int) (*models.L4Proxy, error) {
	// Get existing proxy
	proxy, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrL4ProxyNotFound
		}
		return nil, fmt.Errorf("failed to get l4 proxy: %w", err)
	}

	// Toggle status
	proxy.IsActive = !proxy.IsActive

	// Update in database
	if err := s.repo.Update(proxy); err != nil {
		return nil, fmt.Errorf("failed to toggle l4 proxy status: %w", err)
	}

	s.logger.Info("l4 proxy status toggled",
		zap.Int("id", id),
		zap.String("name", proxy.Name),
		zap.Bool("is_active", proxy.IsActive),
	)

	return proxy, nil
}

// GetStats returns statistics about L4 proxies
func (s *L4ProxyService) GetStats() (*models.L4ProxyStats, error) {
	// Get all proxies with routes to calculate stats
	proxies, _, err := s.repo.List(repository.L4ProxyListParams{
		Page:  1,
		Limit: 10000, // Get all for stats
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get l4 proxy stats: %w", err)
	}

	stats := &models.L4ProxyStats{}

	for i := range proxies {
		stats.TotalProxies++

		if proxies[i].IsActive {
			stats.ActiveProxies++
		}

		switch proxies[i].Protocol {
		case models.L4ProtocolTCP:
			stats.TCPProxies++
		case models.L4ProtocolUDP:
			stats.UDPProxies++
		}

		stats.TotalRoutes += int64(len(proxies[i].Routes))

		for j := range proxies[i].Routes {
			stats.TotalUpstreams += int64(len(proxies[i].Routes[j].Upstreams))
		}
	}

	return stats, nil
}

// checkPortConflict checks if a port+protocol combination is already in use
func (s *L4ProxyService) checkPortConflict(port int, protocol string, excludeID int) error {
	existing, err := s.repo.GetByPort(port, protocol)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No conflict
			return nil
		}
		return fmt.Errorf("failed to check port conflict: %w", err)
	}

	// If we're updating and the found proxy is the same one, no conflict
	if excludeID > 0 && existing.ID == excludeID {
		return nil
	}

	return ErrL4ProxyPortConflict
}

// requestToModel converts a CreateL4ProxyRequest to a models.L4Proxy
func (s *L4ProxyService) requestToModel(req *CreateL4ProxyRequest) *models.L4Proxy {
	proxy := &models.L4Proxy{
		Name:        req.Name,
		Description: req.Description,
		ListenPort:  req.ListenPort,
		Protocol:    req.Protocol,
		IsActive:    req.IsActive,
		Routes:      make([]models.L4Route, 0, len(req.Routes)),
	}

	for i := range req.Routes {
		route := s.routeRequestToModel(&req.Routes[i])
		proxy.Routes = append(proxy.Routes, route)
	}

	return proxy
}

// updateRequestToModel converts an UpdateL4ProxyRequest to a models.L4Proxy
func (s *L4ProxyService) updateRequestToModel(req *UpdateL4ProxyRequest) *models.L4Proxy {
	proxy := &models.L4Proxy{
		Name:        req.Name,
		Description: req.Description,
		ListenPort:  req.ListenPort,
		Protocol:    req.Protocol,
		IsActive:    req.IsActive,
		Routes:      make([]models.L4Route, 0, len(req.Routes)),
	}

	for i := range req.Routes {
		route := s.routeRequestToModel(&req.Routes[i])
		proxy.Routes = append(proxy.Routes, route)
	}

	return proxy
}

// routeRequestToModel converts a CreateL4RouteRequest to a models.L4Route
func (s *L4ProxyService) routeRequestToModel(req *CreateL4RouteRequest) models.L4Route {
	upstreams := make(models.L4UpstreamSlice, 0, len(req.Upstreams))
	for _, u := range req.Upstreams {
		upstreams = append(upstreams, models.L4Upstream{
			Host:   u.Host,
			Port:   u.Port,
			Weight: u.Weight,
		})
	}

	return models.L4Route{
		Priority:             req.Priority,
		MatcherType:          req.MatcherType,
		SNIHostnames:         req.SNIHostnames,
		AllowedIPRanges:      req.AllowedIPRanges,
		RegexPattern:         req.RegexPattern,
		Upstreams:            upstreams,
		LoadBalancingPolicy:  req.LoadBalancingPolicy,
		TLSTerminate:         req.TLSTerminate,
		TLSPassthrough:       req.TLSPassthrough,
		ProxyProtocolVersion: req.ProxyProtocolVersion,
	}
}
