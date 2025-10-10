package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	DiscordToken string
	LogLevel     string
	TTS          TTSConfig
}

// TTSConfig holds TTS-specific configuration
type TTSConfig struct {
	GoogleCloudCredentialsPath string
	DefaultVoice               string
	DefaultSpeed               float32
	DefaultVolume              float32
	MaxQueueSize               int
	MaxMessageLength           int
}

// Load creates a new Config instance by loading values from environment variables
func Load() (*Config, error) {
	config := &Config{
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		LogLevel:     getEnvWithDefault("LOG_LEVEL", "INFO"),
		TTS:          loadTTSConfig(),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadTTSConfig loads TTS configuration from environment variables
func loadTTSConfig() TTSConfig {
	defaultSpeed, _ := strconv.ParseFloat(getEnvWithDefault("TTS_DEFAULT_SPEED", "1.0"), 32)
	defaultVolume, _ := strconv.ParseFloat(getEnvWithDefault("TTS_DEFAULT_VOLUME", "1.0"), 32)
	maxQueueSize, _ := strconv.Atoi(getEnvWithDefault("TTS_MAX_QUEUE_SIZE", "10"))
	maxMessageLength, _ := strconv.Atoi(getEnvWithDefault("TTS_MAX_MESSAGE_LENGTH", "500"))

	return TTSConfig{
		GoogleCloudCredentialsPath: os.Getenv("GOOGLE_CLOUD_CREDENTIALS_PATH"),
		DefaultVoice:               getEnvWithDefault("TTS_DEFAULT_VOICE", "en-US-Standard-A"),
		DefaultSpeed:               float32(defaultSpeed),
		DefaultVolume:              float32(defaultVolume),
		MaxQueueSize:               maxQueueSize,
		MaxMessageLength:           maxMessageLength,
	}
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

	// Validate TTS configuration
	if err := c.validateTTSConfig(); err != nil {
		return err
	}

	return nil
}

// validateTTSConfig validates TTS-specific configuration
func (c *Config) validateTTSConfig() error {
	if c.TTS.DefaultSpeed < 0.25 || c.TTS.DefaultSpeed > 4.0 {
		return errors.New("TTS_DEFAULT_SPEED must be between 0.25 and 4.0")
	}

	if c.TTS.DefaultVolume < 0.0 || c.TTS.DefaultVolume > 2.0 {
		return errors.New("TTS_DEFAULT_VOLUME must be between 0.0 and 2.0")
	}

	if c.TTS.MaxQueueSize < 1 || c.TTS.MaxQueueSize > 100 {
		return errors.New("TTS_MAX_QUEUE_SIZE must be between 1 and 100")
	}

	if c.TTS.MaxMessageLength < 1 || c.TTS.MaxMessageLength > 2000 {
		return errors.New("TTS_MAX_MESSAGE_LENGTH must be between 1 and 2000")
	}

	return nil
}

// getEnvWithDefault returns the value of an environment variable or a default value if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
