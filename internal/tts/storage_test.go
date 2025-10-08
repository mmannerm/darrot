package tts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStorageService(t *testing.T) {
	tempDir := t.TempDir()

	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	if service.dataDir != tempDir {
		t.Errorf("Expected dataDir %s, got %s", tempDir, service.dataDir)
	}

	// Verify directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Data directory was not created")
	}
}

func TestStorageService_GuildConfig(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	guildID := "123456789"
	config := GuildTTSConfig{
		GuildID:       guildID,
		RequiredRoles: []string{"role1", "role2"},
		TTSSettings: TTSConfig{
			Voice:  "en-US-Standard-B",
			Speed:  1.5,
			Volume: 0.8,
			Format: AudioFormatOpus,
		},
		MaxQueueSize: 15,
		UpdatedAt:    time.Now(),
	}

	// Test saving guild config
	err = service.SaveGuildConfig(config)
	if err != nil {
		t.Fatalf("Failed to save guild config: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tempDir, "guild_123456789.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Guild config file was not created")
	}

	// Test loading guild config
	loadedConfig, err := service.LoadGuildConfig(guildID)
	if err != nil {
		t.Fatalf("Failed to load guild config: %v", err)
	}

	if loadedConfig.GuildID != config.GuildID {
		t.Errorf("Expected GuildID %s, got %s", config.GuildID, loadedConfig.GuildID)
	}

	if len(loadedConfig.RequiredRoles) != len(config.RequiredRoles) {
		t.Errorf("Expected %d required roles, got %d", len(config.RequiredRoles), len(loadedConfig.RequiredRoles))
	}

	if loadedConfig.TTSSettings.Voice != config.TTSSettings.Voice {
		t.Errorf("Expected voice %s, got %s", config.TTSSettings.Voice, loadedConfig.TTSSettings.Voice)
	}

	if loadedConfig.MaxQueueSize != config.MaxQueueSize {
		t.Errorf("Expected MaxQueueSize %d, got %d", config.MaxQueueSize, loadedConfig.MaxQueueSize)
	}
}

func TestStorageService_GuildConfig_Default(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	guildID := "nonexistent"

	// Test loading non-existent guild config returns default
	config, err := service.LoadGuildConfig(guildID)
	if err != nil {
		t.Fatalf("Failed to load default guild config: %v", err)
	}

	if config.GuildID != guildID {
		t.Errorf("Expected GuildID %s, got %s", guildID, config.GuildID)
	}

	if len(config.RequiredRoles) != 0 {
		t.Errorf("Expected empty required roles for default config, got %d", len(config.RequiredRoles))
	}

	if config.MaxQueueSize != 10 {
		t.Errorf("Expected default MaxQueueSize 10, got %d", config.MaxQueueSize)
	}
}

func TestStorageService_UserPreferences(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userID := "user123"
	guildID := "guild456"
	prefs := UserTTSPreferences{
		UserID:  userID,
		GuildID: guildID,
		OptedIn: true,
		Settings: UserTTSSettings{
			PreferredVoice: "en-US-Standard-C",
			SpeedModifier:  1.2,
		},
		UpdatedAt: time.Now(),
	}

	// Test saving user preferences
	err = service.SaveUserPreferences(prefs)
	if err != nil {
		t.Fatalf("Failed to save user preferences: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tempDir, "user_user123_guild456.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("User preferences file was not created")
	}

	// Test loading user preferences
	loadedPrefs, err := service.LoadUserPreferences(userID, guildID)
	if err != nil {
		t.Fatalf("Failed to load user preferences: %v", err)
	}

	if loadedPrefs.UserID != prefs.UserID {
		t.Errorf("Expected UserID %s, got %s", prefs.UserID, loadedPrefs.UserID)
	}

	if loadedPrefs.GuildID != prefs.GuildID {
		t.Errorf("Expected GuildID %s, got %s", prefs.GuildID, loadedPrefs.GuildID)
	}

	if loadedPrefs.OptedIn != prefs.OptedIn {
		t.Errorf("Expected OptedIn %t, got %t", prefs.OptedIn, loadedPrefs.OptedIn)
	}

	if loadedPrefs.Settings.PreferredVoice != prefs.Settings.PreferredVoice {
		t.Errorf("Expected PreferredVoice %s, got %s", prefs.Settings.PreferredVoice, loadedPrefs.Settings.PreferredVoice)
	}

	if loadedPrefs.Settings.SpeedModifier != prefs.Settings.SpeedModifier {
		t.Errorf("Expected SpeedModifier %f, got %f", prefs.Settings.SpeedModifier, loadedPrefs.Settings.SpeedModifier)
	}
}

func TestStorageService_UserPreferences_Default(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	userID := "newuser"
	guildID := "newguild"

	// Test loading non-existent user preferences returns default
	prefs, err := service.LoadUserPreferences(userID, guildID)
	if err != nil {
		t.Fatalf("Failed to load default user preferences: %v", err)
	}

	if prefs.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, prefs.UserID)
	}

	if prefs.GuildID != guildID {
		t.Errorf("Expected GuildID %s, got %s", guildID, prefs.GuildID)
	}

	if prefs.OptedIn != false {
		t.Errorf("Expected OptedIn false for default preferences, got %t", prefs.OptedIn)
	}

	if prefs.Settings.SpeedModifier != 1.0 {
		t.Errorf("Expected default SpeedModifier 1.0, got %f", prefs.Settings.SpeedModifier)
	}
}

func TestStorageService_ChannelPairing(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	pairing := ChannelPairingStorage{
		GuildID:        "guild123",
		VoiceChannelID: "voice456",
		TextChannelID:  "text789",
		CreatedBy:      "user123",
		CreatedAt:      time.Now(),
		IsActive:       true,
	}

	// Test saving channel pairing
	err = service.SaveChannelPairing(pairing)
	if err != nil {
		t.Fatalf("Failed to save channel pairing: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tempDir, "pairing_guild123_voice456.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Channel pairing file was not created")
	}

	// Test loading channel pairing
	loadedPairing, err := service.LoadChannelPairing(pairing.GuildID, pairing.VoiceChannelID)
	if err != nil {
		t.Fatalf("Failed to load channel pairing: %v", err)
	}

	if loadedPairing.GuildID != pairing.GuildID {
		t.Errorf("Expected GuildID %s, got %s", pairing.GuildID, loadedPairing.GuildID)
	}

	if loadedPairing.VoiceChannelID != pairing.VoiceChannelID {
		t.Errorf("Expected VoiceChannelID %s, got %s", pairing.VoiceChannelID, loadedPairing.VoiceChannelID)
	}

	if loadedPairing.TextChannelID != pairing.TextChannelID {
		t.Errorf("Expected TextChannelID %s, got %s", pairing.TextChannelID, loadedPairing.TextChannelID)
	}

	if loadedPairing.CreatedBy != pairing.CreatedBy {
		t.Errorf("Expected CreatedBy %s, got %s", pairing.CreatedBy, loadedPairing.CreatedBy)
	}

	if loadedPairing.IsActive != pairing.IsActive {
		t.Errorf("Expected IsActive %t, got %t", pairing.IsActive, loadedPairing.IsActive)
	}
}

func TestStorageService_RemoveChannelPairing(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	pairing := ChannelPairingStorage{
		GuildID:        "guild123",
		VoiceChannelID: "voice456",
		TextChannelID:  "text789",
		CreatedBy:      "user123",
		CreatedAt:      time.Now(),
		IsActive:       true,
	}

	// Save pairing first
	err = service.SaveChannelPairing(pairing)
	if err != nil {
		t.Fatalf("Failed to save channel pairing: %v", err)
	}

	// Test removing channel pairing
	err = service.RemoveChannelPairing(pairing.GuildID, pairing.VoiceChannelID)
	if err != nil {
		t.Fatalf("Failed to remove channel pairing: %v", err)
	}

	// Verify file was removed
	filePath := filepath.Join(tempDir, "pairing_guild123_voice456.json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Channel pairing file was not removed")
	}

	// Test loading removed pairing returns error
	_, err = service.LoadChannelPairing(pairing.GuildID, pairing.VoiceChannelID)
	if err == nil {
		t.Error("Expected error when loading removed channel pairing")
	}
}

func TestStorageService_ListGuildPairings(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	guildID := "guild123"

	// Create multiple pairings
	pairings := []ChannelPairingStorage{
		{
			GuildID:        guildID,
			VoiceChannelID: "voice1",
			TextChannelID:  "text1",
			CreatedBy:      "user1",
			CreatedAt:      time.Now(),
			IsActive:       true,
		},
		{
			GuildID:        guildID,
			VoiceChannelID: "voice2",
			TextChannelID:  "text2",
			CreatedBy:      "user2",
			CreatedAt:      time.Now(),
			IsActive:       true,
		},
		{
			GuildID:        guildID,
			VoiceChannelID: "voice3",
			TextChannelID:  "text3",
			CreatedBy:      "user3",
			CreatedAt:      time.Now(),
			IsActive:       false, // Inactive pairing
		},
	}

	// Save all pairings
	for _, pairing := range pairings {
		err = service.SaveChannelPairing(pairing)
		if err != nil {
			t.Fatalf("Failed to save channel pairing: %v", err)
		}
	}

	// Test listing guild pairings
	activePairings, err := service.ListGuildPairings(guildID)
	if err != nil {
		t.Fatalf("Failed to list guild pairings: %v", err)
	}

	// Should only return active pairings
	if len(activePairings) != 2 {
		t.Errorf("Expected 2 active pairings, got %d", len(activePairings))
	}

	// Verify the active pairings are correct
	foundVoice1 := false
	foundVoice2 := false
	for _, pairing := range activePairings {
		if pairing.VoiceChannelID == "voice1" {
			foundVoice1 = true
		}
		if pairing.VoiceChannelID == "voice2" {
			foundVoice2 = true
		}
		if pairing.VoiceChannelID == "voice3" {
			t.Error("Inactive pairing should not be returned")
		}
	}

	if !foundVoice1 || !foundVoice2 {
		t.Error("Expected active pairings not found in results")
	}
}

func TestStorageService_ListOptedInUsers(t *testing.T) {
	tempDir := t.TempDir()
	service, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	guildID := "guild123"

	// Create multiple user preferences
	users := []UserTTSPreferences{
		{
			UserID:  "user1",
			GuildID: guildID,
			OptedIn: true,
			Settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-A",
				SpeedModifier:  1.0,
			},
		},
		{
			UserID:  "user2",
			GuildID: guildID,
			OptedIn: true,
			Settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-B",
				SpeedModifier:  1.2,
			},
		},
		{
			UserID:  "user3",
			GuildID: guildID,
			OptedIn: false, // Opted out user
			Settings: UserTTSSettings{
				PreferredVoice: "en-US-Standard-C",
				SpeedModifier:  0.8,
			},
		},
	}

	// Save all user preferences
	for _, user := range users {
		err = service.SaveUserPreferences(user)
		if err != nil {
			t.Fatalf("Failed to save user preferences: %v", err)
		}
	}

	// Test listing opted-in users
	optedInUsers, err := service.ListOptedInUsers(guildID)
	if err != nil {
		t.Fatalf("Failed to list opted-in users: %v", err)
	}

	// Should only return opted-in users
	if len(optedInUsers) != 2 {
		t.Errorf("Expected 2 opted-in users, got %d", len(optedInUsers))
	}

	// Verify the opted-in users are correct
	foundUser1 := false
	foundUser2 := false
	for _, userID := range optedInUsers {
		if userID == "user1" {
			foundUser1 = true
		}
		if userID == "user2" {
			foundUser2 = true
		}
		if userID == "user3" {
			t.Error("Opted-out user should not be returned")
		}
	}

	if !foundUser1 || !foundUser2 {
		t.Error("Expected opted-in users not found in results")
	}
}
