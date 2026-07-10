package service

import (
	"errors"
	"fmt"
	"math"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/proxygroup"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// ProxyService handles business logic for proxies
type ProxyService struct {
	repo        repository.ProxyRepositoryInterface
	groupRepo   repository.ProxyGroupRepositoryInterface
	syncService ProxySyncer
	logger      *zap.Logger
}

// ProxyServiceConfig holds configuration for ProxyService
type ProxyServiceConfig struct {
	Repo        repository.ProxyRepositoryInterface
	GroupRepo   repository.ProxyGroupRepositoryInterface
	SyncService ProxySyncer
	Logger      *zap.Logger
}

// NewProxyService creates a new proxy service
func NewProxyService(cfg ProxyServiceConfig) *ProxyService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	return &ProxyService{
		repo:        cfg.Repo,
		groupRepo:   cfg.GroupRepo,
		syncService: cfg.SyncService,
		logger:      cfg.Logger.Named("proxy-service"),
	}
}

// ListProxiesRequest holds parameters for listing proxies
type ListProxiesRequest struct {
	Page         int
	Limit        int
	Search       string
	Types        []string // Filter by multiple types
	TypesExclude []string // Exclude types
	Status       string
	StatusNot    string // Exclude status
	SSLEnabled   *bool  // Filter by SSL enabled
	Target       string // Filter by target/upstream address
	GroupID      *int   // Filter by group (nil = no filter)
	GroupIDIn    []int  // Filter by group, IN list
	GroupIDNot   *int   // Exclude a group (ungrouped proxies are still included)
	Ungrouped    bool   // Filter to proxies with no group
	Sort         string
	Order        string
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
		Page:         req.Page,
		Limit:        req.Limit,
		Search:       req.Search,
		Types:        req.Types,
		TypesExclude: req.TypesExclude,
		Status:       req.Status,
		StatusNot:    req.StatusNot,
		SSLEnabled:   req.SSLEnabled,
		Target:       req.Target,
		GroupID:      req.GroupID,
		GroupIDIn:    req.GroupIDIn,
		GroupIDNot:   req.GroupIDNot,
		Ungrouped:    req.Ungrouped,
		Sort:         req.Sort,
		Order:        req.Order,
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
	// materializeHostname must run before Validate: for a label-addressed
	// proxy it writes the real Hostname, which Validate then checks.
	if err := s.materializeHostname(proxy); err != nil {
		return err
	}

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

	// Write proxy config file and reload Caddy
	if err := s.syncService.SyncProxy(proxy); err != nil {
		// Rollback database entry if file sync fails
		if delErr := s.repo.Delete(proxy.ID); delErr != nil {
			return fmt.Errorf("failed to sync proxy config and rollback failed: %w", errors.Join(err, delErr))
		}
		return fmt.Errorf("failed to sync proxy configuration: %w", err)
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

	// materializeHostname must run before Validate, same as CreateProxy.
	// Detaching a proxy from its group (GroupID/HostnameLabel both nil on the
	// incoming proxy) is a no-op: it leaves proxy.Hostname exactly as the
	// caller set it, which — since UpdateProxy's caller sends the existing
	// materialized hostname unchanged in that case — preserves it.
	if err := s.materializeHostname(proxy); err != nil {
		return err
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
	// SSLEnabled / SSLForced / BlockExploits / TLSInsecureSkipVerify are
	// intentionally NOT preserved from `existing`: they are tri-state
	// (nil = inherit from group / system default), and the handler always
	// sends all four explicitly (nil for inherit), so an omitted field means
	// inherit, not "keep existing". See handlers/proxy.go UpdateProxy.
	proxy.CreatedBy = existing.CreatedBy
	proxy.CreatedAt = existing.CreatedAt

	// Clean up configs that don't apply to the new type
	if proxy.Type != models.ProxyTypeReverseProxy {
		proxy.Upstreams = nil
		proxy.LoadBalancing = nil
		proxy.CustomHeaders = models.CustomHeaders{}
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

	// If hostname changed, remove old file first (filename is based on hostname)
	hostnameChanged := existing.Hostname != proxy.Hostname
	if hostnameChanged {
		if err := s.syncService.RemoveProxy(existing.ID, existing.Hostname); err != nil {
			s.logger.Warn("Failed to remove old proxy file", zap.Int("proxy_id", existing.ID), zap.Error(err))
		}
	}

	// Sync the updated proxy
	if err := s.syncService.SyncProxy(proxy); err != nil {
		return fmt.Errorf("failed to sync proxy configuration: %w", err)
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

	// Delete from the database first. RemoveProxy rebuilds the Caddy config
	// from the database, so the row must be gone before we sync; otherwise the
	// deleted proxy is re-emitted into the config and keeps serving traffic
	// until the next periodic sync.
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete proxy: %w", err)
	}

	// Re-sync Caddy now that the proxy is gone. A sync failure here is
	// non-fatal: the proxy is already deleted and the periodic sync will
	// reconcile the live config.
	if err := s.syncService.RemoveProxy(proxy.ID, proxy.Hostname); err != nil {
		s.logger.Warn("Failed to remove proxy config during delete", zap.Int("proxy_id", id), zap.Error(err))
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

	// Enable proxy by renaming file (.disabled -> .conf) and reload
	if err := s.syncService.EnableProxy(proxy.ID, proxy.Hostname); err != nil {
		// Rollback status update
		if rollbackErr := s.repo.UpdateStatus(id, false); rollbackErr != nil {
			return fmt.Errorf("failed to enable proxy file and rollback failed: %w", errors.Join(err, rollbackErr))
		}
		return fmt.Errorf("failed to enable proxy file: %w", err)
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

	// Disable proxy by renaming file (.conf -> .disabled) and reload
	if err := s.syncService.DisableProxy(proxy.ID, proxy.Hostname); err != nil {
		s.logger.Warn("Failed to disable proxy file", zap.Int("proxy_id", id), zap.Error(err))
	}

	return nil
}

// GetStats returns statistics about proxies
func (s *ProxyService) GetStats() (*repository.ProxyStats, error) {
	return s.repo.GetStats()
}

// materializeHostname enforces the cross-table rule that no CHECK constraint
// can express: a proxy is label-addressed iff it has a group AND that group
// has a base_domain. It writes proxies.hostname, the denormalized cache that
// keeps the unique index and every existing proxy.Hostname reader working.
//
// Detaching a proxy from its group (GroupID set to nil) is a no-op here: the
// already-materialized Hostname is left exactly as it is. That is the point of
// the materialized-cache design — a proxy keeps serving its existing hostname
// after detach rather than losing it.
func (s *ProxyService) materializeHostname(p *models.Proxy) error {
	if p.GroupID == nil {
		if p.HostnameLabel != nil {
			return models.ErrLabelRequiresGroup
		}
		return nil
	}

	g, err := s.groupRepo.GetByID(*p.GroupID)
	if err != nil {
		// ProxyGroupRepository.GetByID returns gorm's raw error (no wrapping),
		// so a genuine not-found is gorm.ErrRecordNotFound. Anything else is a
		// transient/DB error and must not be misreported as "group not found"
		// (which the handler maps to 400/404) — surface it as a wrapped 500
		// instead.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("loading group %d: %w", *p.GroupID, err)
	}

	if g.BaseDomain == nil {
		if p.HostnameLabel != nil {
			return ErrLabelRequiresBaseDomain
		}
		return nil // grouped, but addressed absolutely
	}

	if p.HostnameLabel == nil {
		return ErrLabelRequiredByBaseDomain
	}
	p.Hostname = proxygroup.EffectiveHostname(*p.HostnameLabel, *g.BaseDomain)
	return nil
}

// Service errors
var (
	ErrProxyNotFound        = fmt.Errorf("proxy not found")
	ErrHostnameConflict     = fmt.Errorf("hostname already exists")
	ErrProxyAlreadyEnabled  = fmt.Errorf("proxy is already enabled")
	ErrProxyAlreadyDisabled = fmt.Errorf("proxy is already disabled")

	// ErrLabelRequiresBaseDomain / ErrLabelRequiredByBaseDomain express the two
	// halves of the cross-table rule materializeHostname enforces:
	// hostname_label and the group's base_domain must be present together, or
	// neither at all.
	ErrLabelRequiresBaseDomain   = errors.New("hostname_label requires the group to have a base_domain")
	ErrLabelRequiredByBaseDomain = errors.New("group has a base_domain; hostname_label is required")
)

// CaddyError represents an error from Caddy operations
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
