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
	"github.com/aloks98/waygates/backend/internal/caddy/caddyfile"
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
	builder      caddyfile.BuilderInterface
	fileManager  caddy.FileManagerInterface
	reloader     caddy.ReloaderInterface
	logger       *zap.Logger
	email        string
	acmeProvider string

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
	ACLRepo      repository.ACLRepositoryInterface // Optional: for ACL-enabled proxies
	Builder      caddyfile.BuilderInterface
	FileManager  caddy.FileManagerInterface
	Reloader     caddy.ReloaderInterface
	Logger       *zap.Logger
	Email        string // Email for ACME certificates
	ACMEProvider string // ACME provider: off, http, cloudflare, route53, etc.
}

// NewSyncService creates a new sync service
func NewSyncService(cfg SyncServiceConfig) *SyncService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &SyncService{
		proxyRepo:    cfg.ProxyRepo,
		settingsRepo: cfg.SettingsRepo,
		aclRepo:      cfg.ACLRepo,
		builder:      cfg.Builder,
		fileManager:  cfg.FileManager,
		reloader:     cfg.Reloader,
		logger:       cfg.Logger.Named("sync-service"),
		email:        cfg.Email,
		acmeProvider: cfg.ACMEProvider,
		stopChan:     make(chan struct{}),
		status: SyncStatus{
			LastSyncSuccess: true,
		},
	}
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

// ensureInitialConfigs creates default config files if they don't exist
// This ensures Caddy can start even before the first sync runs
func (s *SyncService) ensureInitialConfigs() error {
	s.logger.Debug("Ensuring initial config files exist")

	// Check if main Caddyfile exists
	caddyfilePath := s.fileManager.GetCaddyfilePath()
	if !s.fileManager.FileExists(caddyfilePath) {
		s.logger.Info("Creating initial Caddyfile")
		content := s.builder.BuildMainCaddyfile(caddyfile.MainCaddyfileOptions{
			Email:        s.email,
			ACMEProvider: s.acmeProvider,
		})
		if err := s.fileManager.WriteMainCaddyfile(content); err != nil {
			return fmt.Errorf("failed to write initial Caddyfile: %w", err)
		}
	}

	// Check if catchall.conf exists
	catchallPath := s.fileManager.GetCatchAllPath()
	if !s.fileManager.FileExists(catchallPath) {
		s.logger.Info("Creating initial catchall.conf")
		// Use default settings for initial catchall
		defaultSettings := &models.NotFoundSettings{
			Mode:        "default",
			RedirectURL: "",
		}
		content := s.builder.BuildCatchAllFile(defaultSettings)
		if err := s.fileManager.WriteCatchAllFile(content); err != nil {
			return fmt.Errorf("failed to write initial catchall.conf: %w", err)
		}
	}

	s.logger.Debug("Initial config files ensured")
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

	// Create backup before making changes
	backupPath, err := s.fileManager.Backup()
	if err != nil {
		s.logger.Warn("Failed to create backup", zap.Error(err))
		// Continue anyway - backup is optional
	}

	// Perform sync with rollback support
	if err := s.performFullSync(); err != nil {
		s.setError(err)

		// Attempt rollback if backup was created
		if backupPath != "" {
			s.logger.Warn("Sync failed, attempting rollback", zap.String("backup", backupPath))
			if rollbackErr := s.fileManager.Restore(backupPath); rollbackErr != nil {
				s.logger.Error("Rollback failed", zap.Error(rollbackErr))
			} else {
				// Try to reload with restored config
				ctx := context.Background()
				if _, reloadErr := s.reloader.Reload(ctx); reloadErr != nil {
					s.logger.Error("Reload after rollback failed", zap.Error(reloadErr))
				}
			}
		}

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
	ctx := context.Background()
	configChanged := false

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

	// 3. Write main Caddyfile (only if changed)
	mainContent := s.builder.BuildMainCaddyfile(caddyfile.MainCaddyfileOptions{
		Email:        s.email,
		ACMEProvider: s.acmeProvider,
	})
	if changed, err := s.fileManager.WriteIfChanged(s.fileManager.GetCaddyfilePath(), mainContent); err != nil {
		return fmt.Errorf("failed to write main Caddyfile: %w", err)
	} else if changed {
		configChanged = true
		s.logger.Debug("Main Caddyfile updated")
	}

	// 4. Build set of expected filenames
	expectedFiles := make(map[string]bool)

	// 5. Write proxy files (only if changed)
	for i := range proxies {
		proxy := &proxies[i]

		filename := s.builder.GetProxyFilename(proxy)
		expectedFiles[filename] = true

		// Load ACL assignments for this proxy if ACL repository is available
		var aclAssignments []models.ProxyACLAssignment
		if s.aclRepo != nil {
			assignments, err := s.aclRepo.GetProxyACLAssignments(proxy.ID)
			if err != nil {
				s.logger.Warn("Failed to get ACL assignments for proxy",
					zap.Int("proxy_id", proxy.ID),
					zap.Error(err))
			} else {
				// Load full ACL group data for each assignment
				for j := range assignments {
					if assignments[j].ACLGroup == nil {
						group, err := s.aclRepo.GetGroupByID(assignments[j].ACLGroupID)
						if err != nil {
							s.logger.Warn("Failed to load ACL group",
								zap.Int("group_id", assignments[j].ACLGroupID),
								zap.Error(err))
							continue
						}
						assignments[j].ACLGroup = group
					}
				}
				aclAssignments = assignments
			}
		}

		// Build proxy config with ACL if available
		content, err := s.builder.BuildProxyFileWithACL(proxy, aclAssignments)
		if err != nil {
			s.logger.Warn("Failed to build proxy config",
				zap.Int("proxy_id", proxy.ID),
				zap.String("name", proxy.Name),
				zap.Error(err))
			continue
		}

		// Write the proxy file only if changed
		proxyPath := s.fileManager.GetProxyFilePath(filename)
		if changed, err := s.fileManager.WriteIfChanged(proxyPath, content); err != nil {
			s.logger.Error("Failed to write proxy file",
				zap.Int("proxy_id", proxy.ID),
				zap.String("filename", filename),
				zap.Error(err))
			continue
		} else if changed {
			configChanged = true
			s.logger.Debug("Proxy file updated",
				zap.String("filename", filename),
				zap.Int("acl_assignments", len(aclAssignments)))
		}

		// Handle enable/disable based on IsActive flag
		if !proxy.IsActive {
			if err := s.fileManager.DisableProxy(filename); err != nil {
				s.logger.Warn("Failed to disable proxy",
					zap.Int("proxy_id", proxy.ID),
					zap.Error(err))
			} else {
				configChanged = true
			}
		}
	}

	// 6. Clean up orphaned files
	enabled, disabled, err := s.fileManager.ListProxyFiles()
	if err != nil {
		s.logger.Warn("Failed to list proxy files", zap.Error(err))
	} else {
		// Check enabled files
		for _, f := range enabled {
			if !expectedFiles[f] {
				s.logger.Info("Removing orphaned proxy file", zap.String("filename", f))
				if err := s.fileManager.DeleteProxyFile(f); err != nil {
					s.logger.Warn("Failed to delete orphaned file", zap.String("filename", f), zap.Error(err))
				} else {
					configChanged = true
				}
			}
		}
		// Check disabled files (ListProxyFiles returns base names without .disabled)
		for _, f := range disabled {
			if !expectedFiles[f] {
				s.logger.Info("Removing orphaned disabled proxy file", zap.String("filename", f))
				if err := s.fileManager.DeleteProxyFile(f); err != nil {
					s.logger.Warn("Failed to delete orphaned file", zap.String("filename", f), zap.Error(err))
				} else {
					configChanged = true
				}
			}
		}
	}

	// 7. Write catch-all config (only if changed)
	catchAllContent := s.builder.BuildCatchAllFile(notFoundSettings)
	if changed, err := s.fileManager.WriteIfChanged(s.fileManager.GetCatchAllPath(), catchAllContent); err != nil {
		return fmt.Errorf("failed to write catch-all config: %w", err)
	} else if changed {
		configChanged = true
		s.logger.Debug("Catch-all config updated")
	}

	// Update status
	s.mu.Lock()
	s.status.ConfigChanged = configChanged
	s.mu.Unlock()

	// 8. Only reload Caddy if configuration changed
	if !configChanged {
		s.logger.Debug("No configuration changes, skipping Caddy reload")
		return nil
	}

	result, err := s.reloader.Reload(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	s.mu.Lock()
	s.status.LastReloadTime = time.Now()
	s.status.ReloadCount++
	s.mu.Unlock()

	s.logger.Info("Caddy configuration reloaded",
		zap.Int("proxy_count", len(proxies)),
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

// SyncProxyWithACL syncs a single proxy to file with ACL assignments.
// If aclAssignments is nil and ACL repo is available, it will load them automatically.
func (s *SyncService) SyncProxyWithACL(proxy *models.Proxy, aclAssignments []models.ProxyACLAssignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Load ACL assignments if not provided and ACL repo is available
	if aclAssignments == nil && s.aclRepo != nil {
		assignments, err := s.aclRepo.GetProxyACLAssignments(proxy.ID)
		if err != nil {
			s.logger.Warn("Failed to get ACL assignments for proxy",
				zap.Int("proxy_id", proxy.ID),
				zap.Error(err))
		} else {
			// Load full ACL group data for each assignment
			for i := range assignments {
				if assignments[i].ACLGroup == nil {
					group, err := s.aclRepo.GetGroupByID(assignments[i].ACLGroupID)
					if err != nil {
						s.logger.Warn("Failed to load ACL group",
							zap.Int("group_id", assignments[i].ACLGroupID),
							zap.Error(err))
						continue
					}
					assignments[i].ACLGroup = group
				}
			}
			aclAssignments = assignments
		}
	}

	// Generate filename and content
	filename := s.builder.GetProxyFilename(proxy)
	content, err := s.builder.BuildProxyFileWithACL(proxy, aclAssignments)
	if err != nil {
		return fmt.Errorf("failed to build proxy config: %w", err)
	}

	// Write the file
	if err := s.fileManager.WriteProxyFile(filename, content); err != nil {
		return fmt.Errorf("failed to write proxy file: %w", err)
	}

	// Handle enable/disable based on IsActive flag
	if !proxy.IsActive {
		if err := s.fileManager.DisableProxy(filename); err != nil {
			return fmt.Errorf("failed to disable proxy: %w", err)
		}
	} else {
		if err := s.fileManager.EnableProxy(filename); err != nil {
			// EnableProxy returns nil if already enabled
			s.logger.Debug("Proxy already enabled or not found", zap.String("filename", filename))
		}
	}

	// Reload Caddy
	if _, err := s.reloader.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	s.logger.Info("Synced proxy",
		zap.Int("proxy_id", proxy.ID),
		zap.String("name", proxy.Name),
		zap.String("filename", filename),
		zap.Int("acl_assignments", len(aclAssignments)))

	return nil
}

// RemoveProxy removes a proxy config file
func (s *SyncService) RemoveProxy(proxyID int, hostname string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Generate filename using hostname (must match GetProxyFilename logic)
	filename := fmt.Sprintf("%d_%s.conf", proxyID, sanitizeFilename(hostname))

	// Delete the file
	if err := s.fileManager.DeleteProxyFile(filename); err != nil {
		return fmt.Errorf("failed to delete proxy file: %w", err)
	}

	// Reload Caddy
	if _, err := s.reloader.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	s.logger.Info("Removed proxy",
		zap.Int("proxy_id", proxyID),
		zap.String("filename", filename))

	return nil
}

// EnableProxy enables a proxy by renaming its config file
func (s *SyncService) EnableProxy(proxyID int, hostname string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	filename := fmt.Sprintf("%d_%s.conf", proxyID, sanitizeFilename(hostname))

	if err := s.fileManager.EnableProxy(filename); err != nil {
		return fmt.Errorf("failed to enable proxy: %w", err)
	}

	// Reload Caddy
	if _, err := s.reloader.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	s.logger.Info("Enabled proxy",
		zap.Int("proxy_id", proxyID),
		zap.String("filename", filename))

	return nil
}

// DisableProxy disables a proxy by renaming its config file
func (s *SyncService) DisableProxy(proxyID int, hostname string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	filename := fmt.Sprintf("%d_%s.conf", proxyID, sanitizeFilename(hostname))

	if err := s.fileManager.DisableProxy(filename); err != nil {
		return fmt.Errorf("failed to disable proxy: %w", err)
	}

	// Reload Caddy
	if _, err := s.reloader.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	s.logger.Info("Disabled proxy",
		zap.Int("proxy_id", proxyID),
		zap.String("filename", filename))

	return nil
}

// UpdateCatchAll updates the catch-all 404 config
func (s *SyncService) UpdateCatchAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Get settings
	notFoundSettings, err := s.settingsRepo.GetNotFoundSettings()
	if err != nil {
		return fmt.Errorf("failed to get 404 settings: %w", err)
	}

	// Build and write catch-all config
	content := s.builder.BuildCatchAllFile(notFoundSettings)
	if err := s.fileManager.WriteCatchAllFile(content); err != nil {
		return fmt.Errorf("failed to write catch-all config: %w", err)
	}

	// Reload Caddy
	if _, err := s.reloader.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	s.logger.Info("Updated catch-all config",
		zap.String("mode", notFoundSettings.Mode))

	return nil
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
