package config

import (
	"errors"
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	DiscordToken string
	LogLevel     string
}

// Load creates a new Config instance by loading values from environment variables
func Load() (*Config, error) {
	config := &Config{
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		LogLevel:     getEnvWithDefault("LOG_LEVEL", "INFO"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that all required configuration values are present and valid
func (c *Config) Validate() error {
	if strings.TrimSpace(c.DiscordToken) == "" {
		return errors.New("DISCORD_TOKEN environment variable is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
	}

	logLevel := strings.ToUpper(c.LogLevel)
	if !validLogLevels[logLevel] {
		return errors.New("LOG_LEVEL must be one of: DEBUG, INFO, WARN, ERROR, FATAL")
	}
	c.LogLevel = logLevel

	return nil
}

// getEnvWithDefault returns the value of an environment variable or a default value if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}