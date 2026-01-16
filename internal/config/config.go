package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Google       GoogleConfig
	JWT          JWTConfig
	Polar        PolarConfig
	LemonSqueezy LemonSqueezyConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port       string
	Host       string
	APIBaseURL string // External API base URL (e.g., https://api.example.com)
	Env        string // Environment: development, staging, production
}

// IsDevelopment returns true if the environment is development
func (s *ServerConfig) IsDevelopment() bool {
	return s.Env == "development" || s.Env == ""
}

// BaseURL returns the base URL for the application
// If APIBaseURL is configured, it returns that, otherwise constructs from host:port
func (s *ServerConfig) BaseURL(path string) string {
	// Use configured API base URL if available
	if s.APIBaseURL != "" {
		return s.APIBaseURL + path
	}

	// Fall back to host:port
	host := s.Host
	// Handle 0.0.0.0 or empty host - use localhost for URLs
	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%s%s", host, s.Port, path)
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// GoogleConfig holds Google OAuth configuration
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret string
}

// PolarConfig holds Polar payment configuration
type PolarConfig struct {
	AccessToken   string
	ProductID     string
	WebhookSecret string
	Server        string
}

// LemonSqueezyConfig holds Lemon Squeezy payment configuration
type LemonSqueezyConfig struct {
	APIKey        string
	StoreID       string
	WebhookSecret string
	VariantID     string // Product variant ID for paid plan
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port:       getEnv("PORT", "8080"),
			Host:       getEnv("HOST", "0.0.0.0"),
			APIBaseURL: getEnv("API_BASE_URL", ""),
			Env:        getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Google: GoogleConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},
		Polar: PolarConfig{
			AccessToken:   getEnv("POLAR_ACCESS_TOKEN", ""),
			ProductID:     getEnv("POLAR_PRODUCT_ID", ""),
			WebhookSecret: getEnv("POLAR_WEBHOOK_SECRET", ""),
			Server:        getEnv("POLAR_SERVER", "sandbox"),
		},
		LemonSqueezy: LemonSqueezyConfig{
			APIKey:        getEnv("LEMONSQUEEZY_API_KEY", ""),
			StoreID:       getEnv("LEMONSQUEEZY_STORE_ID", ""),
			WebhookSecret: getEnv("LEMONSQUEEZY_WEBHOOK_SECRET", ""),
			VariantID:     getEnv("LEMONSQUEEZY_VARIANT_ID", ""),
		},
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.Google.ClientID == "" || c.Google.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required")
	}
	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
