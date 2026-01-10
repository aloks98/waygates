package auth

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// OAuthProviderID represents a supported OAuth provider
type OAuthProviderID string

// Supported OAuth providers
const (
	OAuthProviderGoogle    OAuthProviderID = "google"
	OAuthProviderGitHub    OAuthProviderID = "github"
	OAuthProviderMicrosoft OAuthProviderID = "microsoft"
	OAuthProviderGitLab    OAuthProviderID = "gitlab"
)

// AllOAuthProviders returns all supported provider IDs
func AllOAuthProviders() []OAuthProviderID {
	return []OAuthProviderID{
		OAuthProviderGoogle,
		OAuthProviderGitHub,
		OAuthProviderMicrosoft,
		OAuthProviderGitLab,
	}
}

// OAuthProvider represents a configured OAuth provider
type OAuthProvider struct {
	ID           OAuthProviderID `json:"id"`
	Name         string          `json:"name"`                // Display name
	ClientID     string          `json:"-"`                   // Never expose in JSON
	ClientSecret string          `json:"-"`                   // Never expose in JSON
	Enabled      bool            `json:"enabled"`             // True if properly configured
	AuthURL      string          `json:"auth_url"`            // OAuth authorize URL
	TokenURL     string          `json:"token_url"`           // OAuth token URL
	UserInfoURL  string          `json:"user_info_url"`       // User info endpoint
	Scopes       []string        `json:"scopes"`              // OAuth scopes
	BaseURL      string          `json:"base_url,omitempty"`  // For GitLab: custom instance URL
	TenantID     string          `json:"tenant_id,omitempty"` // For Microsoft: Azure AD tenant
}

// OAuthProviderPublic is the public representation of a provider (for API responses)
type OAuthProviderPublic struct {
	ID        OAuthProviderID `json:"id"`
	Name      string          `json:"name"`
	Available bool            `json:"available"` // Env vars are configured
	Enabled   bool            `json:"enabled"`   // Admin has enabled it (set externally)
	AuthURL   string          `json:"auth_url,omitempty"`
}

// ToPublic converts an OAuthProvider to its public representation
func (p *OAuthProvider) ToPublic() OAuthProviderPublic {
	return OAuthProviderPublic{
		ID:        p.ID,
		Name:      p.Name,
		Available: p.Enabled, // p.Enabled means env vars are present (available)
		Enabled:   false,     // Will be set by caller based on DB settings
		AuthURL:   p.AuthURL,
	}
}

// OAuthProviderManager manages OAuth provider configurations
type OAuthProviderManager struct {
	mu        sync.RWMutex
	providers map[OAuthProviderID]*OAuthProvider
}

// NewOAuthProviderManager creates a new OAuth provider manager and loads configuration from environment
func NewOAuthProviderManager() *OAuthProviderManager {
	m := &OAuthProviderManager{
		providers: make(map[OAuthProviderID]*OAuthProvider),
	}
	m.loadFromEnvironment()
	return m
}

// loadFromEnvironment loads OAuth provider configuration from environment variables
func (m *OAuthProviderManager) loadFromEnvironment() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize all providers with their static configuration
	m.providers = map[OAuthProviderID]*OAuthProvider{
		OAuthProviderGoogle:    m.loadGoogleProvider(),
		OAuthProviderGitHub:    m.loadGitHubProvider(),
		OAuthProviderMicrosoft: m.loadMicrosoftProvider(),
		OAuthProviderGitLab:    m.loadGitLabProvider(),
	}
}

// loadGoogleProvider loads Google OAuth configuration
func (m *OAuthProviderManager) loadGoogleProvider() *OAuthProvider {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	return &OAuthProvider{
		ID:           OAuthProviderGoogle,
		Name:         "Google",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Enabled:      clientID != "" && clientSecret != "",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
	}
}

// loadGitHubProvider loads GitHub OAuth configuration
func (m *OAuthProviderManager) loadGitHubProvider() *OAuthProvider {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	return &OAuthProvider{
		ID:           OAuthProviderGitHub,
		Name:         "GitHub",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Enabled:      clientID != "" && clientSecret != "",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       []string{"read:user", "user:email"},
	}
}

// loadMicrosoftProvider loads Microsoft OAuth configuration
func (m *OAuthProviderManager) loadMicrosoftProvider() *OAuthProvider {
	clientID := os.Getenv("MICROSOFT_CLIENT_ID")
	clientSecret := os.Getenv("MICROSOFT_CLIENT_SECRET")
	tenantID := os.Getenv("MICROSOFT_TENANT_ID")

	// Default to "common" tenant for multi-tenant apps
	if tenantID == "" {
		tenantID = "common"
	}

	return &OAuthProvider{
		ID:           OAuthProviderMicrosoft,
		Name:         "Microsoft",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Enabled:      clientID != "" && clientSecret != "",
		AuthURL:      fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID),
		TokenURL:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		UserInfoURL:  "https://graph.microsoft.com/v1.0/me",
		Scopes:       []string{"openid", "email", "profile", "User.Read"},
		TenantID:     tenantID,
	}
}

// loadGitLabProvider loads GitLab OAuth configuration
func (m *OAuthProviderManager) loadGitLabProvider() *OAuthProvider {
	clientID := os.Getenv("GITLAB_CLIENT_ID")
	clientSecret := os.Getenv("GITLAB_CLIENT_SECRET")
	baseURL := os.Getenv("GITLAB_BASE_URL")

	// Default to gitlab.com
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &OAuthProvider{
		ID:           OAuthProviderGitLab,
		Name:         "GitLab",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Enabled:      clientID != "" && clientSecret != "",
		AuthURL:      fmt.Sprintf("%s/oauth/authorize", baseURL),
		TokenURL:     fmt.Sprintf("%s/oauth/token", baseURL),
		UserInfoURL:  fmt.Sprintf("%s/api/v4/user", baseURL),
		Scopes:       []string{"openid", "read_user", "email"},
		BaseURL:      baseURL,
	}
}

// GetAvailableProviders returns all providers that have env vars configured
func (m *OAuthProviderManager) GetAvailableProviders() []*OAuthProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var available []*OAuthProvider
	for _, p := range m.providers {
		if p.Enabled { // Enabled here means env vars are configured
			available = append(available, p)
		}
	}
	return available
}

// GetAllProviders returns all supported providers with their availability status
func (m *OAuthProviderManager) GetAllProviders() []*OAuthProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*OAuthProvider, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result
}

// GetAllProvidersPublic returns public info for all providers (with availability status)
func (m *OAuthProviderManager) GetAllProvidersPublic() []OAuthProviderPublic {
	providers := m.GetAllProviders()
	result := make([]OAuthProviderPublic, 0, len(providers))
	for _, p := range providers {
		result = append(result, p.ToPublic())
	}
	return result
}

// GetAvailableProvidersPublic returns public info for available providers
func (m *OAuthProviderManager) GetAvailableProvidersPublic() []OAuthProviderPublic {
	available := m.GetAvailableProviders()
	result := make([]OAuthProviderPublic, 0, len(available))
	for _, p := range available {
		result = append(result, p.ToPublic())
	}
	return result
}

// IsAvailable checks if a provider has env vars configured
func (m *OAuthProviderManager) IsAvailable(id OAuthProviderID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[id]
	return ok && p.Enabled // Enabled means env vars are present
}

// GetEnabledProviders returns enabled providers.
//
// Deprecated: Use GetAvailableProviders instead.
func (m *OAuthProviderManager) GetEnabledProviders() []*OAuthProvider {
	return m.GetAvailableProviders()
}

// GetEnabledProvidersPublic returns enabled providers in public format.
//
// Deprecated: Use GetAvailableProvidersPublic instead.
func (m *OAuthProviderManager) GetEnabledProvidersPublic() []OAuthProviderPublic {
	return m.GetAvailableProvidersPublic()
}

// GetProvider returns a provider by ID
func (m *OAuthProviderManager) GetProvider(id OAuthProviderID) (*OAuthProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[id]
	return p, ok
}

// GetProviderByString returns a provider by string ID
func (m *OAuthProviderManager) GetProviderByString(id string) (*OAuthProvider, bool) {
	return m.GetProvider(OAuthProviderID(id))
}

// IsEnabled checks if a provider is enabled
func (m *OAuthProviderManager) IsEnabled(id OAuthProviderID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[id]
	return ok && p.Enabled
}

// HasEnabledProviders returns true if at least one OAuth provider is enabled
func (m *OAuthProviderManager) HasEnabledProviders() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.providers {
		if p.Enabled {
			return true
		}
	}
	return false
}

// Reload reloads configuration from environment variables
// This is useful if environment variables change at runtime
func (m *OAuthProviderManager) Reload() {
	m.loadFromEnvironment()
}

// ValidateProviderID checks if a provider ID is valid (even if not enabled)
func ValidateProviderID(id string) bool {
	switch OAuthProviderID(id) {
	case OAuthProviderGoogle, OAuthProviderGitHub, OAuthProviderMicrosoft, OAuthProviderGitLab:
		return true
	default:
		return false
	}
}
