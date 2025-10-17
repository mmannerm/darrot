package config

import (
	"os"
	"testing"
)

func TestDRTEnvironmentVariableBinding(t *testing.T) {
	// Test that DRT-prefixed environment variables are properly bound

	// Set test environment variables
	os.Setenv("DRT_DISCORD_TOKEN", "test-token")
	os.Setenv("DRT_LOG_LEVEL", "DEBUG")
	os.Setenv("DRT_TTS_DEFAULT_SPEED", "1.5")
	defer func() {
		os.Unsetenv("DRT_DISCORD_TOKEN")
		os.Unsetenv("DRT_LOG_LEVEL")
		os.Unsetenv("DRT_TTS_DEFAULT_SPEED")
	}()

	// Create config manager and test Viper directly
	cm := NewConfigManager()

	// Set defaults first (this is important for AutomaticEnv to work properly)
	cm.SetDefaults()

	// Debug: Check if Viper can read the environment variables
	viper := cm.GetViper()
	t.Logf("Viper discord_token: %s (IsSet: %t)", viper.GetString("discord_token"), viper.IsSet("discord_token"))
	t.Logf("Viper log_level: %s (IsSet: %t)", viper.GetString("log_level"), viper.IsSet("log_level"))
	t.Logf("Viper tts.default_speed: %f (IsSet: %t)", viper.GetFloat64("tts.default_speed"), viper.IsSet("tts.default_speed"))

	// Try to unmarshal directly
	var testConfig Config
	if err := viper.Unmarshal(&testConfig); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	t.Logf("Unmarshaled discord_token: %s", testConfig.DiscordToken)
	t.Logf("Unmarshaled log_level: %s", testConfig.LogLevel)
	t.Logf("Unmarshaled tts.default_speed: %f", testConfig.TTS.DefaultSpeed)

	config, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables are properly bound
	if config.DiscordToken != "test-token" {
		t.Errorf("Expected discord_token to be 'test-token', got '%s'", config.DiscordToken)
	}

	if config.LogLevel != "DEBUG" {
		t.Errorf("Expected log_level to be 'DEBUG', got '%s'", config.LogLevel)
	}

	if config.TTS.DefaultSpeed != 1.5 {
		t.Errorf("Expected tts.default_speed to be 1.5, got %f", config.TTS.DefaultSpeed)
	}
}

func TestConfigKeyToDRTEnvMapping(t *testing.T) {
	cm := NewConfigManager()

	testCases := []struct {
		configKey string
		expected  string
	}{
		{"discord_token", "DRT_DISCORD_TOKEN"},
		{"log_level", "DRT_LOG_LEVEL"},
		{"tts.default_speed", "DRT_TTS_DEFAULT_SPEED"},
		{"tts.google_cloud_credentials_path", "DRT_TTS_GOOGLE_CLOUD_CREDENTIALS_PATH"},
	}

	for _, tc := range testCases {
		result := cm.getDRTEnvKey(tc.configKey)
		if result != tc.expected {
			t.Errorf("getDRTEnvKey(%s): expected %s, got %s", tc.configKey, tc.expected, result)
		}
	}
}

func TestDRTEnvToConfigKeyMapping(t *testing.T) {
	cm := NewConfigManager()

	testCases := []struct {
		envVar   string
		expected string
	}{
		{"DRT_DISCORD_TOKEN", "discord_token"},
		{"DRT_LOG_LEVEL", "log_level"},
		{"DRT_TTS_DEFAULT_SPEED", "tts.default_speed"},
		{"DRT_TTS_GOOGLE_CLOUD_CREDENTIALS_PATH", "tts.google_cloud_credentials_path"},
	}

	for _, tc := range testCases {
		result := cm.getConfigKeyForDRTEnv(tc.envVar)
		if result != tc.expected {
			t.Errorf("getConfigKeyForDRTEnv(%s): expected %s, got %s", tc.envVar, tc.expected, result)
		}
	}
}

func TestConfigurationPrecedence(t *testing.T) {
	// Test that CLI flags > environment variables > config file > defaults precedence works

	// Set required environment variables
	os.Setenv("DRT_DISCORD_TOKEN", "test-token")
	os.Setenv("DRT_LOG_LEVEL", "WARN")
	defer func() {
		os.Unsetenv("DRT_DISCORD_TOKEN")
		os.Unsetenv("DRT_LOG_LEVEL")
	}()

	cm := NewConfigManager()

	// Load config (should get WARN from env var)
	config, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.LogLevel != "WARN" {
		t.Errorf("Expected log_level to be 'WARN' from env var, got '%s'", config.LogLevel)
	}

	// Override with programmatic setting (simulating CLI flag)
	cm.SetConfigValue("log_level", "ERROR")

	// Reload config
	config, err = cm.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if config.LogLevel != "ERROR" {
		t.Errorf("Expected log_level to be 'ERROR' from programmatic override, got '%s'", config.LogLevel)
	}
}
