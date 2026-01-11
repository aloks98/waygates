package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOAuthProviderManager(t *testing.T) {
	// Clear any existing env vars
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	manager := NewOAuthProviderManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.providers)

	// All providers should be present but disabled (no env vars set)
	for _, providerID := range AllOAuthProviders() {
		provider, ok := manager.GetProvider(providerID)
		assert.True(t, ok, "provider %s should exist", providerID)
		assert.False(t, provider.Enabled, "provider %s should be disabled without credentials", providerID)
	}
}

func TestOAuthProviderManager_GoogleProvider(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	// Set Google credentials
	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-client-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-client-secret")

	manager := NewOAuthProviderManager()

	provider, ok := manager.GetProvider(OAuthProviderGoogle)
	require.True(t, ok)
	assert.True(t, provider.Enabled)
	assert.Equal(t, "Google", provider.Name)
	assert.Equal(t, "google-client-id", provider.ClientID)
	assert.Equal(t, "google-client-secret", provider.ClientSecret)
	assert.Equal(t, "https://accounts.google.com/o/oauth2/v2/auth", provider.AuthURL)
	assert.Equal(t, "https://oauth2.googleapis.com/token", provider.TokenURL)
	assert.Equal(t, "https://www.googleapis.com/oauth2/v2/userinfo", provider.UserInfoURL)
	assert.Contains(t, provider.Scopes, "openid")
	assert.Contains(t, provider.Scopes, "email")
	assert.Contains(t, provider.Scopes, "profile")
}

func TestOAuthProviderManager_GitHubProvider(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	// Set GitHub credentials
	_ = os.Setenv("GITHUB_CLIENT_ID", "github-client-id")
	_ = os.Setenv("GITHUB_CLIENT_SECRET", "github-client-secret")

	manager := NewOAuthProviderManager()

	provider, ok := manager.GetProvider(OAuthProviderGitHub)
	require.True(t, ok)
	assert.True(t, provider.Enabled)
	assert.Equal(t, "GitHub", provider.Name)
	assert.Equal(t, "github-client-id", provider.ClientID)
	assert.Equal(t, "github-client-secret", provider.ClientSecret)
	assert.Equal(t, "https://github.com/login/oauth/authorize", provider.AuthURL)
	assert.Equal(t, "https://github.com/login/oauth/access_token", provider.TokenURL)
	assert.Equal(t, "https://api.github.com/user", provider.UserInfoURL)
	assert.Contains(t, provider.Scopes, "read:user")
	assert.Contains(t, provider.Scopes, "user:email")
}

func TestOAuthProviderManager_MicrosoftProvider(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	t.Run("default tenant", func(t *testing.T) {
		_ = os.Setenv("MICROSOFT_CLIENT_ID", "microsoft-client-id")
		_ = os.Setenv("MICROSOFT_CLIENT_SECRET", "microsoft-client-secret")
		defer func() { _ = os.Unsetenv("MICROSOFT_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("MICROSOFT_CLIENT_SECRET") }()

		manager := NewOAuthProviderManager()

		provider, ok := manager.GetProvider(OAuthProviderMicrosoft)
		require.True(t, ok)
		assert.True(t, provider.Enabled)
		assert.Equal(t, "Microsoft", provider.Name)
		assert.Equal(t, "common", provider.TenantID)
		assert.Contains(t, provider.AuthURL, "common")
		assert.Contains(t, provider.TokenURL, "common")
	})

	t.Run("custom tenant", func(t *testing.T) {
		_ = os.Setenv("MICROSOFT_CLIENT_ID", "microsoft-client-id")
		_ = os.Setenv("MICROSOFT_CLIENT_SECRET", "microsoft-client-secret")
		_ = os.Setenv("MICROSOFT_TENANT_ID", "my-tenant-id")
		defer func() { _ = os.Unsetenv("MICROSOFT_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("MICROSOFT_CLIENT_SECRET") }()
		defer func() { _ = os.Unsetenv("MICROSOFT_TENANT_ID") }()

		manager := NewOAuthProviderManager()

		provider, ok := manager.GetProvider(OAuthProviderMicrosoft)
		require.True(t, ok)
		assert.Equal(t, "my-tenant-id", provider.TenantID)
		assert.Contains(t, provider.AuthURL, "my-tenant-id")
		assert.Contains(t, provider.TokenURL, "my-tenant-id")
	})
}

func TestOAuthProviderManager_GitLabProvider(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	t.Run("default gitlab.com", func(t *testing.T) {
		_ = os.Setenv("GITLAB_CLIENT_ID", "gitlab-client-id")
		_ = os.Setenv("GITLAB_CLIENT_SECRET", "gitlab-client-secret")
		defer func() { _ = os.Unsetenv("GITLAB_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GITLAB_CLIENT_SECRET") }()

		manager := NewOAuthProviderManager()

		provider, ok := manager.GetProvider(OAuthProviderGitLab)
		require.True(t, ok)
		assert.True(t, provider.Enabled)
		assert.Equal(t, "GitLab", provider.Name)
		assert.Equal(t, "https://gitlab.com", provider.BaseURL)
		assert.Equal(t, "https://gitlab.com/oauth/authorize", provider.AuthURL)
		assert.Equal(t, "https://gitlab.com/oauth/token", provider.TokenURL)
		assert.Equal(t, "https://gitlab.com/api/v4/user", provider.UserInfoURL)
	})

	t.Run("custom gitlab instance", func(t *testing.T) {
		_ = os.Setenv("GITLAB_CLIENT_ID", "gitlab-client-id")
		_ = os.Setenv("GITLAB_CLIENT_SECRET", "gitlab-client-secret")
		_ = os.Setenv("GITLAB_BASE_URL", "https://gitlab.mycompany.com")
		defer func() { _ = os.Unsetenv("GITLAB_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GITLAB_CLIENT_SECRET") }()
		defer func() { _ = os.Unsetenv("GITLAB_BASE_URL") }()

		manager := NewOAuthProviderManager()

		provider, ok := manager.GetProvider(OAuthProviderGitLab)
		require.True(t, ok)
		assert.Equal(t, "https://gitlab.mycompany.com", provider.BaseURL)
		assert.Equal(t, "https://gitlab.mycompany.com/oauth/authorize", provider.AuthURL)
		assert.Equal(t, "https://gitlab.mycompany.com/oauth/token", provider.TokenURL)
	})

	t.Run("trailing slash removed", func(t *testing.T) {
		_ = os.Setenv("GITLAB_CLIENT_ID", "gitlab-client-id")
		_ = os.Setenv("GITLAB_CLIENT_SECRET", "gitlab-client-secret")
		_ = os.Setenv("GITLAB_BASE_URL", "https://gitlab.mycompany.com/")
		defer func() { _ = os.Unsetenv("GITLAB_CLIENT_ID") }()
		defer func() { _ = os.Unsetenv("GITLAB_CLIENT_SECRET") }()
		defer func() { _ = os.Unsetenv("GITLAB_BASE_URL") }()

		manager := NewOAuthProviderManager()

		provider, ok := manager.GetProvider(OAuthProviderGitLab)
		require.True(t, ok)
		assert.Equal(t, "https://gitlab.mycompany.com", provider.BaseURL)
	})
}

func TestOAuthProviderManager_GetEnabledProviders(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	t.Run("no providers enabled", func(t *testing.T) {
		manager := NewOAuthProviderManager()
		enabled := manager.GetEnabledProviders()
		assert.Empty(t, enabled)
	})

	t.Run("some providers enabled", func(t *testing.T) {
		_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
		_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
		_ = os.Setenv("GITHUB_CLIENT_ID", "github-id")
		_ = os.Setenv("GITHUB_CLIENT_SECRET", "github-secret")
		defer clearOAuthEnvVars()

		manager := NewOAuthProviderManager()
		enabled := manager.GetEnabledProviders()
		assert.Len(t, enabled, 2)

		ids := make(map[OAuthProviderID]bool)
		for _, p := range enabled {
			ids[p.ID] = true
		}
		assert.True(t, ids[OAuthProviderGoogle])
		assert.True(t, ids[OAuthProviderGitHub])
	})
}

func TestOAuthProviderManager_GetEnabledProvidersPublic(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")

	manager := NewOAuthProviderManager()
	public := manager.GetEnabledProvidersPublic()

	require.Len(t, public, 1)
	assert.Equal(t, OAuthProviderGoogle, public[0].ID)
	assert.Equal(t, "Google", public[0].Name)
	// Available is true when env vars are configured
	// Enabled is false by default (needs to be set by caller based on DB settings)
	assert.True(t, public[0].Available)
	assert.False(t, public[0].Enabled) // Will be set by caller
	assert.NotEmpty(t, public[0].AuthURL)
}

func TestOAuthProviderManager_IsEnabled(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	manager := NewOAuthProviderManager()

	assert.False(t, manager.IsEnabled(OAuthProviderGoogle))

	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
	manager.Reload()

	assert.True(t, manager.IsEnabled(OAuthProviderGoogle))
	assert.False(t, manager.IsEnabled(OAuthProviderGitHub))
}

func TestOAuthProviderManager_HasEnabledProviders(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	manager := NewOAuthProviderManager()
	assert.False(t, manager.HasEnabledProviders())

	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
	manager.Reload()

	assert.True(t, manager.HasEnabledProviders())
}

func TestOAuthProviderManager_GetProviderByString(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")

	manager := NewOAuthProviderManager()

	provider, ok := manager.GetProviderByString("google")
	assert.True(t, ok)
	assert.NotNil(t, provider)
	assert.Equal(t, OAuthProviderGoogle, provider.ID)

	provider, ok = manager.GetProviderByString("invalid")
	assert.False(t, ok)
	assert.Nil(t, provider)
}

func TestOAuthProviderManager_Reload(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	manager := NewOAuthProviderManager()
	assert.False(t, manager.IsEnabled(OAuthProviderGoogle))

	// Add credentials and reload
	_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
	_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
	manager.Reload()

	assert.True(t, manager.IsEnabled(OAuthProviderGoogle))

	// Remove credentials and reload
	_ = os.Unsetenv("GOOGLE_CLIENT_ID")
	manager.Reload()

	assert.False(t, manager.IsEnabled(OAuthProviderGoogle))
}

func TestValidateProviderID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"google", true},
		{"github", true},
		{"microsoft", true},
		{"gitlab", true},
		{"invalid", false},
		{"", false},
		{"GOOGLE", false}, // Case sensitive
		{"Google", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.valid, ValidateProviderID(tt.id))
		})
	}
}

func TestOAuthProvider_ToPublic(t *testing.T) {
	provider := &OAuthProvider{
		ID:           OAuthProviderGoogle,
		Name:         "Google",
		ClientID:     "secret-client-id",
		ClientSecret: "secret-client-secret",
		Enabled:      true, // In OAuthProvider, Enabled means env vars are configured
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "email"},
	}

	public := provider.ToPublic()

	assert.Equal(t, OAuthProviderGoogle, public.ID)
	assert.Equal(t, "Google", public.Name)
	// Available reflects whether env vars are configured (from provider.Enabled)
	assert.True(t, public.Available)
	// Enabled is false by default - will be set by caller based on DB settings
	assert.False(t, public.Enabled)
	assert.Equal(t, "https://example.com/auth", public.AuthURL)
}

func TestAllOAuthProviders(t *testing.T) {
	providers := AllOAuthProviders()
	assert.Len(t, providers, 4)
	assert.Contains(t, providers, OAuthProviderGoogle)
	assert.Contains(t, providers, OAuthProviderGitHub)
	assert.Contains(t, providers, OAuthProviderMicrosoft)
	assert.Contains(t, providers, OAuthProviderGitLab)
}

func TestOAuthProviderManager_PartialCredentials(t *testing.T) {
	clearOAuthEnvVars()
	defer clearOAuthEnvVars()

	t.Run("only client id", func(t *testing.T) {
		_ = os.Setenv("GOOGLE_CLIENT_ID", "google-id")
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_ID") }()

		manager := NewOAuthProviderManager()
		assert.False(t, manager.IsEnabled(OAuthProviderGoogle))
	})

	t.Run("only client secret", func(t *testing.T) {
		_ = os.Setenv("GOOGLE_CLIENT_SECRET", "google-secret")
		defer func() { _ = os.Unsetenv("GOOGLE_CLIENT_SECRET") }()

		manager := NewOAuthProviderManager()
		assert.False(t, manager.IsEnabled(OAuthProviderGoogle))
	})
}

// clearOAuthEnvVars clears all OAuth-related environment variables
func clearOAuthEnvVars() {
	envVars := []string{
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
		"MICROSOFT_CLIENT_ID",
		"MICROSOFT_CLIENT_SECRET",
		"MICROSOFT_TENANT_ID",
		"GITLAB_CLIENT_ID",
		"GITLAB_CLIENT_SECRET",
		"GITLAB_BASE_URL",
	}
	for _, v := range envVars {
		_ = os.Unsetenv(v)
	}
}
