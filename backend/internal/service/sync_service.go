package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/caddy/config"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// SyncStatus represents the current sync status
type SyncStatus struct {
	LastSyncTime    time.Time `json:"last_sync_time"`
	IsSyncing       bool      `json:"is_syncing"`
	LastSyncSuccess bool      `json:"last_sync_success"`
	LastError       string    `json:"last_error,omitempty"`
	SyncCount       int       `json:"sync_count"`
	LastReloadTime  time.Time `json:"last_reload_time,omitempty"`
	ReloadCount     int       `json:"reload_count"`
	ConfigChanged   bool      `json:"config_changed"`
}

// SyncService handles periodic synchronization between database and Caddy
type SyncService struct {
	proxyRepo    repository.ProxyRepositoryInterface
	settingsRepo repository.SettingsRepositoryInterface
	aclRepo      repository.ACLRepositoryInterface
	l4ProxyRepo  repository.L4ProxyRepositoryInterface
	fileManager  caddy.FileManagerInterface
	reloader     caddy.ReloaderInterface
	logger       *zap.Logger
	email        string
	acmeProvider string

	// JSON configuration builder
	jsonBuilder *config.Builder

	// L4 configuration builder
	l4Builder *config.L4Builder

	// Waygates auth URLs for ACL
	waygatesVerifyURL string
	waygatesLoginURL  string

	// Backup configuration
	configRetentionDays int // Days to retain backups

	// Sync state
	ticker   *time.Ticker
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	status   SyncStatus
}

// SyncServiceConfig holds configuration for the sync service
type SyncServiceConfig struct {
	ProxyRepo    repository.ProxyRepositoryInterface
	SettingsRepo repository.SettingsRepositoryInterface
	ACLRepo      repository.ACLRepositoryInterface     // Optional: for ACL-enabled proxies
	L4ProxyRepo  repository.L4ProxyRepositoryInterface // Optional: for L4 proxy support
	FileManager  caddy.FileManagerInterface
	Reloader     caddy.ReloaderInterface
	Logger       *zap.Logger
	Email        string // Email for ACME certificates
	ACMEProvider string // ACME provider: off, http, cloudflare, route53, etc.

	// JSON mode configuration
	WaygatesVerifyURL string // Waygates auth verify URL for ACL
	WaygatesLoginURL  string // Waygates auth login URL for ACL
	StoragePath       string // Caddy storage path (default: /data)

	// Trusted proxy configuration for real-client-IP resolution behind a
	// tunnel/upstream proxy (empty = Caddy is the edge).
	TrustedProxies  []string
	ClientIPHeaders []string

	// Backup configuration
	ConfigRetentionDays int // Days to retain backups (default: 7)
}

// NewSyncService creates a new sync service
func NewSyncService(cfg SyncServiceConfig) *SyncService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	logger := cfg.Logger.Named("sync-service")

	svc := &SyncService{
		proxyRepo:           cfg.ProxyRepo,
		settingsRepo:        cfg.SettingsRepo,
		aclRepo:             cfg.ACLRepo,
		l4ProxyRepo:         cfg.L4ProxyRepo,
		fileManager:         cfg.FileManager,
		reloader:            cfg.Reloader,
		logger:              logger,
		email:               cfg.Email,
		acmeProvider:        cfg.ACMEProvider,
		waygatesVerifyURL:   cfg.WaygatesVerifyURL,
		waygatesLoginURL:    cfg.WaygatesLoginURL,
		configRetentionDays: cfg.ConfigRetentionDays,
		stopChan:            make(chan struct{}),
		status: SyncStatus{
			LastSyncSuccess: true,
		},
	}

	// Initialize JSON builder
	svc.initJSONBuilder(cfg, logger)

	// Initialize L4 builder if L4 proxy repository is provided
	if cfg.L4ProxyRepo != nil {
		svc.l4Builder = config.NewL4Builder(logger)
	}

	return svc
}

// initJSONBuilder initializes the JSON configuration builder
func (s *SyncService) initJSONBuilder(cfg SyncServiceConfig, logger *zap.Logger) {
	// Always create the ACL builder. Self-contained ACL methods (HTTP basic auth,
	// IP allow/deny/bypass) do not require the Waygates forward-auth URLs; gating
	// the builder's existence on those URLs silently disabled basic-auth/IP ACLs
	// whenever the login URL was unset. The Waygates URLs are only consulted by the
	// forward-auth handler, which is only emitted when a group actually configures
	// Waygates/OAuth/external-provider auth.
	aclBuilder := config.NewACLBuilder(logger)
	aclBuilder.SetWaygatesURLs(cfg.WaygatesVerifyURL, cfg.WaygatesLoginURL)

	// Create builder options
	opts := []config.BuilderOption{
		config.WithLogger(logger),
		config.WithACLBuilder(aclBuilder),
	}

	// Create the JSON builder
	s.jsonBuilder = config.NewBuilder(opts...)

	// Set configuration settings
	storagePath := cfg.StoragePath
	if storagePath == "" {
		storagePath = "/data"
	}

	s.jsonBuilder.SetSettings(&config.Settings{
		AdminEmail:        cfg.Email,
		ACMEProvider:      cfg.ACMEProvider,
		StoragePath:       storagePath,
		WaygatesVerifyURL: cfg.WaygatesVerifyURL,
		WaygatesLoginURL:  cfg.WaygatesLoginURL,
		TrustedProxies:    cfg.TrustedProxies,
		ClientIPHeaders:   cfg.ClientIPHeaders,
	})
}

// Start begins the periodic sync process
func (s *SyncService) Start(interval time.Duration) {
	s.logger.Info("Starting sync service", zap.Duration("interval", interval))

	// Ensure directories exist
	if err := s.fileManager.EnsureDirectories(); err != nil {
		s.logger.Error("Failed to ensure directories", zap.Error(err))
	}

	// Generate initial configs if they don't exist
	if err := s.ensureInitialConfigs(); err != nil {
		s.logger.Error("Failed to ensure initial configs", zap.Error(err))
	}

	// Run initial sync
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Wait a bit for Caddy to be ready, but abort promptly if Stop is
		// called during the delay so graceful shutdown isn't held up.
		select {
		case <-time.After(5 * time.Second):
		case <-s.stopChan:
			return
		}
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

// ensureInitialConfigs creates default config files if they don't exist
// This ensures Caddy can start even before the first sync runs
func (s *SyncService) ensureInitialConfigs() error {
	s.logger.Debug("Ensuring initial config files exist")
	return s.ensureInitialJSONConfig()
}

// ensureInitialJSONConfig creates an initial JSON config if it doesn't exist
func (s *SyncService) ensureInitialJSONConfig() error {
	jsonConfigPath := s.fileManager.GetJSONConfigPath()
	if !s.fileManager.FileExists(jsonConfigPath) {
		s.logger.Info("Creating initial JSON config", zap.String("path", jsonConfigPath))

		// Build a minimal JSON config with no proxies
		s.jsonBuilder.SetHTTPProxies(nil)
		s.jsonBuilder.SetACLGroups(nil)
		s.jsonBuilder.SetACLAssignments(nil)
		s.jsonBuilder.SetLayer4App(nil)
		s.jsonBuilder.SetNotFoundSettings(&models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		})

		configBytes, err := s.jsonBuilder.BuildJSON()
		if err != nil {
			return fmt.Errorf("failed to build initial JSON config: %w", err)
		}

		if err := s.fileManager.WriteJSONConfig(jsonConfigPath, configBytes); err != nil {
			return fmt.Errorf("failed to write initial JSON config: %w", err)
		}
	}

	s.logger.Debug("Initial JSON config ensured")
	return nil
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

	// Perform sync (uses atomic writes, so partial failures are safe)
	// Backup is handled inside performFullSync only when config actually changes
	if err := s.performFullSync(); err != nil {
		s.setError(err)
		return err
	}

	s.mu.Lock()
	s.status.LastSyncSuccess = true
	s.status.LastError = ""
	s.status.SyncCount++
	s.mu.Unlock()

	s.logger.Debug("Full sync completed successfully")
	return nil
}

// performFullSync executes the actual sync logic
func (s *SyncService) performFullSync() error {
	return s.performFullSyncJSON()
}

// performFullSyncJSON executes sync using JSON configuration builder
func (s *SyncService) performFullSyncJSON() error {
	ctx := context.Background()

	s.logger.Debug("Starting JSON config sync")

	// 1. Get all proxies from DB
	proxies, _, err := s.proxyRepo.List(repository.ProxyListParams{
		Limit: 10000, // Get all proxies
	})
	if err != nil {
		return fmt.Errorf("failed to list proxies: %w", err)
	}

	// 2. Get 404 settings from DB
	notFoundSettings, err := s.settingsRepo.GetNotFoundSettings()
	if err != nil {
		s.logger.Warn("Failed to get 404 settings, using default", zap.Error(err))
		notFoundSettings = &models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		}
	}

	// 3. Load ACL groups and assignments if ACL repository is available
	var aclGroups []models.ACLGroup
	var aclAssignments []models.ProxyACLAssignment

	if s.aclRepo != nil {
		// Get all ACL groups
		groups, _, err := s.aclRepo.ListGroups(repository.ACLGroupListParams{
			Limit: 10000, // Get all groups
			Page:  1,
		})
		if err != nil {
			s.logger.Warn("Failed to list ACL groups", zap.Error(err))
		} else {
			// ListGroups preloads all relations, so the groups are ready to use
			// directly — no per-group GetGroupByID query needed.
			aclGroups = groups
		}

		// Get ACL assignments for each proxy
		for i := range proxies {
			assignments, err := s.aclRepo.GetProxyACLAssignments(proxies[i].ID)
			if err != nil {
				s.logger.Warn("Failed to get ACL assignments for proxy",
					zap.Int("proxy_id", proxies[i].ID),
					zap.Error(err))
				continue
			}
			aclAssignments = append(aclAssignments, assignments...)
		}
	}

	// 4. Load L4 proxies if L4 repository is available
	var l4ProxyCount int
	// Clear any existing L4 config first to handle proxy removals
	s.jsonBuilder.SetLayer4App(nil)
	s.jsonBuilder.SetL4TLSHostnames(nil)

	if s.l4ProxyRepo != nil && s.l4Builder != nil {
		isActive := true
		l4Proxies, _, err := s.l4ProxyRepo.List(repository.L4ProxyListParams{
			Limit:    10000, // Get all L4 proxies
			Page:     1,
			IsActive: &isActive, // Only active proxies
		})
		if err != nil {
			s.logger.Warn("Failed to list L4 proxies", zap.Error(err))
		} else if len(l4Proxies) > 0 {
			// Build Layer4 configuration
			layer4App, err := s.l4Builder.BuildL4Config(l4Proxies)
			if err != nil {
				s.logger.Warn("Failed to build L4 config", zap.Error(err))
			} else if layer4App != nil && len(layer4App.Servers) > 0 {
				s.jsonBuilder.SetLayer4App(layer4App)
				l4ProxyCount = len(l4Proxies)
				s.logger.Debug("Built L4 config",
					zap.Int("l4_proxy_count", l4ProxyCount),
					zap.Int("l4_server_count", len(layer4App.Servers)))
			}

			// Extract SNI hostnames that need TLS certificates
			l4TLSHostnames := extractL4TLSHostnames(l4Proxies)
			if len(l4TLSHostnames) > 0 {
				s.jsonBuilder.SetL4TLSHostnames(l4TLSHostnames)
				s.logger.Debug("Extracted L4 TLS hostnames for certificate provisioning",
					zap.Strings("hostnames", l4TLSHostnames))
			}
		}
	}

	// 5. Configure the JSON builder with all data
	// Note: Layer4App is set in step 4 if L4 proxies exist, otherwise it remains nil
	s.jsonBuilder.SetHTTPProxies(proxies)
	s.jsonBuilder.SetACLGroups(aclGroups)
	s.jsonBuilder.SetACLAssignments(aclAssignments)
	s.jsonBuilder.SetNotFoundSettings(notFoundSettings)

	// 6. Build the JSON configuration
	configBytes, err := s.jsonBuilder.BuildJSON()
	if err != nil {
		return fmt.Errorf("failed to build JSON config: %w", err)
	}

	// 7. Get JSON config path from file manager
	jsonConfigPath := s.fileManager.GetJSONConfigPath()

	// 8. Check if config actually changed
	configChanged, err := s.fileManager.ConfigChanged(jsonConfigPath, configBytes)
	if err != nil {
		s.logger.Warn("Failed to check config changes", zap.Error(err))
		// Assume changed if we can't check
		configChanged = true
	}

	// 9. If config hasn't changed, skip backup and reload
	if !configChanged {
		s.logger.Debug("JSON config unchanged, skipping sync")
		s.mu.Lock()
		s.status.ConfigChanged = false
		s.mu.Unlock()
		return nil
	}

	// 10. Backup existing config before overwriting
	if err := s.fileManager.BackupJSONConfig(jsonConfigPath); err != nil {
		s.logger.Warn("Failed to backup JSON config", zap.Error(err))
		// Continue anyway - backup is optional
	}

	// 11. Cleanup old backups by age
	if err := s.fileManager.CleanupOldBackupsByAge(s.configRetentionDays); err != nil {
		s.logger.Warn("Failed to cleanup old backups", zap.Error(err))
	}

	// 12. Write the new JSON config
	if err := s.fileManager.WriteJSONConfig(jsonConfigPath, configBytes); err != nil {
		return fmt.Errorf("failed to write JSON config: %w", err)
	}

	s.logger.Debug("JSON config written", zap.String("path", jsonConfigPath))

	// 13. Validate the JSON configuration
	if err := s.reloader.ValidateJSON(jsonConfigPath); err != nil {
		return fmt.Errorf("JSON config validation failed: %w", err)
	}

	// 14. Reload Caddy with the JSON configuration
	result, err := s.reloader.ReloadJSON(ctx, jsonConfigPath)
	if err != nil {
		return fmt.Errorf("failed to reload Caddy with JSON config: %w", err)
	}

	// Update status
	s.mu.Lock()
	s.status.ConfigChanged = true
	s.status.LastReloadTime = time.Now()
	s.status.ReloadCount++
	s.mu.Unlock()

	s.logger.Info("Caddy JSON configuration reloaded",
		zap.Int("http_proxy_count", len(proxies)),
		zap.Int("l4_proxy_count", l4ProxyCount),
		zap.Int("acl_group_count", len(aclGroups)),
		zap.Duration("reload_duration", result.Duration))

	return nil
}

// SyncProxy syncs a single proxy to file
func (s *SyncService) SyncProxy(proxy *models.Proxy) error {
	return s.SyncProxyWithACL(proxy, nil)
}

// SyncProxyByID syncs a single proxy to file by its ID
func (s *SyncService) SyncProxyByID(proxyID int) error {
	proxy, err := s.proxyRepo.GetByID(proxyID)
	if err != nil {
		return fmt.Errorf("getting proxy: %w", err)
	}
	return s.SyncProxyWithACL(proxy, nil)
}

// SyncProxyWithACL syncs a single proxy with ACL assignments by triggering a full JSON sync.
// This ensures the JSON configuration is rebuilt with all proxies and their ACL settings.
func (s *SyncService) SyncProxyWithACL(proxy *models.Proxy, _ []models.ProxyACLAssignment) error {
	s.logger.Info("Syncing proxy via JSON rebuild",
		zap.Int("proxy_id", proxy.ID),
		zap.String("name", proxy.Name))

	// In JSON mode, we rebuild the entire config rather than individual files
	return s.performFullSyncJSON()
}

// RemoveProxy triggers a JSON config rebuild after a proxy is removed.
// The proxy should already be deleted from the database before calling this.
func (s *SyncService) RemoveProxy(proxyID int, hostname string) error {
	s.logger.Info("Rebuilding config after proxy removal",
		zap.Int("proxy_id", proxyID),
		zap.String("hostname", hostname))

	// In JSON mode, we rebuild the entire config
	return s.performFullSyncJSON()
}

// EnableProxy triggers a JSON config rebuild after a proxy is enabled.
// The proxy's IsActive flag should be updated in the database before calling this.
func (s *SyncService) EnableProxy(proxyID int, hostname string) error {
	s.logger.Info("Rebuilding config after enabling proxy",
		zap.Int("proxy_id", proxyID),
		zap.String("hostname", hostname))

	// In JSON mode, we rebuild the entire config
	return s.performFullSyncJSON()
}

// DisableProxy triggers a JSON config rebuild after a proxy is disabled.
// The proxy's IsActive flag should be updated in the database before calling this.
func (s *SyncService) DisableProxy(proxyID int, hostname string) error {
	s.logger.Info("Rebuilding config after disabling proxy",
		zap.Int("proxy_id", proxyID),
		zap.String("hostname", hostname))

	// In JSON mode, we rebuild the entire config
	return s.performFullSyncJSON()
}

// UpdateCatchAll triggers a JSON config rebuild after catch-all settings change.
func (s *SyncService) UpdateCatchAll() error {
	s.logger.Info("Rebuilding config after catch-all update")

	// In JSON mode, we rebuild the entire config
	return s.performFullSyncJSON()
}

func (s *SyncService) setError(err error) {
	s.mu.Lock()
	s.status.LastSyncSuccess = false
	s.status.LastError = err.Error()
	s.mu.Unlock()
	s.logger.Error("Sync error", zap.Error(err))
}

// sanitizeFilename removes unsafe characters from hostname to create a safe filename
// This must match the logic in caddyfile/builder.go's sanitizeFilename
func sanitizeFilename(hostname string) string {
	// Replace dots and other unsafe chars with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, hostname)

	// Remove consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Trim underscores from ends
	result = strings.Trim(result, "_")

	return result
}

// GetProxyFilename returns the config filename for a proxy (using hostname)
func GetProxyFilename(proxyID int, hostname string) string {
	return strconv.Itoa(proxyID) + "_" + sanitizeFilename(hostname) + ".conf"
}

// extractL4TLSHostnames extracts SNI hostnames from L4 proxies that need TLS certificates.
// This includes routes with TLS termination enabled and TLS/SNI matcher type.
func extractL4TLSHostnames(l4Proxies []models.L4Proxy) []string {
	hostnameSet := make(map[string]bool)

	for i := range l4Proxies {
		proxy := &l4Proxies[i]
		if !proxy.IsActive {
			continue
		}

		for j := range proxy.Routes {
			route := &proxy.Routes[j]
			// Check if this route needs TLS termination
			if !route.TLSTerminate {
				continue
			}

			// For TLS matcher, collect SNI hostnames
			if route.MatcherType == "tls" && len(route.SNIHostnames) > 0 {
				for _, hostname := range route.SNIHostnames {
					if hostname != "" {
						hostnameSet[hostname] = true
					}
				}
			}
		}
	}

	// Convert set to slice
	hostnames := make([]string, 0, len(hostnameSet))
	for hostname := range hostnameSet {
		hostnames = append(hostnames, hostname)
	}

	return hostnames
}
