package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		MockServer: struct {
			Host           string        `yaml:"host"`
			Port           int           `yaml:"port"`
			GatewayPort    int           `yaml:"gateway_port"`
			StartupTimeout time.Duration `yaml:"startup_timeout"`
			ResponseDelay  time.Duration `yaml:"response_delay"`
		}{
			Host:           "localhost",
			Port:           8080,
			GatewayPort:    8081,
			StartupTimeout: 30 * time.Second,
			ResponseDelay:  100 * time.Millisecond,
		},
		Bot: struct {
			BinaryPath     string        `yaml:"binary_path"`
			ConfigPath     string        `yaml:"config_path"`
			StartupTimeout time.Duration `yaml:"startup_timeout"`
			LogLevel       string        `yaml:"log_level"`
			WorkingDir     string        `yaml:"working_dir"`
		}{
			BinaryPath:     "./darrot",
			ConfigPath:     "",
			StartupTimeout: 30 * time.Second,
			LogLevel:       "DEBUG",
			WorkingDir:     ".",
		},
		Audio: struct {
			CaptureTimeout    time.Duration `yaml:"capture_timeout"`
			ExpectedFormat    string        `yaml:"expected_format"`
			QualityThresholds struct {
				MinBitrate int           `yaml:"min_bitrate"`
				MaxLatency time.Duration `yaml:"max_latency"`
			} `yaml:"quality_thresholds"`
		}{
			CaptureTimeout: 30 * time.Second,
			ExpectedFormat: "opus",
			QualityThresholds: struct {
				MinBitrate int           `yaml:"min_bitrate"`
				MaxLatency time.Duration `yaml:"max_latency"`
			}{
				MinBitrate: 32000, // 32 kbps minimum
				MaxLatency: 500 * time.Millisecond,
			},
		},
		Test: struct {
			DefaultTimeout time.Duration `yaml:"default_timeout"`
			RetryAttempts  int           `yaml:"retry_attempts"`
			RetryDelay     time.Duration `yaml:"retry_delay"`
			ParallelTests  int           `yaml:"parallel_tests"`
		}{
			DefaultTimeout: 60 * time.Second,
			RetryAttempts:  3,
			RetryDelay:     1 * time.Second,
			ParallelTests:  1,
		},
	}
}

// LoadTestConfig loads test configuration from a file
func LoadTestConfig(configPath string) (*TestConfig, error) {
	// Start with default config
	config := DefaultTestConfig()

	// If no config path provided, return defaults
	if configPath == "" {
		return config, nil
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // Return defaults if file doesn't exist
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return config, nil
}

// SaveTestConfig saves test configuration to a file
func SaveTestConfig(config *TestConfig, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateTestConfig validates the test configuration
func ValidateTestConfig(config *TestConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate mock server config
	if config.MockServer.Host == "" {
		return fmt.Errorf("mock server host cannot be empty")
	}
	if config.MockServer.Port <= 0 || config.MockServer.Port > 65535 {
		return fmt.Errorf("mock server port must be between 1 and 65535")
	}
	if config.MockServer.GatewayPort <= 0 || config.MockServer.GatewayPort > 65535 {
		return fmt.Errorf("mock server gateway port must be between 1 and 65535")
	}
	if config.MockServer.StartupTimeout <= 0 {
		return fmt.Errorf("mock server startup timeout must be positive")
	}

	// Validate bot config
	if config.Bot.BinaryPath == "" {
		return fmt.Errorf("bot binary path cannot be empty")
	}
	if config.Bot.StartupTimeout <= 0 {
		return fmt.Errorf("bot startup timeout must be positive")
	}
	if config.Bot.LogLevel == "" {
		config.Bot.LogLevel = "INFO" // Set default
	}
	if config.Bot.WorkingDir == "" {
		config.Bot.WorkingDir = "." // Set default
	}

	// Validate audio config
	if config.Audio.ExpectedFormat == "" {
		config.Audio.ExpectedFormat = "opus" // Set default
	}
	if config.Audio.CaptureTimeout <= 0 {
		return fmt.Errorf("audio capture timeout must be positive")
	}
	if config.Audio.QualityThresholds.MinBitrate <= 0 {
		return fmt.Errorf("minimum bitrate must be positive")
	}
	if config.Audio.QualityThresholds.MaxLatency <= 0 {
		return fmt.Errorf("maximum latency must be positive")
	}

	// Validate test config
	if config.Test.DefaultTimeout <= 0 {
		return fmt.Errorf("default test timeout must be positive")
	}
	if config.Test.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}
	if config.Test.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative")
	}
	if config.Test.ParallelTests <= 0 {
		config.Test.ParallelTests = 1 // Set default
	}

	return nil
}

// GetTestConfigPath returns the default test config path
func GetTestConfigPath() string {
	return filepath.Join("tests", "acceptance", "config.yaml")
}

// CreateDefaultConfigFile creates a default configuration file
func CreateDefaultConfigFile(configPath string) error {
	config := DefaultTestConfig()
	return SaveTestConfig(config, configPath)
}

// MergeConfigs merges two configurations, with override taking precedence
func MergeConfigs(base, override *TestConfig) *TestConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	// Create a copy of base config
	merged := *base

	// Override mock server settings
	if override.MockServer.Host != "" {
		merged.MockServer.Host = override.MockServer.Host
	}
	if override.MockServer.Port != 0 {
		merged.MockServer.Port = override.MockServer.Port
	}
	if override.MockServer.GatewayPort != 0 {
		merged.MockServer.GatewayPort = override.MockServer.GatewayPort
	}
	if override.MockServer.StartupTimeout != 0 {
		merged.MockServer.StartupTimeout = override.MockServer.StartupTimeout
	}
	if override.MockServer.ResponseDelay != 0 {
		merged.MockServer.ResponseDelay = override.MockServer.ResponseDelay
	}

	// Override bot settings
	if override.Bot.BinaryPath != "" {
		merged.Bot.BinaryPath = override.Bot.BinaryPath
	}
	if override.Bot.ConfigPath != "" {
		merged.Bot.ConfigPath = override.Bot.ConfigPath
	}
	if override.Bot.StartupTimeout != 0 {
		merged.Bot.StartupTimeout = override.Bot.StartupTimeout
	}
	if override.Bot.LogLevel != "" {
		merged.Bot.LogLevel = override.Bot.LogLevel
	}
	if override.Bot.WorkingDir != "" {
		merged.Bot.WorkingDir = override.Bot.WorkingDir
	}

	// Override audio settings
	if override.Audio.CaptureTimeout != 0 {
		merged.Audio.CaptureTimeout = override.Audio.CaptureTimeout
	}
	if override.Audio.ExpectedFormat != "" {
		merged.Audio.ExpectedFormat = override.Audio.ExpectedFormat
	}
	if override.Audio.QualityThresholds.MinBitrate != 0 {
		merged.Audio.QualityThresholds.MinBitrate = override.Audio.QualityThresholds.MinBitrate
	}
	if override.Audio.QualityThresholds.MaxLatency != 0 {
		merged.Audio.QualityThresholds.MaxLatency = override.Audio.QualityThresholds.MaxLatency
	}

	// Override test settings
	if override.Test.DefaultTimeout != 0 {
		merged.Test.DefaultTimeout = override.Test.DefaultTimeout
	}
	if override.Test.RetryAttempts != 0 {
		merged.Test.RetryAttempts = override.Test.RetryAttempts
	}
	if override.Test.RetryDelay != 0 {
		merged.Test.RetryDelay = override.Test.RetryDelay
	}
	if override.Test.ParallelTests != 0 {
		merged.Test.ParallelTests = override.Test.ParallelTests
	}

	return &merged
}
