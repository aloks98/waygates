package service

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// Note: TLS certificates are handled automatically by Caddy using the global
// acme_dns cloudflare setting in Caddyfile. No manual TLS policy management needed.

// SyncStatus represents the current sync status
type SyncStatus struct {
	LastSyncTime    time.Time `json:"last_sync_time"`
	IsSyncing       bool      `json:"is_syncing"`
	LastSyncSuccess bool      `json:"last_sync_success"`
	LastError       string    `json:"last_error,omitempty"`
}

// SyncService handles periodic synchronization between database and Caddy
type SyncService struct {
	proxyRepo    *repository.ProxyRepository
	settingsRepo *repository.SettingsRepository
	caddyClient  *caddy.Client
	logger       *zap.Logger

	// Sync state
	ticker   *time.Ticker
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	status   SyncStatus
}

// NewSyncService creates a new sync service
func NewSyncService(
	proxyRepo *repository.ProxyRepository,
	settingsRepo *repository.SettingsRepository,
	caddyClient *caddy.Client,
	logger *zap.Logger,
) *SyncService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SyncService{
		proxyRepo:    proxyRepo,
		settingsRepo: settingsRepo,
		caddyClient:  caddyClient,
		logger:       logger.Named("sync-service"),
		stopChan:     make(chan struct{}),
		status: SyncStatus{
			LastSyncSuccess: true,
		},
	}
}

// Start begins the periodic sync process
func (s *SyncService) Start(interval time.Duration) {
	s.logger.Info("Starting sync service", zap.Duration("interval", interval))

	// Run initial sync
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Wait a bit for Caddy to be ready
		time.Sleep(5 * time.Second)
		if err := s.FullSync(); err != nil {
			s.logger.Error("Initial sync failed", zap.Error(err))
		}
	}()

	// Start periodic sync
	s.ticker = time.NewTicker(interval)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ticker.C:
				if err := s.FullSync(); err != nil {
					s.logger.Error("Periodic sync failed", zap.Error(err))
				}
			case <-s.stopChan:
				s.logger.Info("Sync service stopping")
				return
			}
		}
	}()
}

// Stop stops the periodic sync process
func (s *SyncService) Stop() {
	close(s.stopChan)
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.wg.Wait()
	s.logger.Info("Sync service stopped")
}

// GetStatus returns the current sync status
func (s *SyncService) GetStatus() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// FullSync performs a complete sync of all configurations to Caddy
func (s *SyncService) FullSync() error {
	s.mu.Lock()
	if s.status.IsSyncing {
		s.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	s.status.IsSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.status.IsSyncing = false
		s.status.LastSyncTime = time.Now()
		s.mu.Unlock()
	}()

	s.logger.Debug("Starting full sync")

	// Check Caddy health first
	if err := s.caddyClient.HealthCheck(); err != nil {
		s.setError(fmt.Errorf("Caddy not reachable: %w", err))
		return err
	}

	// Sync proxy routes (TLS is handled automatically by Caddy)
	if err := s.syncProxyRoutes(); err != nil {
		s.setError(fmt.Errorf("Route sync failed: %w", err))
		return err
	}

	s.mu.Lock()
	s.status.LastSyncSuccess = true
	s.status.LastError = ""
	s.mu.Unlock()

	s.logger.Debug("Full sync completed successfully")
	return nil
}

func (s *SyncService) setError(err error) {
	s.mu.Lock()
	s.status.LastSyncSuccess = false
	s.status.LastError = err.Error()
	s.mu.Unlock()
	s.logger.Error("Sync error", zap.Error(err))
}

// syncProxyRoutes syncs all active proxy routes to Caddy
func (s *SyncService) syncProxyRoutes() error {
	// Get all active proxies
	proxies, _, err := s.proxyRepo.List(repository.ProxyListParams{
		Status: "active",
		Limit:  1000,
	})
	if err != nil {
		return err
	}

	// Build routes
	routes := make([]interface{}, 0, len(proxies)+1)
	for i := range proxies {
		proxy := &proxies[i]
		route, err := caddy.BuildRouteConfig(proxy)
		if err != nil {
			s.logger.Warn("Failed to build route",
				zap.Int("proxy_id", proxy.ID),
				zap.Error(err))
			continue
		}
		routes = append(routes, route)
	}

	// Get 404 settings and build catch-all route
	notFoundSettings, err := s.settingsRepo.GetNotFoundSettings()
	if err != nil {
		s.logger.Warn("Failed to get 404 settings, using default", zap.Error(err))
		notFoundSettings = &models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		}
	}

	catchAllRoute := caddy.BuildCatchAllRoute(
		caddy.NotFoundMode(notFoundSettings.Mode),
		notFoundSettings.RedirectURL,
	)
	routes = append(routes, catchAllRoute)

	// Apply to Caddy
	if err := s.caddyClient.PATCH(caddy.RoutesPath, routes); err != nil {
		return err
	}

	s.logger.Debug("Synced proxy routes", zap.Int("count", len(proxies)))
	return nil
}
