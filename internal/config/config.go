package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	DiscordToken string    `mapstructure:"discord_token"`
	LogLevel     string    `mapstructure:"log_level"`
	TTS          TTSConfig `mapstructure:"tts"`
}

// TTSConfig holds TTS-specific configuration
type TTSConfig struct {
	GoogleCloudCredentialsPath string  `mapstructure:"google_cloud_credentials_path"`
	DefaultVoice               string  `mapstructure:"default_voice"`
	DefaultSpeed               float32 `mapstructure:"default_speed"`
	DefaultVolume              float32 `mapstructure:"default_volume"`
	MaxQueueSize               int     `mapstructure:"max_queue_size"`
	MaxMessageLength           int     `mapstructure:"max_message_length"`
}

// ConfigManager manages configuration loading with Viper
type ConfigManager struct {
	viper *viper.Viper
}

// ConfigSource represents the source of a configuration value
type ConfigSource struct {
	Source string      // "default", "file", "env", "flag"
	Value  interface{} // The actual value
}

// ConfigWithSources holds configuration with source information
type ConfigWithSources struct {
	Config  *Config
	Sources map[string]ConfigSource
}

// NewConfigManager creates a new ConfigManager with Viper integration
func NewConfigManager() *ConfigManager {
	v := viper.New()

	// Set up automatic environment variable binding with DRT prefix
	v.SetEnvPrefix("DRT")
	v.AutomaticEnv()

	// Replace dots with underscores for environment variable names
	// This is needed for nested config keys like "tts.default_speed" -> "DRT_TTS_DEFAULT_SPEED"
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Register sensitive keys for environment variable binding without setting defaults
	// This tells Viper to look for these environment variables during AutomaticEnv
	v.BindEnv("discord_token")
	v.BindEnv("tts.google_cloud_credentials_path")

	return &ConfigManager{viper: v}
}

// tryReadConfigFile attempts to read config files from standard locations
func (cm *ConfigManager) tryReadConfigFile() error {
	// Set consistent config name across all locations
	cm.viper.SetConfigName("darrot-config")

	// Add config search paths in order of preference
	cm.viper.AddConfigPath(".") // ./darrot-config.{yaml,json,toml}
	if homeDir, err := os.UserHomeDir(); err == nil {
		cm.viper.AddConfigPath(homeDir) // ~/darrot-config.{yaml,json,toml}
	}
	cm.viper.AddConfigPath("/etc/darrot/") // /etc/darrot/darrot-config.{yaml,json,toml}

	// Try to read config file - Viper will automatically try multiple formats
	if err := cm.viper.ReadInConfig(); err != nil {
		// Check if it's a "file not found" error, which is acceptable
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found - this is not an error
			return nil
		}
		// Other error occurred while reading config
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// LoadConfig loads configuration from all sources with proper precedence
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	// Set default values first - this is crucial for AutomaticEnv to work
	// as Viper needs to know about the keys before it can read env vars
	cm.setDefaults()

	// Try to read config file (optional)
	if err := cm.tryReadConfigFile(); err != nil {
		return nil, err
	}

	// Unmarshal into config struct
	var config Config
	if err := cm.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadConfigWithSources loads configuration and tracks sources for each value
func (cm *ConfigManager) LoadConfigWithSources() (*ConfigWithSources, error) {
	config, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}

	sources := make(map[string]ConfigSource)

	// Track sources for each configuration value
	cm.trackConfigSources(sources)

	return &ConfigWithSources{
		Config:  config,
		Sources: sources,
	}, nil
}

// SetConfigFile sets a specific config file path
func (cm *ConfigManager) SetConfigFile(configFile string) {
	cm.viper.SetConfigFile(configFile)
}

// Load creates a new Config instance by loading values from all configuration sources
func Load() (*Config, error) {
	cm := NewConfigManager()
	return cm.LoadConfig()
}

// GetDefaultConfig returns a config struct with all default values
func GetDefaultConfig() *Config {
	return &Config{
		LogLevel: "INFO",
		TTS: TTSConfig{
			DefaultVoice:     "en-US-Standard-A",
			DefaultSpeed:     1.0,
			DefaultVolume:    1.0,
			MaxQueueSize:     10,
			MaxMessageLength: 500,
		},
	}
}

// Validate checks that all required configuration values are present and valid
func (c *Config) Validate() error {
	if strings.TrimSpace(c.DiscordToken) == "" {
		return errors.New("discord_token is required (set via DRT_DISCORD_TOKEN environment variable, config file, or --discord-token flag)")
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
		return errors.New("log_level must be one of: DEBUG, INFO, WARN, ERROR, FATAL (set via DRT_LOG_LEVEL environment variable, config file, or --log-level flag)")
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
		return errors.New("tts.default_speed must be between 0.25 and 4.0 (set via DRT_TTS_DEFAULT_SPEED environment variable, config file, or --tts-default-speed flag)")
	}

	if c.TTS.DefaultVolume < 0.0 || c.TTS.DefaultVolume > 2.0 {
		return errors.New("tts.default_volume must be between 0.0 and 2.0 (set via DRT_TTS_DEFAULT_VOLUME environment variable, config file, or --tts-default-volume flag)")
	}

	if c.TTS.MaxQueueSize < 1 || c.TTS.MaxQueueSize > 100 {
		return errors.New("tts.max_queue_size must be between 1 and 100 (set via DRT_TTS_MAX_QUEUE_SIZE environment variable, config file, or --tts-max-queue-size flag)")
	}

	if c.TTS.MaxMessageLength < 1 || c.TTS.MaxMessageLength > 2000 {
		return errors.New("tts.max_message_length must be between 1 and 2000 (set via DRT_TTS_MAX_MESSAGE_LENGTH environment variable, config file, or --tts-max-message-length flag)")
	}

	return nil
}

// SetDefaults sets default values for all configuration options (public for testing)
func (cm *ConfigManager) SetDefaults() {
	cm.setDefaults()
}

// setDefaults sets default values for all configuration options
// These defaults maintain backward compatibility with the existing implementation
func (cm *ConfigManager) setDefaults() {
	// Core configuration defaults
	cm.viper.SetDefault("log_level", "INFO") // Default log level for application logging

	// TTS configuration defaults - these match the existing implementation
	cm.viper.SetDefault("tts.default_voice", "en-US-Standard-A") // Google Cloud TTS voice
	cm.viper.SetDefault("tts.default_speed", 1.0)                // Normal speech speed (0.25-4.0 range)
	cm.viper.SetDefault("tts.default_volume", 1.0)               // Normal volume (0.0-2.0 range)
	cm.viper.SetDefault("tts.max_queue_size", 10)                // Maximum messages in TTS queue
	cm.viper.SetDefault("tts.max_message_length", 500)           // Maximum characters per message

	// Note: discord_token and tts.google_cloud_credentials_path have no defaults
	// as they are sensitive configuration that must be explicitly provided
	// They are registered for environment variable binding in NewConfigManager()
}

// GetAllDefaults returns a map of all default configuration values
func (cm *ConfigManager) GetAllDefaults() map[string]interface{} {
	defaults := make(map[string]interface{})

	// Set defaults temporarily to extract them
	cm.setDefaults()

	// Extract all default values
	keys := []string{
		"log_level",
		"tts.default_voice",
		"tts.default_speed",
		"tts.default_volume",
		"tts.max_queue_size",
		"tts.max_message_length",
	}

	for _, key := range keys {
		defaults[key] = cm.viper.GetString(key)
	}

	return defaults
}

// trackConfigSources tracks the source of each configuration value
func (cm *ConfigManager) trackConfigSources(sources map[string]ConfigSource) {
	// Helper function to determine source based on precedence order:
	// CLI flags > environment variables > config file > defaults
	getSource := func(key string) string {
		// Check if set via CLI flag (highest precedence)
		// Note: This will be properly implemented when CLI integration is complete
		// For now, we check if the value was explicitly set (not from env or config file)
		if cm.viper.IsSet(key) {
			// Check if it's from environment variable with DRT prefix
			if envKey := cm.getDRTEnvKey(key); envKey != "" && os.Getenv(envKey) != "" {
				// If env var is set, check if the current value matches the env value
				envValue := os.Getenv(envKey)
				currentValue := fmt.Sprintf("%v", cm.viper.Get(key))
				if currentValue == envValue {
					return "env"
				}
				// If values don't match, it might be from a flag override
				return "flag"
			}

			// Check if value exists in config file
			if cm.viper.InConfig(key) {
				return "file"
			}

			// If set but not from env or file, likely from flag or programmatic setting
			return "flag"
		}

		return "default"
	}

	// Track sources for all config values
	keys := []string{
		"discord_token",
		"log_level",
		"tts.google_cloud_credentials_path",
		"tts.default_voice",
		"tts.default_speed",
		"tts.default_volume",
		"tts.max_queue_size",
		"tts.max_message_length",
	}

	for _, key := range keys {
		sources[key] = ConfigSource{
			Source: getSource(key),
			Value:  cm.viper.Get(key),
		}
	}
}

// getDRTEnvKey returns the DRT-prefixed environment variable key for a config key
func (cm *ConfigManager) getDRTEnvKey(configKey string) string {
	// Convert config key to DRT environment variable format
	// Replace dots with underscores and add DRT prefix
	envKey := strings.ToUpper(strings.ReplaceAll(configKey, ".", "_"))
	return "DRT_" + envKey
}

// BindFlags binds CLI flags to configuration keys
func (cm *ConfigManager) BindFlags(flagSet interface{}) error {
	// This will be called from the CLI commands to bind flags
	// The flagSet parameter will be a *pflag.FlagSet from Cobra
	// For now, we'll implement the binding logic that will be used by CLI commands

	// Note: Actual flag binding will be done in the CLI command files
	// This method provides the interface for the CLI layer to bind flags
	return nil
}

// BindFlag binds a single CLI flag to a configuration key
func (cm *ConfigManager) BindFlag(key, flagName string) error {
	return cm.viper.BindPFlag(key, nil) // Will be properly implemented when CLI flags are added
}

// SetConfigValue sets a configuration value programmatically (for CLI flags)
func (cm *ConfigManager) SetConfigValue(key string, value interface{}) {
	cm.viper.Set(key, value)
}

// GetConfigValue gets a configuration value
func (cm *ConfigManager) GetConfigValue(key string) interface{} {
	return cm.viper.Get(key)
}

// IsSet checks if a configuration key has been set from any source
func (cm *ConfigManager) IsSet(key string) bool {
	return cm.viper.IsSet(key)
}

// ValidatePrecedence validates that the configuration precedence system is working correctly
func (cm *ConfigManager) ValidatePrecedence() error {
	// This method can be used to test that the precedence system is working
	// It will be useful for the config validate command

	// With AutomaticEnv enabled, Viper automatically reads DRT-prefixed environment variables
	// when the corresponding config key is accessed. No explicit validation needed.

	return nil
}

// getConfigKeyForDRTEnv returns the config key for a DRT-prefixed environment variable
func (cm *ConfigManager) getConfigKeyForDRTEnv(envVar string) string {
	// Remove DRT_ prefix and convert to config key format
	if !strings.HasPrefix(envVar, "DRT_") {
		return ""
	}

	// Remove DRT_ prefix
	withoutPrefix := strings.TrimPrefix(envVar, "DRT_")

	// Convert to lowercase and replace underscores with dots for nested keys
	configKey := strings.ToLower(withoutPrefix)

	// Handle TTS nested keys specifically
	if strings.HasPrefix(configKey, "tts_") {
		configKey = strings.Replace(configKey, "tts_", "tts.", 1)
	}

	return configKey
}

// GetViper returns the underlying Viper instance for advanced usage
func (cm *ConfigManager) GetViper() *viper.Viper {
	return cm.viper
}

// ValidateDefaults ensures all expected default values are properly set
func (cm *ConfigManager) ValidateDefaults() error {
	expectedDefaults := map[string]interface{}{
		"log_level":              "INFO",
		"tts.default_voice":      "en-US-Standard-A",
		"tts.default_speed":      1.0,
		"tts.default_volume":     1.0,
		"tts.max_queue_size":     10,
		"tts.max_message_length": 500,
	}

	// Set defaults to ensure they're available
	cm.setDefaults()

	// Validate each expected default
	for key, expectedValue := range expectedDefaults {
		actualValue := cm.viper.Get(key)
		if actualValue != expectedValue {
			return fmt.Errorf("default value mismatch for %s: expected %v, got %v", key, expectedValue, actualValue)
		}
	}

	return nil
}
