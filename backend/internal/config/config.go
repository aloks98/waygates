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
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string // "sqlite" or "postgres"
	Path     string // For SQLite
	Host     string // For PostgreSQL
	Port     int
	User     string
	Password string
	Name     string
}

// CaddyConfig holds Caddy Admin API configuration
type CaddyConfig struct {
	AdminURL string
	Timeout  time.Duration
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
			Type:     viper.GetString("DB_TYPE"),
			Path:     viper.GetString("DB_PATH"),
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
		},
		Caddy: CaddyConfig{
			AdminURL: viper.GetString("CADDY_ADMIN_URL"),
			Timeout:  viper.GetDuration("CADDY_TIMEOUT"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			AccessExpiry:  viper.GetDuration("JWT_ACCESS_EXPIRY"),
			RefreshExpiry: viper.GetDuration("JWT_REFRESH_EXPIRY"),
		},
		Security: SecurityConfig{
			BcryptCost:  viper.GetInt("BCRYPT_COST"),
			CORSOrigins: viper.GetStringSlice("CORS_ORIGINS"),
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

	// Database
	viper.SetDefault("DB_TYPE", "sqlite")
	viper.SetDefault("DB_PATH", "./backend/data/caddy-manager.db")
	viper.SetDefault("DB_PORT", 5432)

	// Caddy
	viper.SetDefault("CADDY_ADMIN_URL", "http://localhost:2019")
	viper.SetDefault("CADDY_TIMEOUT", 10*time.Second)

	// JWT
	viper.SetDefault("JWT_ACCESS_EXPIRY", 15*time.Minute)
	viper.SetDefault("JWT_REFRESH_EXPIRY", 7*24*time.Hour)

	// Security
	viper.SetDefault("BCRYPT_COST", 10)
	viper.SetDefault("CORS_ORIGINS", []string{"http://localhost:3000"})

	// Logging
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
}

// validate checks if required configuration is present
func validate(cfg *Config) error {
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.Database.Type != "sqlite" && cfg.Database.Type != "postgres" {
		return fmt.Errorf("DB_TYPE must be 'sqlite' or 'postgres'")
	}

	if cfg.Database.Type == "sqlite" && cfg.Database.Path == "" {
		return fmt.Errorf("DB_PATH is required when using SQLite")
	}

	if cfg.Database.Type == "postgres" {
		if cfg.Database.Host == "" || cfg.Database.Name == "" {
			return fmt.Errorf("DB_HOST and DB_NAME are required when using PostgreSQL")
		}
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	if c.Database.Type == "sqlite" {
		return c.Database.Path
	}

	// PostgreSQL DSN
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
	)
}

// GetServerAddr returns the server address in host:port format
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
