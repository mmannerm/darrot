package tts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StorageService provides JSON-based storage for TTS configuration data
type StorageService struct {
	dataDir string
	mutex   sync.RWMutex
}

// NewStorageService creates a new storage service with the specified data directory
func NewStorageService(dataDir string) (*StorageService, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &StorageService{
		dataDir: dataDir,
	}, nil
}

// SaveGuildConfig saves guild TTS configuration to JSON file
func (s *StorageService) SaveGuildConfig(config GuildTTSConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := ValidateGuildConfig(config); err != nil {
		return fmt.Errorf("invalid guild config: %w", err)
	}

	config.UpdatedAt = time.Now()

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("guild_%s.json", config.GuildID))
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal guild config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write guild config file: %w", err)
	}

	return nil
}

// LoadGuildConfig loads guild TTS configuration from JSON file
func (s *StorageService) LoadGuildConfig(guildID string) (*GuildTTSConfig, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("guild_%s.json", guildID))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		defaultConfig := DefaultGuildTTSConfig(guildID)
		return &defaultConfig, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read guild config file: %w", err)
	}

	var config GuildTTSConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal guild config: %w", err)
	}

	return &config, nil
}

// SaveUserPreferences saves user TTS preferences to JSON file
func (s *StorageService) SaveUserPreferences(prefs UserTTSPreferences) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := ValidateUserPreferences(prefs); err != nil {
		return fmt.Errorf("invalid user preferences: %w", err)
	}

	prefs.UpdatedAt = time.Now()

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("user_%s_%s.json", prefs.UserID, prefs.GuildID))
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user preferences: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write user preferences file: %w", err)
	}

	return nil
}

// LoadUserPreferences loads user TTS preferences from JSON file
func (s *StorageService) LoadUserPreferences(userID, guildID string) (*UserTTSPreferences, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("user_%s_%s.json", userID, guildID))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return default preferences if file doesn't exist
		defaultPrefs := DefaultUserPreferences(userID, guildID)
		return &defaultPrefs, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user preferences file: %w", err)
	}

	var prefs UserTTSPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user preferences: %w", err)
	}

	return &prefs, nil
}

// SaveChannelPairing saves channel pairing to JSON file
func (s *StorageService) SaveChannelPairing(pairing ChannelPairingStorage) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := ValidateChannelPairing(pairing); err != nil {
		return fmt.Errorf("invalid channel pairing: %w", err)
	}

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("pairing_%s_%s.json", pairing.GuildID, pairing.VoiceChannelID))
	data, err := json.MarshalIndent(pairing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal channel pairing: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write channel pairing file: %w", err)
	}

	return nil
}

// LoadChannelPairing loads channel pairing from JSON file
func (s *StorageService) LoadChannelPairing(guildID, voiceChannelID string) (*ChannelPairingStorage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("pairing_%s_%s.json", guildID, voiceChannelID))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("channel pairing not found")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read channel pairing file: %w", err)
	}

	var pairing ChannelPairingStorage
	if err := json.Unmarshal(data, &pairing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel pairing: %w", err)
	}

	return &pairing, nil
}

// RemoveChannelPairing removes channel pairing file
func (s *StorageService) RemoveChannelPairing(guildID, voiceChannelID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filePath := filepath.Join(s.dataDir, fmt.Sprintf("pairing_%s_%s.json", guildID, voiceChannelID))

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove channel pairing file: %w", err)
	}

	return nil
}

// ListGuildPairings returns all active channel pairings for a guild
func (s *StorageService) ListGuildPairings(guildID string) ([]ChannelPairingStorage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	pattern := filepath.Join(s.dataDir, fmt.Sprintf("pairing_%s_*.json", guildID))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list pairing files: %w", err)
	}

	var pairings []ChannelPairingStorage
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var pairing ChannelPairingStorage
		if err := json.Unmarshal(data, &pairing); err != nil {
			continue // Skip files that can't be parsed
		}

		if pairing.IsActive {
			pairings = append(pairings, pairing)
		}
	}

	return pairings, nil
}

// ListOptedInUsers returns all users who have opted in for a guild
func (s *StorageService) ListOptedInUsers(guildID string) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	pattern := filepath.Join(s.dataDir, fmt.Sprintf("user_*_%s.json", guildID))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list user preference files: %w", err)
	}

	var optedInUsers []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var prefs UserTTSPreferences
		if err := json.Unmarshal(data, &prefs); err != nil {
			continue // Skip files that can't be parsed
		}

		if prefs.OptedIn {
			optedInUsers = append(optedInUsers, prefs.UserID)
		}
	}

	return optedInUsers, nil
}
