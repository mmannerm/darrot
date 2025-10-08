package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantErr     bool
		expectedCfg *Config
	}{
		{
			name: "valid_configuration_with_defaults",
			envVars: map[string]string{
				"DISCORD_TOKEN": "test-token-123",
			},
			wantErr: false,
			expectedCfg: &Config{
				DiscordToken: "test-token-123",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					GoogleCloudCredentialsPath: "",
					DefaultVoice:               "en-US-Standard-A",
					DefaultSpeed:               1.0,
					DefaultVolume:              1.0,
					MaxQueueSize:               10,
					MaxMessageLength:           500,
				},
			},
		},
		{
			name: "valid_configuration_with_custom_log_level",
			envVars: map[string]string{
				"DISCORD_TOKEN": "test-token-456",
				"LOG_LEVEL":     "debug",
			},
			wantErr: false,
			expectedCfg: &Config{
				DiscordToken: "test-token-456",
				LogLevel:     "DEBUG",
				TTS: TTSConfig{
					GoogleCloudCredentialsPath: "",
					DefaultVoice:               "en-US-Standard-A",
					DefaultSpeed:               1.0,
					DefaultVolume:              1.0,
					MaxQueueSize:               10,
					MaxMessageLength:           500,
				},
			},
		},
		{
			name: "missing_discord_token",
			envVars: map[string]string{
				"LOG_LEVEL": "INFO",
			},
			wantErr:     true,
			expectedCfg: nil,
		},
		{
			name: "empty_discord_token",
			envVars: map[string]string{
				"DISCORD_TOKEN": "",
				"LOG_LEVEL":     "INFO",
			},
			wantErr:     true,
			expectedCfg: nil,
		},
		{
			name: "whitespace_only_discord_token",
			envVars: map[string]string{
				"DISCORD_TOKEN": "   ",
				"LOG_LEVEL":     "INFO",
			},
			wantErr:     true,
			expectedCfg: nil,
		},
		{
			name: "invalid_log_level",
			envVars: map[string]string{
				"DISCORD_TOKEN": "test-token-789",
				"LOG_LEVEL":     "INVALID",
			},
			wantErr:     true,
			expectedCfg: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if config.DiscordToken != tt.expectedCfg.DiscordToken {
				t.Errorf("Load() DiscordToken = %v, want %v", config.DiscordToken, tt.expectedCfg.DiscordToken)
			}

			if config.LogLevel != tt.expectedCfg.LogLevel {
				t.Errorf("Load() LogLevel = %v, want %v", config.LogLevel, tt.expectedCfg.LogLevel)
			}

			// Check TTS configuration
			if config.TTS.DefaultVoice != tt.expectedCfg.TTS.DefaultVoice {
				t.Errorf("Load() TTS.DefaultVoice = %v, want %v", config.TTS.DefaultVoice, tt.expectedCfg.TTS.DefaultVoice)
			}
			if config.TTS.DefaultSpeed != tt.expectedCfg.TTS.DefaultSpeed {
				t.Errorf("Load() TTS.DefaultSpeed = %v, want %v", config.TTS.DefaultSpeed, tt.expectedCfg.TTS.DefaultSpeed)
			}
			if config.TTS.DefaultVolume != tt.expectedCfg.TTS.DefaultVolume {
				t.Errorf("Load() TTS.DefaultVolume = %v, want %v", config.TTS.DefaultVolume, tt.expectedCfg.TTS.DefaultVolume)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid_config",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultVoice:     "en-US-Standard-A",
					DefaultSpeed:     1.0,
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: false,
		},
		{
			name: "empty_discord_token",
			config: &Config{
				DiscordToken: "",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultSpeed:     1.0,
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
		{
			name: "whitespace_discord_token",
			config: &Config{
				DiscordToken: "   ",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultSpeed:     1.0,
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_log_level",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "INVALID",
				TTS: TTSConfig{
					DefaultSpeed:     1.0,
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
		{
			name: "lowercase_log_level_gets_normalized",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "warn",
				TTS: TTSConfig{
					DefaultSpeed:     1.0,
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_tts_speed_too_low",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultSpeed:     0.1, // Too low
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_tts_speed_too_high",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultSpeed:     5.0, // Too high
					DefaultVolume:    1.0,
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_tts_volume_too_high",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultSpeed:     1.0,
					DefaultVolume:    3.0, // Too high
					MaxQueueSize:     10,
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid_queue_size_too_high",
			config: &Config{
				DiscordToken: "valid-token",
				LogLevel:     "INFO",
				TTS: TTSConfig{
					DefaultSpeed:     1.0,
					DefaultVolume:    1.0,
					MaxQueueSize:     200, // Too high
					MaxMessageLength: 500,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}

			// Check that lowercase log levels get normalized to uppercase
			if tt.name == "lowercase_log_level_gets_normalized" {
				if tt.config.LogLevel != "WARN" {
					t.Errorf("Validate() LogLevel normalization failed, got %v, want WARN", tt.config.LogLevel)
				}
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "env_var_exists",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			setEnv:       true,
			expected:     "custom",
		},
		{
			name:         "env_var_not_set",
			key:          "MISSING_VAR",
			defaultValue: "default",
			envValue:     "",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "env_var_empty_string",
			key:          "EMPTY_VAR",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the environment variable
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnvWithDefault(tt.key, tt.defaultValue)

			if result != tt.expected {
				t.Errorf("getEnvWithDefault() = %v, want %v", result, tt.expected)
			}

			// Clean up
			os.Unsetenv(tt.key)
		})
	}
}
