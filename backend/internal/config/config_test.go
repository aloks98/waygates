package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
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
	if cfg.Caddy.DisableAutoHTTPS {
		t.Error("Expected default CADDY_DISABLE_AUTO_HTTPS false")
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
	setEnv(t, "CADDY_DISABLE_AUTO_HTTPS", "true")
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
		unsetEnv(t, "CADDY_DISABLE_AUTO_HTTPS")
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
	if !cfg.Caddy.DisableAutoHTTPS {
		t.Error("Expected CADDY_DISABLE_AUTO_HTTPS true")
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
