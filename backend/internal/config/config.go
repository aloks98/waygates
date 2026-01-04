package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Caddy       CaddyConfig
	JWT         JWTConfig
	Security    SecurityConfig
	Logging     LoggingConfig
	DefaultUser DefaultUserConfig
	UI          UIConfig
}

// UIConfig holds UI static file serving configuration
type UIConfig struct {
	Enabled bool   // Whether to serve UI static files
	Path    string // Path to the UI dist folder
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

// CaddyConfig holds Caddy configuration
type CaddyConfig struct {
	Email        string // Email for ACME certificates
	ACMEProvider string // DNS provider for ACME challenge: cloudflare, route53, duckdns, digitalocean, hetzner, porkbun, azure, vultr, namecheap, ovh, http, off
}

// ACMEProviderEnvVars maps ACME providers to their required environment variables
var ACMEProviderEnvVars = map[string][]string{
	"cloudflare":   {"CLOUDFLARE_API_TOKEN"},
	"route53":      {"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
	"duckdns":      {"DUCKDNS_API_TOKEN"},
	"digitalocean": {"DO_AUTH_TOKEN"},
	"hetzner":      {"HETZNER_API_TOKEN"},
	"porkbun":      {"PORKBUN_API_KEY", "PORKBUN_API_SECRET_KEY"},
	"azure":        {"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET", "AZURE_SUBSCRIPTION_ID", "AZURE_RESOURCE_GROUP"},
	"vultr":        {"VULTR_API_KEY"},
	"namecheap":    {"NAMECHEAP_API_USER", "NAMECHEAP_API_KEY"},
	"ovh":          {"OVH_ENDPOINT", "OVH_APPLICATION_KEY", "OVH_APPLICATION_SECRET", "OVH_CONSUMER_KEY"},
	"http":         {},
	"off":          {},
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	BcryptCost  int
	CORSOrigins []string
	RBACPath    string // Path to RBAC configuration file
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json" or "console"
}

// DefaultUserConfig holds the default user credentials
type DefaultUserConfig struct {
	Name     string
	Username string
	Email    string
	Password string
}

// Load reads configuration from environment variables and config files
func Load() (*Config, error) {
	// Set default values
	setDefaults()

	// Read from .env file if it exists
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig() // Ignore error if .env doesn't exist

	// Build config struct
	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("SERVER_HOST"),
			Port: viper.GetInt("SERVER_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
		},
		Caddy: CaddyConfig{
			Email:        viper.GetString("CADDY_EMAIL"),
			ACMEProvider: viper.GetString("CADDY_ACME_PROVIDER"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			AccessExpiry:  viper.GetDuration("JWT_ACCESS_EXPIRY"),
			RefreshExpiry: viper.GetDuration("JWT_REFRESH_EXPIRY"),
		},
		Security: SecurityConfig{
			BcryptCost:  viper.GetInt("BCRYPT_COST"),
			CORSOrigins: viper.GetStringSlice("CORS_ORIGINS"),
			RBACPath:    viper.GetString("RBAC_PATH"),
		},
		Logging: LoggingConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
		},
		DefaultUser: DefaultUserConfig{
			Name:     viper.GetString("DEFAULT_USER_NAME"),
			Username: viper.GetString("DEFAULT_USER_USERNAME"),
			Email:    viper.GetString("DEFAULT_USER_EMAIL"),
			Password: viper.GetString("DEFAULT_USER_PASSWORD"),
		},
		UI: UIConfig{
			Enabled: viper.GetBool("UI_ENABLED"),
			Path:    viper.GetString("UI_PATH"),
		},
	}

	// Validate critical configuration
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server
	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 8080)

	// Database (PostgreSQL)
	viper.SetDefault("DB_HOST", "postgres")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "waygates")
	viper.SetDefault("DB_PASSWORD", "waygates")
	viper.SetDefault("DB_NAME", "waygates")

	// Caddy ACME configuration
	viper.SetDefault("CADDY_EMAIL", "")
	viper.SetDefault("CADDY_ACME_PROVIDER", "off") // off, http, cloudflare, route53, duckdns, digitalocean, hetzner, porkbun, azure, vultr, namecheap, ovh

	// JWT
	viper.SetDefault("JWT_ACCESS_EXPIRY", 15*time.Minute)
	viper.SetDefault("JWT_REFRESH_EXPIRY", 7*24*time.Hour)

	// Security
	viper.SetDefault("BCRYPT_COST", 12) // Increased from 10 for better security
	// This should be configured via the .env file for production
	viper.SetDefault("CORS_ORIGINS", []string{})
	viper.SetDefault("RBAC_PATH", "./backend/rbac.yaml")

	// Logging
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	// UI static file serving
	viper.SetDefault("UI_ENABLED", true)
	viper.SetDefault("UI_PATH", "./ui")
}

// MinJWTSecretLength is the minimum required length for JWT secret
const MinJWTSecretLength = 32

// validate checks if required configuration is present
func validate(cfg *Config) error {
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if len(cfg.JWT.Secret) < MinJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters for security", MinJWTSecretLength)
	}

	if cfg.Database.Host == "" || cfg.Database.Name == "" {
		return fmt.Errorf("DB_HOST and DB_NAME are required")
	}

	// Validate ACME provider
	if err := validateACMEProvider(cfg.Caddy.ACMEProvider); err != nil {
		return err
	}

	return nil
}

// validateACMEProvider checks if the ACME provider is valid and required env vars are set
func validateACMEProvider(provider string) error {
	requiredVars, ok := ACMEProviderEnvVars[provider]
	if !ok {
		validProviders := make([]string, 0, len(ACMEProviderEnvVars))
		for p := range ACMEProviderEnvVars {
			validProviders = append(validProviders, p)
		}
		return fmt.Errorf("invalid CADDY_ACME_PROVIDER '%s', valid options: %v", provider, validProviders)
	}

	// Check required environment variables for the provider
	for _, envVar := range requiredVars {
		if viper.GetString(envVar) == "" {
			return fmt.Errorf("CADDY_ACME_PROVIDER '%s' requires %s to be set", provider, envVar)
		}
	}

	return nil
}

// GetDatabaseDSN returns the database connection string for GORM
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
	)
}

// GetDatabaseURL returns the database URL for golang-migrate
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
	)
}

// GetServerAddr returns the server address in host:port format
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
