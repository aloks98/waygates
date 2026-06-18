package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
	viper.AutomaticEnv() // Re-enable reading from environment variables
}

// setEnv is a test helper that sets an environment variable
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set env %s: %v", key, err)
	}
}

// unsetEnv is a test helper that unsets an environment variable
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset env %s: %v", key, err)
	}
}

func TestLoad_Success(t *testing.T) {
	resetViper()
	defer resetViper()

	// Set required environment variables
	setEnv(t, "JWT_SECRET", "this-is-a-very-secure-secret-key-for-testing-purposes")
	setEnv(t, "DB_HOST", "localhost")
	setEnv(t, "DB_NAME", "testdb")
	defer func() {
		unsetEnv(t, "JWT_SECRET")
		unsetEnv(t, "DB_HOST")
		unsetEnv(t, "DB_NAME")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.JWT.Secret != "this-is-a-very-secure-secret-key-for-testing-purposes" {
		t.Errorf("Expected JWT secret to be set, got %s", cfg.JWT.Secret)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected DB_HOST 'localhost', got %s", cfg.Database.Host)
	}

	if cfg.Database.Name != "testdb" {
		t.Errorf("Expected DB_NAME 'testdb', got %s", cfg.Database.Name)
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	resetViper()
	defer resetViper()

	setEnv(t, "JWT_SECRET", "this-is-a-very-secure-secret-key-for-testing-purposes")
	setEnv(t, "CADDY_TRUSTED_PROXIES", "172.18.0.0/16, 127.0.0.1/8")
	setEnv(t, "CADDY_CLIENT_IP_HEADERS", "Cf-Connecting-Ip")
	defer func() {
		unsetEnv(t, "JWT_SECRET")
		unsetEnv(t, "CADDY_TRUSTED_PROXIES")
		unsetEnv(t, "CADDY_CLIENT_IP_HEADERS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	wantProxies := []string{"172.18.0.0/16", "127.0.0.1/8"}
	if len(cfg.Caddy.TrustedProxies) != len(wantProxies) {
		t.Fatalf("expected %d trusted proxies, got %v", len(wantProxies), cfg.Caddy.TrustedProxies)
	}
	for i, want := range wantProxies {
		if cfg.Caddy.TrustedProxies[i] != want {
			t.Errorf("trusted proxy %d: expected %q, got %q", i, want, cfg.Caddy.TrustedProxies[i])
		}
	}

	if len(cfg.Caddy.ClientIPHeaders) != 1 || cfg.Caddy.ClientIPHeaders[0] != "Cf-Connecting-Ip" {
		t.Errorf("expected client IP headers [Cf-Connecting-Ip], got %v", cfg.Caddy.ClientIPHeaders)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	resetViper()
	defer resetViper()

	// Set DB config but not JWT secret
	setEnv(t, "DB_HOST", "localhost")
	setEnv(t, "DB_NAME", "testdb")
	defer func() {
		unsetEnv(t, "DB_HOST")
		unsetEnv(t, "DB_NAME")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Expected error when JWT_SECRET is missing")
	}
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	resetViper()
	defer resetViper()

	// Set a short JWT secret
	setEnv(t, "JWT_SECRET", "short")
	setEnv(t, "DB_HOST", "localhost")
	setEnv(t, "DB_NAME", "testdb")
	defer func() {
		unsetEnv(t, "JWT_SECRET")
		unsetEnv(t, "DB_HOST")
		unsetEnv(t, "DB_NAME")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Expected error when JWT_SECRET is too short")
	}
}

func TestValidate_MissingDatabaseConfig(t *testing.T) {
	// Test validation directly since viper defaults override empty env vars
	cfg := &Config{
		JWT: JWTConfig{
			Secret: "this-is-a-very-secure-secret-key-for-testing-purposes",
		},
		Database: DatabaseConfig{
			Host: "",
			Name: "testdb",
		},
	}

	err := validate(cfg)
	if err == nil {
		t.Error("Expected error when DB_HOST is empty")
	}

	cfg.Database.Host = "localhost"
	cfg.Database.Name = ""
	err = validate(cfg)
	if err == nil {
		t.Error("Expected error when DB_NAME is empty")
	}
}

func TestLoad_Defaults(t *testing.T) {
	resetViper()
	defer resetViper()

	setEnv(t, "JWT_SECRET", "this-is-a-very-secure-secret-key-for-testing-purposes")
	defer unsetEnv(t, "JWT_SECRET")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check server defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default SERVER_HOST '0.0.0.0', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default SERVER_PORT 8080, got %d", cfg.Server.Port)
	}

	// Check database defaults
	if cfg.Database.Port != 5432 {
		t.Errorf("Expected default DB_PORT 5432, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "waygates" {
		t.Errorf("Expected default DB_USER 'waygates', got %s", cfg.Database.User)
	}

	// Check JWT defaults
	if cfg.JWT.AccessExpiry != 15*time.Minute {
		t.Errorf("Expected default JWT_ACCESS_EXPIRY 15m, got %v", cfg.JWT.AccessExpiry)
	}
	if cfg.JWT.RefreshExpiry != 7*24*time.Hour {
		t.Errorf("Expected default JWT_REFRESH_EXPIRY 7d, got %v", cfg.JWT.RefreshExpiry)
	}

	// Check security defaults
	if cfg.Security.BcryptCost != 12 {
		t.Errorf("Expected default BCRYPT_COST 12, got %d", cfg.Security.BcryptCost)
	}

	// Check logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default LOG_LEVEL 'info', got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Expected default LOG_FORMAT 'json', got %s", cfg.Logging.Format)
	}

	// Check UI defaults
	if !cfg.UI.Enabled {
		t.Error("Expected default UI_ENABLED true")
	}
	if cfg.UI.Path != "./ui" {
		t.Errorf("Expected default UI_PATH './ui', got %s", cfg.UI.Path)
	}

	// Check Caddy defaults
	if cfg.Caddy.ACMEProvider != "off" {
		t.Errorf("Expected default CADDY_ACME_PROVIDER 'off', got %s", cfg.Caddy.ACMEProvider)
	}
}

func TestGetDatabaseDSN(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
		},
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if got := cfg.GetDatabaseDSN(); got != expected {
		t.Errorf("GetDatabaseDSN() = %s, want %s", got, expected)
	}
}

func TestGetDatabaseURL(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
		},
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if got := cfg.GetDatabaseURL(); got != expected {
		t.Errorf("GetDatabaseURL() = %s, want %s", got, expected)
	}
}

func TestGetServerAddr(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	expected := "0.0.0.0:8080"
	if got := cfg.GetServerAddr(); got != expected {
		t.Errorf("GetServerAddr() = %s, want %s", got, expected)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				JWT: JWTConfig{
					Secret: "this-is-a-very-secure-secret-key-for-testing",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Name: "testdb",
				},
				Caddy: CaddyConfig{
					ACMEProvider: "off",
				},
			},
			wantErr: false,
		},
		{
			name: "Empty JWT secret",
			cfg: &Config{
				JWT: JWTConfig{
					Secret: "",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Name: "testdb",
				},
				Caddy: CaddyConfig{
					ACMEProvider: "off",
				},
			},
			wantErr: true,
		},
		{
			name: "Short JWT secret",
			cfg: &Config{
				JWT: JWTConfig{
					Secret: "tooshort",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Name: "testdb",
				},
				Caddy: CaddyConfig{
					ACMEProvider: "off",
				},
			},
			wantErr: true,
		},
		{
			name: "Empty DB host",
			cfg: &Config{
				JWT: JWTConfig{
					Secret: "this-is-a-very-secure-secret-key-for-testing",
				},
				Database: DatabaseConfig{
					Host: "",
					Name: "testdb",
				},
				Caddy: CaddyConfig{
					ACMEProvider: "off",
				},
			},
			wantErr: true,
		},
		{
			name: "Empty DB name",
			cfg: &Config{
				JWT: JWTConfig{
					Secret: "this-is-a-very-secure-secret-key-for-testing",
				},
				Database: DatabaseConfig{
					Host: "localhost",
					Name: "",
				},
				Caddy: CaddyConfig{
					ACMEProvider: "off",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_CustomValues(t *testing.T) {
	resetViper()
	defer resetViper()

	// Set custom values
	setEnv(t, "JWT_SECRET", "this-is-a-very-secure-secret-key-for-testing-purposes")
	setEnv(t, "SERVER_HOST", "127.0.0.1")
	setEnv(t, "SERVER_PORT", "9090")
	setEnv(t, "DB_HOST", "db.example.com")
	setEnv(t, "DB_PORT", "5433")
	setEnv(t, "DB_USER", "admin")
	setEnv(t, "DB_PASSWORD", "secret")
	setEnv(t, "DB_NAME", "production")
	setEnv(t, "CADDY_EMAIL", "admin@example.com")
	setEnv(t, "CADDY_ACME_PROVIDER", "cloudflare")
	setEnv(t, "CLOUDFLARE_API_TOKEN", "test-token") // Required for cloudflare provider
	setEnv(t, "BCRYPT_COST", "14")
	setEnv(t, "LOG_LEVEL", "debug")
	setEnv(t, "LOG_FORMAT", "console")
	setEnv(t, "UI_ENABLED", "false")
	setEnv(t, "UI_PATH", "/app/ui")

	defer func() {
		unsetEnv(t, "JWT_SECRET")
		unsetEnv(t, "SERVER_HOST")
		unsetEnv(t, "SERVER_PORT")
		unsetEnv(t, "DB_HOST")
		unsetEnv(t, "DB_PORT")
		unsetEnv(t, "DB_USER")
		unsetEnv(t, "DB_PASSWORD")
		unsetEnv(t, "DB_NAME")
		unsetEnv(t, "CADDY_EMAIL")
		unsetEnv(t, "CADDY_ACME_PROVIDER")
		unsetEnv(t, "CLOUDFLARE_API_TOKEN")
		unsetEnv(t, "BCRYPT_COST")
		unsetEnv(t, "LOG_LEVEL")
		unsetEnv(t, "LOG_FORMAT")
		unsetEnv(t, "UI_ENABLED")
		unsetEnv(t, "UI_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify custom values
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected SERVER_HOST '127.0.0.1', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected SERVER_PORT 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Expected DB_HOST 'db.example.com', got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("Expected DB_PORT 5433, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "admin" {
		t.Errorf("Expected DB_USER 'admin', got %s", cfg.Database.User)
	}
	if cfg.Database.Password != "secret" {
		t.Errorf("Expected DB_PASSWORD 'secret', got %s", cfg.Database.Password)
	}
	if cfg.Caddy.Email != "admin@example.com" {
		t.Errorf("Expected CADDY_EMAIL 'admin@example.com', got %s", cfg.Caddy.Email)
	}
	if cfg.Caddy.ACMEProvider != "cloudflare" {
		t.Errorf("Expected CADDY_ACME_PROVIDER 'cloudflare', got %s", cfg.Caddy.ACMEProvider)
	}
	if cfg.Security.BcryptCost != 14 {
		t.Errorf("Expected BCRYPT_COST 14, got %d", cfg.Security.BcryptCost)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected LOG_LEVEL 'debug', got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "console" {
		t.Errorf("Expected LOG_FORMAT 'console', got %s", cfg.Logging.Format)
	}
	if cfg.UI.Enabled {
		t.Error("Expected UI_ENABLED false")
	}
	if cfg.UI.Path != "/app/ui" {
		t.Errorf("Expected UI_PATH '/app/ui', got %s", cfg.UI.Path)
	}
}

// =============================================================================
// validateACMEProvider Tests
// =============================================================================

func TestValidateACMEProvider_InvalidProvider(t *testing.T) {
	resetViper()
	defer resetViper()

	err := validateACMEProvider("invalid_provider")
	if err == nil {
		t.Error("Expected error for invalid provider")
	}

	// Check error message contains expected text
	if err != nil && !containsAll(err.Error(), []string{"invalid", "CADDY_ACME_PROVIDER", "invalid_provider"}) {
		t.Errorf("Error message should indicate invalid provider: %v", err)
	}
}

func TestValidateACMEProvider_ValidProvidersNoEnvVars(t *testing.T) {
	resetViper()
	defer resetViper()

	// HTTP and OFF providers don't require any env vars
	providersNoEnv := []string{"http", "off"}

	for _, provider := range providersNoEnv {
		t.Run(provider, func(t *testing.T) {
			err := validateACMEProvider(provider)
			if err != nil {
				t.Errorf("Expected no error for provider %s, got: %v", provider, err)
			}
		})
	}
}

func TestValidateACMEProvider_CloudflareMissingToken(t *testing.T) {
	resetViper()
	defer resetViper()

	// Don't set CLOUDFLARE_API_TOKEN
	err := validateACMEProvider("cloudflare")
	if err == nil {
		t.Error("Expected error when CLOUDFLARE_API_TOKEN is missing")
	}
	if err != nil && !containsAll(err.Error(), []string{"cloudflare", "CLOUDFLARE_API_TOKEN"}) {
		t.Errorf("Error should mention cloudflare and required env var: %v", err)
	}
}

func TestValidateACMEProvider_CloudflareWithToken(t *testing.T) {
	resetViper()
	defer resetViper()

	setEnv(t, "CLOUDFLARE_API_TOKEN", "test-token")
	defer unsetEnv(t, "CLOUDFLARE_API_TOKEN")

	err := validateACMEProvider("cloudflare")
	if err != nil {
		t.Errorf("Expected no error when CLOUDFLARE_API_TOKEN is set, got: %v", err)
	}
}

func TestValidateACMEProvider_Route53MissingCredentials(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing both AWS credentials
	err := validateACMEProvider("route53")
	if err == nil {
		t.Error("Expected error when AWS credentials are missing")
	}

	// Set only one credential
	setEnv(t, "AWS_ACCESS_KEY_ID", "test-key")
	defer unsetEnv(t, "AWS_ACCESS_KEY_ID")

	err = validateACMEProvider("route53")
	if err == nil {
		t.Error("Expected error when AWS_SECRET_ACCESS_KEY is missing")
	}
}

func TestValidateACMEProvider_Route53WithCredentials(t *testing.T) {
	resetViper()
	defer resetViper()

	setEnv(t, "AWS_ACCESS_KEY_ID", "test-key")
	setEnv(t, "AWS_SECRET_ACCESS_KEY", "test-secret")
	defer func() {
		unsetEnv(t, "AWS_ACCESS_KEY_ID")
		unsetEnv(t, "AWS_SECRET_ACCESS_KEY")
	}()

	err := validateACMEProvider("route53")
	if err != nil {
		t.Errorf("Expected no error when AWS credentials are set, got: %v", err)
	}
}

func TestValidateACMEProvider_DuckDNS(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing token
	err := validateACMEProvider("duckdns")
	if err == nil {
		t.Error("Expected error when DUCKDNS_API_TOKEN is missing")
	}

	// With token
	setEnv(t, "DUCKDNS_API_TOKEN", "test-token")
	defer unsetEnv(t, "DUCKDNS_API_TOKEN")

	err = validateACMEProvider("duckdns")
	if err != nil {
		t.Errorf("Expected no error when DUCKDNS_API_TOKEN is set, got: %v", err)
	}
}

func TestValidateACMEProvider_DigitalOcean(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing token
	err := validateACMEProvider("digitalocean")
	if err == nil {
		t.Error("Expected error when DO_AUTH_TOKEN is missing")
	}

	// With token
	setEnv(t, "DO_AUTH_TOKEN", "test-token")
	defer unsetEnv(t, "DO_AUTH_TOKEN")

	err = validateACMEProvider("digitalocean")
	if err != nil {
		t.Errorf("Expected no error when DO_AUTH_TOKEN is set, got: %v", err)
	}
}

func TestValidateACMEProvider_Hetzner(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing token
	err := validateACMEProvider("hetzner")
	if err == nil {
		t.Error("Expected error when HETZNER_API_TOKEN is missing")
	}

	// With token
	setEnv(t, "HETZNER_API_TOKEN", "test-token")
	defer unsetEnv(t, "HETZNER_API_TOKEN")

	err = validateACMEProvider("hetzner")
	if err != nil {
		t.Errorf("Expected no error when HETZNER_API_TOKEN is set, got: %v", err)
	}
}

func TestValidateACMEProvider_Porkbun(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing credentials
	err := validateACMEProvider("porkbun")
	if err == nil {
		t.Error("Expected error when porkbun credentials are missing")
	}

	// Set only API key
	setEnv(t, "PORKBUN_API_KEY", "test-key")
	defer unsetEnv(t, "PORKBUN_API_KEY")

	err = validateACMEProvider("porkbun")
	if err == nil {
		t.Error("Expected error when PORKBUN_API_SECRET_KEY is missing")
	}

	// Set both
	setEnv(t, "PORKBUN_API_SECRET_KEY", "test-secret")
	defer unsetEnv(t, "PORKBUN_API_SECRET_KEY")

	err = validateACMEProvider("porkbun")
	if err != nil {
		t.Errorf("Expected no error when porkbun credentials are set, got: %v", err)
	}
}

func TestValidateACMEProvider_Azure(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing all Azure credentials
	err := validateACMEProvider("azure")
	if err == nil {
		t.Error("Expected error when Azure credentials are missing")
	}

	// Set all required Azure credentials
	azureVars := []string{
		"AZURE_TENANT_ID",
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_RESOURCE_GROUP",
	}
	for _, v := range azureVars {
		setEnv(t, v, "test-value")
	}
	defer func() {
		for _, v := range azureVars {
			unsetEnv(t, v)
		}
	}()

	err = validateACMEProvider("azure")
	if err != nil {
		t.Errorf("Expected no error when Azure credentials are set, got: %v", err)
	}
}

func TestValidateACMEProvider_Vultr(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing token
	err := validateACMEProvider("vultr")
	if err == nil {
		t.Error("Expected error when VULTR_API_KEY is missing")
	}

	// With token
	setEnv(t, "VULTR_API_KEY", "test-key")
	defer unsetEnv(t, "VULTR_API_KEY")

	err = validateACMEProvider("vultr")
	if err != nil {
		t.Errorf("Expected no error when VULTR_API_KEY is set, got: %v", err)
	}
}

func TestValidateACMEProvider_Namecheap(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing credentials
	err := validateACMEProvider("namecheap")
	if err == nil {
		t.Error("Expected error when Namecheap credentials are missing")
	}

	// Set both
	setEnv(t, "NAMECHEAP_API_USER", "test-user")
	setEnv(t, "NAMECHEAP_API_KEY", "test-key")
	defer func() {
		unsetEnv(t, "NAMECHEAP_API_USER")
		unsetEnv(t, "NAMECHEAP_API_KEY")
	}()

	err = validateACMEProvider("namecheap")
	if err != nil {
		t.Errorf("Expected no error when Namecheap credentials are set, got: %v", err)
	}
}

func TestValidateACMEProvider_OVH(t *testing.T) {
	resetViper()
	defer resetViper()

	// Missing credentials
	err := validateACMEProvider("ovh")
	if err == nil {
		t.Error("Expected error when OVH credentials are missing")
	}

	// Set all required OVH credentials
	ovhVars := []string{
		"OVH_ENDPOINT",
		"OVH_APPLICATION_KEY",
		"OVH_APPLICATION_SECRET",
		"OVH_CONSUMER_KEY",
	}
	for _, v := range ovhVars {
		setEnv(t, v, "test-value")
	}
	defer func() {
		for _, v := range ovhVars {
			unsetEnv(t, v)
		}
	}()

	err = validateACMEProvider("ovh")
	if err != nil {
		t.Errorf("Expected no error when OVH credentials are set, got: %v", err)
	}
}

func TestACMEProviderEnvVars_AllProvidersHaveEntries(t *testing.T) {
	// Verify the map contains expected providers
	expectedProviders := []string{
		"cloudflare", "route53", "duckdns", "digitalocean",
		"hetzner", "porkbun", "azure", "vultr", "namecheap", "ovh",
		"http", "off",
	}

	for _, provider := range expectedProviders {
		if _, ok := ACMEProviderEnvVars[provider]; !ok {
			t.Errorf("Expected provider %s to be in ACMEProviderEnvVars map", provider)
		}
	}
}

func TestMinJWTSecretLength_Constant(t *testing.T) {
	if MinJWTSecretLength != 32 {
		t.Errorf("Expected MinJWTSecretLength to be 32, got %d", MinJWTSecretLength)
	}
}

// Helper function to check if string contains all substrings
func containsAll(s string, substrings []string) bool {
	for _, sub := range substrings {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
