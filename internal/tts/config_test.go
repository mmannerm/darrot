package tts

import (
	"testing"
	"time"
)

func TestDefaultTTSConfig(t *testing.T) {
	config := DefaultTTSConfig()

	if config.Voice != "en-US-Standard-A" {
		t.Errorf("Expected default voice 'en-US-Standard-A', got '%s'", config.Voice)
	}

	if config.Speed != 1.0 {
		t.Errorf("Expected default speed 1.0, got %f", config.Speed)
	}

	if config.Volume != 1.0 {
		t.Errorf("Expected default volume 1.0, got %f", config.Volume)
	}

	if config.Format != AudioFormatOpus {
		t.Errorf("Expected default format %s, got %s", AudioFormatOpus, config.Format)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  TTSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: AudioFormatOpus,
			},
			wantErr: false,
		},
		{
			name: "speed too low",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  0.1,
				Volume: 1.0,
				Format: AudioFormatOpus,
			},
			wantErr: true,
			errMsg:  "speed must be between 0.25 and 4.0",
		},
		{
			name: "speed too high",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  5.0,
				Volume: 1.0,
				Format: AudioFormatOpus,
			},
			wantErr: true,
			errMsg:  "speed must be between 0.25 and 4.0",
		},
		{
			name: "volume too low",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: -0.1,
				Format: AudioFormatOpus,
			},
			wantErr: true,
			errMsg:  "volume must be between 0.0 and 2.0",
		},
		{
			name: "volume too high",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 2.1,
				Format: AudioFormatOpus,
			},
			wantErr: true,
			errMsg:  "volume must be between 0.0 and 2.0",
		},
		{
			name: "invalid format",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid audio format: invalid",
		},
		{
			name: "boundary values - minimum",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  0.25,
				Volume: 0.0,
				Format: AudioFormatDCA,
			},
			wantErr: false,
		},
		{
			name: "boundary values - maximum",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  4.0,
				Volume: 2.0,
				Format: AudioFormatPCM,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ValidateConfig() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestDefaultGuildTTSConfig(t *testing.T) {
	guildID := "123456789"
	config := DefaultGuildTTSConfig(guildID)

	if config.GuildID != guildID {
		t.Errorf("Expected GuildID %s, got %s", guildID, config.GuildID)
	}

	if len(config.RequiredRoles) != 0 {
		t.Errorf("Expected empty RequiredRoles, got %d roles", len(config.RequiredRoles))
	}

	if config.MaxQueueSize != 10 {
		t.Errorf("Expected MaxQueueSize 10, got %d", config.MaxQueueSize)
	}

	// Verify default TTS settings
	if config.TTSSettings.Voice != "en-US-Standard-A" {
		t.Errorf("Expected default voice 'en-US-Standard-A', got '%s'", config.TTSSettings.Voice)
	}

	if config.TTSSettings.Speed != 1.0 {
		t.Errorf("Expected default speed 1.0, got %f", config.TTSSettings.Speed)
	}
}

func TestValidateGuildConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  GuildTTSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: GuildTTSConfig{
				GuildID:       "123456789",
				RequiredRoles: []string{"role1", "role2"},
				TTSSettings:   DefaultTTSConfig(),
				MaxQueueSize:  15,
			},
			wantErr: false,
		},
		{
			name: "empty guild ID",
			config: GuildTTSConfig{
				GuildID:       "",
				RequiredRoles: []string{},
				TTSSettings:   DefaultTTSConfig(),
				MaxQueueSize:  10,
			},
			wantErr: true,
			errMsg:  "guild ID is required",
		},
		{
			name: "queue size too small",
			config: GuildTTSConfig{
				GuildID:       "123456789",
				RequiredRoles: []string{},
				TTSSettings:   DefaultTTSConfig(),
				MaxQueueSize:  0,
			},
			wantErr: true,
			errMsg:  "max queue size must be between 1 and 100",
		},
		{
			name: "queue size too large",
			config: GuildTTSConfig{
				GuildID:       "123456789",
				RequiredRoles: []string{},
				TTSSettings:   DefaultTTSConfig(),
				MaxQueueSize:  101,
			},
			wantErr: true,
			errMsg:  "max queue size must be between 1 and 100",
		},
		{
			name: "invalid TTS settings",
			config: GuildTTSConfig{
				GuildID:       "123456789",
				RequiredRoles: []string{},
				TTSSettings: TTSConfig{
					Voice:  "en-US-Standard-A",
					Speed:  5.0, // Invalid speed
					Volume: 1.0,
					Format: AudioFormatOpus,
				},
				MaxQueueSize: 10,
			},
			wantErr: true,
			errMsg:  "speed must be between 0.25 and 4.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGuildConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGuildConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ValidateGuildConfig() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestDefaultUserPreferences(t *testing.T) {
	userID := "user123"
	guildID := "guild456"
	prefs := DefaultUserPreferences(userID, guildID)

	if prefs.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, prefs.UserID)
	}

	if prefs.GuildID != guildID {
		t.Errorf("Expected GuildID %s, got %s", guildID, prefs.GuildID)
	}

	if prefs.OptedIn != false {
		t.Errorf("Expected OptedIn false by default, got %t", prefs.OptedIn)
	}

	if prefs.Settings.PreferredVoice != "en-US-Standard-A" {
		t.Errorf("Expected default PreferredVoice 'en-US-Standard-A', got '%s'", prefs.Settings.PreferredVoice)
	}

	if prefs.Settings.SpeedModifier != 1.0 {
		t.Errorf("Expected default SpeedModifier 1.0, got %f", prefs.Settings.SpeedModifier)
	}
}

func TestValidateUserPreferences(t *testing.T) {
	tests := []struct {
		name    string
		prefs   UserTTSPreferences
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid preferences",
			prefs: UserTTSPreferences{
				UserID:  "user123",
				GuildID: "guild456",
				OptedIn: true,
				Settings: UserTTSSettings{
					PreferredVoice: "en-US-Standard-B",
					SpeedModifier:  1.2,
				},
			},
			wantErr: false,
		},
		{
			name: "empty user ID",
			prefs: UserTTSPreferences{
				UserID:  "",
				GuildID: "guild456",
				OptedIn: true,
				Settings: UserTTSSettings{
					PreferredVoice: "en-US-Standard-B",
					SpeedModifier:  1.2,
				},
			},
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name: "empty guild ID",
			prefs: UserTTSPreferences{
				UserID:  "user123",
				GuildID: "",
				OptedIn: true,
				Settings: UserTTSSettings{
					PreferredVoice: "en-US-Standard-B",
					SpeedModifier:  1.2,
				},
			},
			wantErr: true,
			errMsg:  "guild ID is required",
		},
		{
			name: "speed modifier too low",
			prefs: UserTTSPreferences{
				UserID:  "user123",
				GuildID: "guild456",
				OptedIn: true,
				Settings: UserTTSSettings{
					PreferredVoice: "en-US-Standard-B",
					SpeedModifier:  0.1,
				},
			},
			wantErr: true,
			errMsg:  "speed modifier must be between 0.25 and 4.0",
		},
		{
			name: "speed modifier too high",
			prefs: UserTTSPreferences{
				UserID:  "user123",
				GuildID: "guild456",
				OptedIn: true,
				Settings: UserTTSSettings{
					PreferredVoice: "en-US-Standard-B",
					SpeedModifier:  5.0,
				},
			},
			wantErr: true,
			errMsg:  "speed modifier must be between 0.25 and 4.0",
		},
		{
			name: "empty preferred voice",
			prefs: UserTTSPreferences{
				UserID:  "user123",
				GuildID: "guild456",
				OptedIn: true,
				Settings: UserTTSSettings{
					PreferredVoice: "",
					SpeedModifier:  1.0,
				},
			},
			wantErr: true,
			errMsg:  "preferred voice is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserPreferences(tt.prefs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserPreferences() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ValidateUserPreferences() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidateChannelPairing(t *testing.T) {
	validTime := time.Now()

	tests := []struct {
		name    string
		pairing ChannelPairingStorage
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pairing",
			pairing: ChannelPairingStorage{
				GuildID:        "guild123",
				VoiceChannelID: "voice456",
				TextChannelID:  "text789",
				CreatedBy:      "user123",
				CreatedAt:      validTime,
				IsActive:       true,
			},
			wantErr: false,
		},
		{
			name: "empty guild ID",
			pairing: ChannelPairingStorage{
				GuildID:        "",
				VoiceChannelID: "voice456",
				TextChannelID:  "text789",
				CreatedBy:      "user123",
				CreatedAt:      validTime,
				IsActive:       true,
			},
			wantErr: true,
			errMsg:  "guild ID is required",
		},
		{
			name: "empty voice channel ID",
			pairing: ChannelPairingStorage{
				GuildID:        "guild123",
				VoiceChannelID: "",
				TextChannelID:  "text789",
				CreatedBy:      "user123",
				CreatedAt:      validTime,
				IsActive:       true,
			},
			wantErr: true,
			errMsg:  "voice channel ID is required",
		},
		{
			name: "empty text channel ID",
			pairing: ChannelPairingStorage{
				GuildID:        "guild123",
				VoiceChannelID: "voice456",
				TextChannelID:  "",
				CreatedBy:      "user123",
				CreatedAt:      validTime,
				IsActive:       true,
			},
			wantErr: true,
			errMsg:  "text channel ID is required",
		},
		{
			name: "empty created by",
			pairing: ChannelPairingStorage{
				GuildID:        "guild123",
				VoiceChannelID: "voice456",
				TextChannelID:  "text789",
				CreatedBy:      "",
				CreatedAt:      validTime,
				IsActive:       true,
			},
			wantErr: false, // Empty created by is now allowed
		},
		{
			name: "zero created at time",
			pairing: ChannelPairingStorage{
				GuildID:        "guild123",
				VoiceChannelID: "voice456",
				TextChannelID:  "text789",
				CreatedBy:      "user123",
				CreatedAt:      time.Time{},
				IsActive:       true,
			},
			wantErr: true,
			errMsg:  "created at timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChannelPairing(tt.pairing)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChannelPairing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ValidateChannelPairing() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
