package tts

import (
	"darrot/internal/config"
	"errors"
	"fmt"
	"sync"
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

// configService implements the ConfigService interface
type configService struct {
	storage      *StorageService
	defaultTTS   config.TTSConfig
	guildConfigs map[string]*GuildTTSConfig
	mu           sync.RWMutex
}

// NewConfigService creates a new config service
func NewConfigService(storage *StorageService, defaultTTS config.TTSConfig) ConfigService {
	return &configService{
		storage:      storage,
		defaultTTS:   defaultTTS,
		guildConfigs: make(map[string]*GuildTTSConfig),
	}
}

// GetGuildConfig retrieves the TTS configuration for a guild
func (cs *configService) GetGuildConfig(guildID string) (*GuildTTSConfig, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Check cache first
	if config, exists := cs.guildConfigs[guildID]; exists {
		return config, nil
	}

	// Try to load from storage
	config, err := cs.storage.LoadGuildConfig(guildID)
	if err != nil {
		// If not found, return default config
		defaultConfig := cs.createDefaultGuildConfig(guildID)
		cs.guildConfigs[guildID] = &defaultConfig
		return &defaultConfig, nil
	}

	// Cache the loaded config
	cs.guildConfigs[guildID] = config
	return config, nil
}

// SetGuildConfig sets the TTS configuration for a guild
func (cs *configService) SetGuildConfig(guildID string, config *GuildTTSConfig) error {
	if err := cs.ValidateConfig(config); err != nil {
		return err
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Save to storage
	if err := cs.storage.SaveGuildConfig(*config); err != nil {
		return err
	}

	// Update cache
	cs.guildConfigs[guildID] = config
	return nil
}

// SetRequiredRoles sets the required roles for bot invitations
func (cs *configService) SetRequiredRoles(guildID string, roleIDs []string) error {
	config, err := cs.GetGuildConfig(guildID)
	if err != nil {
		return err
	}

	config.RequiredRoles = roleIDs
	return cs.SetGuildConfig(guildID, config)
}

// GetRequiredRoles gets the required roles for bot invitations
func (cs *configService) GetRequiredRoles(guildID string) ([]string, error) {
	config, err := cs.GetGuildConfig(guildID)
	if err != nil {
		return nil, err
	}

	return config.RequiredRoles, nil
}

// SetTTSSettings sets the TTS voice settings for a guild
func (cs *configService) SetTTSSettings(guildID string, settings TTSConfig) error {
	if err := ValidateConfig(settings); err != nil {
		return err
	}

	config, err := cs.GetGuildConfig(guildID)
	if err != nil {
		return err
	}

	config.TTSSettings = settings
	return cs.SetGuildConfig(guildID, config)
}

// GetTTSSettings gets the TTS voice settings for a guild
func (cs *configService) GetTTSSettings(guildID string) (*TTSConfig, error) {
	config, err := cs.GetGuildConfig(guildID)
	if err != nil {
		return nil, err
	}

	return &config.TTSSettings, nil
}

// SetMaxQueueSize sets the maximum queue size for a guild
func (cs *configService) SetMaxQueueSize(guildID string, size int) error {
	if size < 1 || size > 100 {
		return fmt.Errorf("queue size must be between 1 and 100")
	}

	config, err := cs.GetGuildConfig(guildID)
	if err != nil {
		return err
	}

	config.MaxQueueSize = size
	return cs.SetGuildConfig(guildID, config)
}

// GetMaxQueueSize gets the maximum queue size for a guild
func (cs *configService) GetMaxQueueSize(guildID string) (int, error) {
	config, err := cs.GetGuildConfig(guildID)
	if err != nil {
		return 0, err
	}

	return config.MaxQueueSize, nil
}

// ValidateConfig validates a guild TTS configuration
func (cs *configService) ValidateConfig(config *GuildTTSConfig) error {
	if config.GuildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	if err := ValidateConfig(config.TTSSettings); err != nil {
		return fmt.Errorf("invalid TTS settings: %w", err)
	}

	if config.MaxQueueSize < 1 || config.MaxQueueSize > 100 {
		return fmt.Errorf("max queue size must be between 1 and 100")
	}

	return nil
}

// createDefaultGuildConfig creates a default configuration for a guild
func (cs *configService) createDefaultGuildConfig(guildID string) GuildTTSConfig {
	return GuildTTSConfig{
		GuildID:       guildID,
		RequiredRoles: []string{}, // Empty means any user can invite
		TTSSettings: TTSConfig{
			Voice:  cs.defaultTTS.DefaultVoice,
			Speed:  cs.defaultTTS.DefaultSpeed,
			Volume: cs.defaultTTS.DefaultVolume,
			Format: AudioFormatOpus,
		},
		MaxQueueSize: cs.defaultTTS.MaxQueueSize,
	}
}
