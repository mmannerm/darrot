package tts

import (
	"errors"
	"fmt"
	"sync"
)

// Mock implementations for error scenario testing

// mockVoiceManagerError provides a voice manager mock that can simulate various error conditions
type mockVoiceManagerError struct {
	connections      map[string]*VoiceConnection
	connectionErrors map[string]error
	playbackErrors   map[string]error
	recoveryAttempts map[string]bool
	mu               sync.RWMutex
}

func newMockVoiceManagerError() *mockVoiceManagerError {
	return &mockVoiceManagerError{
		connections:      make(map[string]*VoiceConnection),
		connectionErrors: make(map[string]error),
		playbackErrors:   make(map[string]error),
		recoveryAttempts: make(map[string]bool),
	}
}

func (m *mockVoiceManagerError) JoinChannel(guildID, channelID string) (*VoiceConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for simulated connection error
	if err, exists := m.connectionErrors[guildID]; exists {
		return nil, err
	}

	conn := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		IsPlaying: false,
	}
	m.connections[guildID] = conn
	return conn, nil
}

func (m *mockVoiceManagerError) LeaveChannel(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.connections, guildID)
	delete(m.connectionErrors, guildID)
	delete(m.playbackErrors, guildID)
	return nil
}

func (m *mockVoiceManagerError) GetConnection(guildID string) (*VoiceConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[guildID]
	return conn, exists
}

func (m *mockVoiceManagerError) PlayAudio(guildID string, audioData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for simulated playback error
	if err, exists := m.playbackErrors[guildID]; exists {
		return err
	}

	if conn, exists := m.connections[guildID]; exists {
		conn.IsPlaying = true
		return nil
	}
	return fmt.Errorf("no voice connection for guild %s", guildID)
}

func (m *mockVoiceManagerError) IsConnected(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.connections[guildID]
	return exists
}

func (m *mockVoiceManagerError) PausePlayback(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[guildID]; exists {
		conn.IsPlaying = false
		return nil
	}
	return fmt.Errorf("no voice connection for guild %s", guildID)
}

func (m *mockVoiceManagerError) ResumePlayback(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[guildID]; exists {
		conn.IsPlaying = true
		return nil
	}
	return fmt.Errorf("no voice connection for guild %s", guildID)
}

func (m *mockVoiceManagerError) SkipCurrentMessage(guildID string) error {
	return nil
}

func (m *mockVoiceManagerError) IsPaused(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if conn, exists := m.connections[guildID]; exists {
		return !conn.IsPlaying
	}
	return false
}

func (m *mockVoiceManagerError) RecoverConnection(guildID string) error {
	return m.attemptRecovery(guildID)
}

func (m *mockVoiceManagerError) HealthCheck() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]error)
	for guildID := range m.connections {
		result[guildID] = nil
	}
	return result
}

func (m *mockVoiceManagerError) SetConnectionStateCallback(callback func(guildID string, connected bool)) {
	// Mock implementation
}

func (m *mockVoiceManagerError) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections = make(map[string]*VoiceConnection)
	m.connectionErrors = make(map[string]error)
	m.playbackErrors = make(map[string]error)
	m.recoveryAttempts = make(map[string]bool)
	return nil
}

func (m *mockVoiceManagerError) GetActiveConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var guilds []string
	for guildID := range m.connections {
		guilds = append(guilds, guildID)
	}
	return guilds
}

// Error simulation methods
func (m *mockVoiceManagerError) setConnectionError(guildID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionErrors[guildID] = err
}

func (m *mockVoiceManagerError) setPlaybackError(guildID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.playbackErrors[guildID] = err
}

func (m *mockVoiceManagerError) simulateConnectionLoss(guildID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.connections, guildID)
}

func (m *mockVoiceManagerError) wasRecoveryAttempted(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.recoveryAttempts[guildID]
}

// Simulate recovery attempt
func (m *mockVoiceManagerError) attemptRecovery(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recoveryAttempts[guildID] = true

	// Simulate successful recovery by recreating connection
	if _, exists := m.connections[guildID]; !exists {
		m.connections[guildID] = &VoiceConnection{
			GuildID:   guildID,
			ChannelID: "recovered-channel",
			IsPlaying: false,
		}
	}
	return nil
}

// mockTTSManagerError provides a TTS manager mock that can simulate various error conditions
type mockTTSManagerError struct {
	voiceConfigs       map[string]TTSConfig
	conversionErrors   map[string]error
	serviceUnavailable bool
	mu                 sync.RWMutex
}

func newMockTTSManagerError() *mockTTSManagerError {
	return &mockTTSManagerError{
		voiceConfigs:     make(map[string]TTSConfig),
		conversionErrors: make(map[string]error),
	}
}

func (m *mockTTSManagerError) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if service is unavailable
	if m.serviceUnavailable {
		return nil, errors.New("TTS service is currently unavailable")
	}

	// Check for text-specific conversion errors
	if err, exists := m.conversionErrors[text]; exists {
		return nil, err
	}

	// Validate input
	if text == "" {
		return nil, errors.New("empty text provided")
	}
	if len(text) > MaxMessageLength {
		return nil, fmt.Errorf("text too long: %d characters (max: %d)", len(text), MaxMessageLength)
	}

	// Simulate successful conversion
	audioData := []byte(fmt.Sprintf("audio_data_for_%s", text))
	return audioData, nil
}

func (m *mockTTSManagerError) ProcessMessageQueue(guildID string) error {
	// This mock doesn't implement queue processing
	return nil
}

func (m *mockTTSManagerError) SetVoiceConfig(guildID string, config TTSConfig) error {
	if err := validateTTSConfig(config); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.voiceConfigs[guildID] = config
	return nil
}

func (m *mockTTSManagerError) GetSupportedVoices() []Voice {
	return []Voice{
		{ID: "en-US-Standard-A", Name: "English (US) - Standard A", Language: "en-US"},
		{ID: "en-US-Standard-B", Name: "English (US) - Standard B", Language: "en-US"},
	}
}

func (m *mockTTSManagerError) getVoiceConfig(guildID string) TTSConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if config, exists := m.voiceConfigs[guildID]; exists {
		return config
	}

	return TTSConfig{
		Voice:  DefaultVoice,
		Speed:  DefaultTTSSpeed,
		Volume: DefaultTTSVolume,
		Format: AudioFormatDCA,
	}
}

// Error simulation methods
func (m *mockTTSManagerError) setConversionError(text string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversionErrors[text] = err
}

func (m *mockTTSManagerError) setServiceUnavailable(unavailable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceUnavailable = unavailable
}

// mockChannelServiceError provides a channel service mock that can simulate error conditions
type mockChannelServiceError struct {
	pairings            map[string]*ChannelPairing
	channelAccessErrors map[string]error // "userID:channelID" -> error
	mu                  sync.RWMutex
}

func newMockChannelServiceError() *mockChannelServiceError {
	return &mockChannelServiceError{
		pairings:            make(map[string]*ChannelPairing),
		channelAccessErrors: make(map[string]error),
	}
}

func (m *mockChannelServiceError) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	return m.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, "")
}

func (m *mockChannelServiceError) CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", guildID, voiceChannelID)
	m.pairings[key] = &ChannelPairing{
		GuildID:        guildID,
		VoiceChannelID: voiceChannelID,
		TextChannelID:  textChannelID,
		CreatedBy:      createdBy,
	}
	return nil
}

func (m *mockChannelServiceError) RemovePairing(guildID, voiceChannelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", guildID, voiceChannelID)
	delete(m.pairings, key)
	return nil
}

func (m *mockChannelServiceError) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", guildID, voiceChannelID)
	if pairing, exists := m.pairings[key]; exists {
		return pairing, nil
	}
	return nil, fmt.Errorf("no pairing found for guild %s, voice channel %s", guildID, voiceChannelID)
}

func (m *mockChannelServiceError) ValidateChannelAccess(userID, channelID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, channelID)
	if err, exists := m.channelAccessErrors[key]; exists {
		return err
	}
	return nil
}

func (m *mockChannelServiceError) IsChannelPaired(guildID, textChannelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pairing := range m.pairings {
		if pairing.GuildID == guildID && pairing.TextChannelID == textChannelID {
			return true
		}
	}
	return false
}

// Error simulation methods
func (m *mockChannelServiceError) setChannelAccessError(userID, channelID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", userID, channelID)
	m.channelAccessErrors[key] = err
}

// mockPermissionServiceError provides a permission service mock that can simulate error conditions
type mockPermissionServiceError struct {
	canInviteBot  map[string]bool     // "userID:guildID" -> canInvite
	canControlBot map[string]bool     // "userID:guildID" -> canControl
	requiredRoles map[string][]string // guildID -> roleIDs
	mu            sync.RWMutex
}

func newMockPermissionServiceError() *mockPermissionServiceError {
	return &mockPermissionServiceError{
		canInviteBot:  make(map[string]bool),
		canControlBot: make(map[string]bool),
		requiredRoles: make(map[string][]string),
	}
}

func (m *mockPermissionServiceError) CanInviteBot(userID, guildID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	return m.canInviteBot[key], nil
}

func (m *mockPermissionServiceError) CanControlBot(userID, guildID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	return m.canControlBot[key], nil
}

func (m *mockPermissionServiceError) HasChannelAccess(userID, channelID string) (bool, error) {
	// Always allow channel access in error tests unless specifically configured
	return true, nil
}

func (m *mockPermissionServiceError) SetRequiredRoles(guildID string, roleIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requiredRoles[guildID] = roleIDs
	return nil
}

func (m *mockPermissionServiceError) GetRequiredRoles(guildID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if roles, exists := m.requiredRoles[guildID]; exists {
		return roles, nil
	}
	return []string{}, nil
}

// Permission configuration methods
func (m *mockPermissionServiceError) setCanInviteBot(userID, guildID string, canInvite bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	m.canInviteBot[key] = canInvite
}

func (m *mockPermissionServiceError) setCanControlBot(userID, guildID string, canControl bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	m.canControlBot[key] = canControl
}

// validateTTSConfig validates TTS configuration for error testing
func validateTTSConfig(config TTSConfig) error {
	if config.Voice == "" {
		return errors.New("voice cannot be empty")
	}
	if config.Speed < 0.25 || config.Speed > 4.0 {
		return fmt.Errorf("invalid speed: %f (must be between 0.25 and 4.0)", config.Speed)
	}
	if config.Volume < 0.0 || config.Volume > 1.0 {
		return fmt.Errorf("invalid volume: %f (must be between 0.0 and 1.0)", config.Volume)
	}
	if config.Format != AudioFormatPCM && config.Format != AudioFormatDCA && config.Format != AudioFormatOpus {
		return fmt.Errorf("invalid audio format: %s", config.Format)
	}
	return nil
}
