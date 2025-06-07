package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	OAuth2      OAuth2Config
	Game        GameConfig
	Leaderboard LeaderboardConfig
	I18n        I18nConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host                string
	Port                string
	JWTSecret           string
	GinMode             string
	StaticFilesEmbedded bool
	EnableMetrics       bool
	EnableHealthCheck   bool
	CORSOrigins         []string
	Debug               bool
	LogLevel            string
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// OAuth2Config holds OAuth2-related configuration
type OAuth2Config struct {
	Provider     string
	ClientID     string
	ClientSecret string
	RedirectURL  string

	// Custom OAuth2 endpoints
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Scopes      []string

	// User info field mappings
	UserIDField     string
	UserEmailField  string
	UserNameField   string
	UserAvatarField string
}

// GameConfig holds game-related configuration
type GameConfig struct {
	VictoryTile        int
	MaxConcurrentGames int
	GameSessionTimeout int
}

// LeaderboardConfig holds leaderboard-related configuration
type LeaderboardConfig struct {
	CacheTTL   int
	MaxEntries int
}

// I18nConfig holds internationalization configuration
type I18nConfig struct {
	DefaultLanguage   string
	SupportedLanguages []string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file from multiple possible locations
	envPaths := []string{
		".env",       // Current directory
		"../.env",    // Parent directory (for backend/ subdirectory)
		"../../.env", // Two levels up (for deeper nesting)
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded environment variables from: %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Println("No .env file found, using environment variables and defaults")
	}

	config := &Config{
		Server: ServerConfig{
			Host:                getEnv("SERVER_HOST", "0.0.0.0"),
			Port:                getEnv("SERVER_PORT", "6060"),
			JWTSecret:           getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
			GinMode:             getEnv("GIN_MODE", "release"),
			StaticFilesEmbedded: getEnvBool("STATIC_FILES_EMBEDDED", true),
			EnableMetrics:       getEnvBool("ENABLE_METRICS", true),
			EnableHealthCheck:   getEnvBool("ENABLE_HEALTH_CHECK", true),
			CORSOrigins:         getEnvSlice("CORS_ORIGINS", []string{"http://localhost:3000", "http://localhost:6060"}),
			Debug:               getEnvBool("DEBUG", false),
			LogLevel:            getEnv("LOG_LEVEL", "info"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "game2048"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		OAuth2: OAuth2Config{
			Provider:     getEnv("OAUTH2_PROVIDER", "custom"),
			ClientID:     getEnv("OAUTH2_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH2_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OAUTH2_REDIRECT_URL", "http://localhost:6060/auth/callback"),

			// Custom OAuth2 endpoints
			AuthURL:     getEnv("OAUTH2_AUTH_URL", ""),
			TokenURL:    getEnv("OAUTH2_TOKEN_URL", ""),
			UserInfoURL: getEnv("OAUTH2_USERINFO_URL", ""),
			Scopes:      getEnvSlice("OAUTH2_SCOPES", []string{"openid", "profile", "email"}),

			// User info field mappings
			UserIDField:     getEnv("OAUTH2_USER_ID_FIELD", "id"),
			UserEmailField:  getEnv("OAUTH2_USER_EMAIL_FIELD", "email"),
			UserNameField:   getEnv("OAUTH2_USER_NAME_FIELD", "name"),
			UserAvatarField: getEnv("OAUTH2_USER_AVATAR_FIELD", "avatar"),
		},
		Game: GameConfig{
			VictoryTile:        getEnvInt("VICTORY_TILE", 16384), // Two 8192 tiles merged
			MaxConcurrentGames: getEnvInt("MAX_CONCURRENT_GAMES", 1000),
			GameSessionTimeout: getEnvInt("GAME_SESSION_TIMEOUT", 3600),
		},
		Leaderboard: LeaderboardConfig{
			CacheTTL:   getEnvInt("LEADERBOARD_CACHE_TTL", 300),
			MaxEntries: getEnvInt("MAX_LEADERBOARD_ENTRIES", 100),
		},
		I18n: I18nConfig{
			DefaultLanguage:    getEnv("DEFAULT_LANGUAGE", "en"),
			SupportedLanguages: getEnvSlice("SUPPORTED_LANGUAGES", []string{"en", "zh-CN", "zh-TW", "ja", "ko", "es", "fr", "de", "ru"}),
		},
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.JWTSecret == "" || c.Server.JWTSecret == "your-super-secret-jwt-key" || c.Server.JWTSecret == "your-super-secret-jwt-key-change-this-in-production" {
		return fmt.Errorf("JWT_SECRET must be set to a secure value")
	}

	if c.OAuth2.ClientID == "" || c.OAuth2.ClientSecret == "" {
		return fmt.Errorf("OAuth2 client ID and secret must be set")
	}

	if c.Database.Host == "" || c.Database.Name == "" || c.Database.User == "" {
		return fmt.Errorf("database configuration is incomplete")
	}

	if c.Game.VictoryTile <= 0 {
		return fmt.Errorf("victory tile must be positive")
	}

	return nil
}

// GetDatabaseURL returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User,
		c.Database.Password, c.Database.Name, c.Database.SSLMode)
}

// GetRedisURL returns the Redis connection URL
func (c *Config) GetRedisURL() string {
	if c.Redis.Password != "" {
		return fmt.Sprintf("redis://:%s@%s:%s/%d",
			c.Redis.Password, c.Redis.Host, c.Redis.Port, c.Redis.DB)
	}
	return fmt.Sprintf("redis://%s:%s/%d",
		c.Redis.Host, c.Redis.Port, c.Redis.DB)
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
