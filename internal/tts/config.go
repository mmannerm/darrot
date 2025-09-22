package tts

import (
	"errors"
	"fmt"
)

// DefaultTTSConfig returns the default TTS configuration
func DefaultTTSConfig() TTSConfig {
	return TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatOpus,
	}
}

// ValidateConfig validates a TTS configuration
func ValidateConfig(config TTSConfig) error {
	if config.Speed < 0.25 || config.Speed > 4.0 {
		return errors.New("speed must be between 0.25 and 4.0")
	}

	if config.Volume < 0.0 || config.Volume > 2.0 {
		return errors.New("volume must be between 0.0 and 2.0")
	}

	validFormats := map[AudioFormat]bool{
		AudioFormatOpus: true,
		AudioFormatDCA:  true,
		AudioFormatPCM:  true,
	}

	if !validFormats[config.Format] {
		return fmt.Errorf("invalid audio format: %s", config.Format)
	}

	return nil
}

// DefaultGuildTTSConfig returns the default guild TTS configuration
func DefaultGuildTTSConfig(guildID string) GuildTTSConfig {
	return GuildTTSConfig{
		GuildID:       guildID,
		RequiredRoles: []string{}, // Empty means any user can invite
		TTSSettings:   DefaultTTSConfig(),
		MaxQueueSize:  10,
	}
}

// ValidateGuildConfig validates a guild TTS configuration
func ValidateGuildConfig(config GuildTTSConfig) error {
	if config.GuildID == "" {
		return errors.New("guild ID is required")
	}

	if config.MaxQueueSize < 1 || config.MaxQueueSize > 100 {
		return errors.New("max queue size must be between 1 and 100")
	}

	return ValidateConfig(config.TTSSettings)
}

// DefaultUserPreferences returns the default user TTS preferences
func DefaultUserPreferences(userID, guildID string) UserTTSPreferences {
	return UserTTSPreferences{
		UserID:  userID,
		GuildID: guildID,
		OptedIn: false, // Users must explicitly opt-in
		Settings: UserTTSSettings{
			PreferredVoice: "en-US-Standard-A",
			SpeedModifier:  1.0,
		},
	}
}

// ValidateUserPreferences validates user TTS preferences
func ValidateUserPreferences(prefs UserTTSPreferences) error {
	if prefs.UserID == "" {
		return errors.New("user ID is required")
	}

	if prefs.GuildID == "" {
		return errors.New("guild ID is required")
	}

	if prefs.Settings.SpeedModifier < 0.25 || prefs.Settings.SpeedModifier > 4.0 {
		return errors.New("speed modifier must be between 0.25 and 4.0")
	}

	if prefs.Settings.PreferredVoice == "" {
		return errors.New("preferred voice is required")
	}

	return nil
}

// ValidateChannelPairing validates a channel pairing
func ValidateChannelPairing(pairing ChannelPairingStorage) error {
	if pairing.GuildID == "" {
		return errors.New("guild ID is required")
	}

	if pairing.VoiceChannelID == "" {
		return errors.New("voice channel ID is required")
	}

	if pairing.TextChannelID == "" {
		return errors.New("text channel ID is required")
	}

	// CreatedBy can be empty for system-created pairings

	if pairing.CreatedAt.IsZero() {
		return errors.New("created at timestamp is required")
	}

	return nil
}
