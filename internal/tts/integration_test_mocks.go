package tts

import (
	"fmt"
	"sync"
	"time"
)

// Mock implementations for integration testing

// mockVoiceManagerIntegration provides a comprehensive mock for voice management
type mockVoiceManagerIntegration struct {
	connections map[string]*VoiceConnection
	mu          sync.RWMutex
}

func newMockVoiceManagerIntegration() *mockVoiceManagerIntegration {
	return &mockVoiceManagerIntegration{
		connections: make(map[string]*VoiceConnection),
	}
}

func (m *mockVoiceManagerIntegration) JoinChannel(guildID, channelID string) (*VoiceConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		IsPlaying: false,
	}
	m.connections[guildID] = conn
	return conn, nil
}

func (m *mockVoiceManagerIntegration) LeaveChannel(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.connections, guildID)
	return nil
}

func (m *mockVoiceManagerIntegration) GetConnection(guildID string) (*VoiceConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[guildID]
	return conn, exists
}

func (m *mockVoiceManagerIntegration) PlayAudio(guildID string, audioData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[guildID]; exists {
		conn.IsPlaying = true
		// Simulate audio playback time
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.mu.Lock()
			conn.IsPlaying = false
			m.mu.Unlock()
		}()
		return nil
	}
	return fmt.Errorf("no voice connection for guild %s", guildID)
}

func (m *mockVoiceManagerIntegration) IsConnected(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.connections[guildID]
	return exists
}

func (m *mockVoiceManagerIntegration) PausePlayback(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[guildID]; exists {
		conn.IsPlaying = false
		return nil
	}
	return fmt.Errorf("no voice connection for guild %s", guildID)
}

func (m *mockVoiceManagerIntegration) ResumePlayback(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[guildID]; exists {
		conn.IsPlaying = true
		return nil
	}
	return fmt.Errorf("no voice connection for guild %s", guildID)
}

func (m *mockVoiceManagerIntegration) SkipCurrentMessage(guildID string) error {
	// Mock implementation - just return success
	return nil
}

func (m *mockVoiceManagerIntegration) IsPaused(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if conn, exists := m.connections[guildID]; exists {
		return !conn.IsPlaying
	}
	return false
}

func (m *mockVoiceManagerIntegration) RecoverConnection(guildID string) error {
	// Mock recovery - recreate connection if it doesn't exist
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[guildID]; !exists {
		m.connections[guildID] = &VoiceConnection{
			GuildID:   guildID,
			ChannelID: "recovered-channel",
			IsPlaying: false,
		}
	}
	return nil
}

func (m *mockVoiceManagerIntegration) HealthCheck() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]error)
	for guildID := range m.connections {
		result[guildID] = nil // All connections are healthy in mock
	}
	return result
}

func (m *mockVoiceManagerIntegration) SetConnectionStateCallback(callback func(guildID string, connected bool)) {
	// Mock implementation - store callback but don't use it
}

func (m *mockVoiceManagerIntegration) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections = make(map[string]*VoiceConnection)
	return nil
}

func (m *mockVoiceManagerIntegration) GetActiveConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var guilds []string
	for guildID := range m.connections {
		guilds = append(guilds, guildID)
	}
	return guilds
}

// mockTTSManagerIntegration provides a comprehensive mock for TTS management
type mockTTSManagerIntegration struct {
	voiceConfigs map[string]TTSConfig
	mu           sync.RWMutex
	messageQueue MessageQueue
}

func newMockTTSManagerIntegration() *mockTTSManagerIntegration {
	return &mockTTSManagerIntegration{
		voiceConfigs: make(map[string]TTSConfig),
		messageQueue: nil, // Will be set by the test environment
	}
}

func (m *mockTTSManagerIntegration) setMessageQueue(queue MessageQueue) {
	m.messageQueue = queue
}

func (m *mockTTSManagerIntegration) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	if len(text) > MaxMessageLength {
		return nil, fmt.Errorf("text too long")
	}

	// Simulate TTS conversion
	audioData := []byte(fmt.Sprintf("audio_data_for_%s", text))
	return audioData, nil
}

func (m *mockTTSManagerIntegration) ProcessMessageQueue(guildID string) error {
	// Use the message queue from the integration environment
	for {
		message, err := m.messageQueue.Dequeue(guildID)
		if err != nil || message == nil {
			break
		}

		config := m.getVoiceConfig(guildID)
		_, err = m.ConvertToSpeech(message.Content, config.Voice, config)
		if err != nil {
			// Log error but continue processing
			continue
		}
	}
	return nil
}

func (m *mockTTSManagerIntegration) SetVoiceConfig(guildID string, config TTSConfig) error {
	if err := validateTTSConfig(config); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.voiceConfigs[guildID] = config
	return nil
}

func (m *mockTTSManagerIntegration) GetSupportedVoices() []Voice {
	return []Voice{
		{ID: "en-US-Standard-A", Name: "English (US) - Standard A", Language: "en-US"},
		{ID: "en-US-Standard-B", Name: "English (US) - Standard B", Language: "en-US"},
		{ID: "en-US-Wavenet-A", Name: "English (US) - Wavenet A", Language: "en-US"},
		{ID: "en-US-Wavenet-B", Name: "English (US) - Wavenet B", Language: "en-US"},
	}
}

func (m *mockTTSManagerIntegration) getVoiceConfig(guildID string) TTSConfig {
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

// mockMessageQueueIntegration provides a comprehensive mock for message queuing
type mockMessageQueueIntegration struct {
	queues   map[string][]*QueuedMessage
	maxSizes map[string]int
	mu       sync.RWMutex
}

func newMockMessageQueueIntegration() *mockMessageQueueIntegration {
	return &mockMessageQueueIntegration{
		queues:   make(map[string][]*QueuedMessage),
		maxSizes: make(map[string]int),
	}
}

func (m *mockMessageQueueIntegration) Enqueue(message *QueuedMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	guildID := message.GuildID
	maxSize := m.getMaxSize(guildID)

	if len(m.queues[guildID]) >= maxSize {
		// Remove oldest message if queue is full
		if len(m.queues[guildID]) > 0 {
			m.queues[guildID] = m.queues[guildID][1:]
		}
	}

	m.queues[guildID] = append(m.queues[guildID], message)
	return nil
}

func (m *mockMessageQueueIntegration) Dequeue(guildID string) (*QueuedMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	queue := m.queues[guildID]
	if len(queue) == 0 {
		return nil, nil
	}

	message := queue[0]
	m.queues[guildID] = queue[1:]
	return message, nil
}

func (m *mockMessageQueueIntegration) Clear(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queues[guildID] = nil
	return nil
}

func (m *mockMessageQueueIntegration) Size(guildID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.queues[guildID])
}

func (m *mockMessageQueueIntegration) SetMaxSize(guildID string, size int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.maxSizes[guildID] = size
	return nil
}

func (m *mockMessageQueueIntegration) getMaxSize(guildID string) int {
	if size, exists := m.maxSizes[guildID]; exists {
		return size
	}
	return DefaultMaxQueueSize
}

func (m *mockMessageQueueIntegration) SkipNext(guildID string) (*QueuedMessage, error) {
	// Same as Dequeue for mock implementation
	return m.Dequeue(guildID)
}

// mockChannelServiceIntegration provides a comprehensive mock for channel management
type mockChannelServiceIntegration struct {
	pairings map[string]*ChannelPairing
	mu       sync.RWMutex
}

func newMockChannelServiceIntegration() *mockChannelServiceIntegration {
	return &mockChannelServiceIntegration{
		pairings: make(map[string]*ChannelPairing),
	}
}

func (m *mockChannelServiceIntegration) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	return m.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, "")
}

func (m *mockChannelServiceIntegration) CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", guildID, voiceChannelID)
	m.pairings[key] = &ChannelPairing{
		GuildID:        guildID,
		VoiceChannelID: voiceChannelID,
		TextChannelID:  textChannelID,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
	}
	return nil
}

func (m *mockChannelServiceIntegration) RemovePairing(guildID, voiceChannelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", guildID, voiceChannelID)
	delete(m.pairings, key)
	return nil
}

func (m *mockChannelServiceIntegration) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", guildID, voiceChannelID)
	if pairing, exists := m.pairings[key]; exists {
		return pairing, nil
	}
	return nil, fmt.Errorf("no pairing found for guild %s, voice channel %s", guildID, voiceChannelID)
}

func (m *mockChannelServiceIntegration) ValidateChannelAccess(userID, channelID string) error {
	// Always allow access in integration tests
	return nil
}

func (m *mockChannelServiceIntegration) IsChannelPaired(guildID, textChannelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pairing := range m.pairings {
		if pairing.GuildID == guildID && pairing.TextChannelID == textChannelID {
			return true
		}
	}
	return false
}

// mockPermissionServiceIntegration provides a comprehensive mock for permission management
type mockPermissionServiceIntegration struct {
	canInviteBot  map[string]bool     // "userID:guildID" -> canInvite
	canControlBot map[string]bool     // "userID:guildID" -> canControl
	requiredRoles map[string][]string // guildID -> roleIDs
	mu            sync.RWMutex
}

func newMockPermissionServiceIntegration() *mockPermissionServiceIntegration {
	return &mockPermissionServiceIntegration{
		canInviteBot:  make(map[string]bool),
		canControlBot: make(map[string]bool),
		requiredRoles: make(map[string][]string),
	}
}

func (m *mockPermissionServiceIntegration) CanInviteBot(userID, guildID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	return m.canInviteBot[key], nil
}

func (m *mockPermissionServiceIntegration) CanControlBot(userID, guildID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	return m.canControlBot[key], nil
}

func (m *mockPermissionServiceIntegration) HasChannelAccess(userID, channelID string) (bool, error) {
	// Always allow channel access in integration tests
	return true, nil
}

func (m *mockPermissionServiceIntegration) SetRequiredRoles(guildID string, roleIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requiredRoles[guildID] = roleIDs
	return nil
}

func (m *mockPermissionServiceIntegration) GetRequiredRoles(guildID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if roles, exists := m.requiredRoles[guildID]; exists {
		return roles, nil
	}
	return []string{}, nil
}

func (m *mockPermissionServiceIntegration) setCanInviteBot(userID, guildID string, canInvite bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	m.canInviteBot[key] = canInvite
}

func (m *mockPermissionServiceIntegration) setCanControlBot(userID, guildID string, canControl bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	m.canControlBot[key] = canControl
}

// mockUserServiceIntegration provides a comprehensive mock for user management
type mockUserServiceIntegration struct {
	optInStatus map[string]bool // "userID:guildID" -> optedIn
	mu          sync.RWMutex
}

func newMockUserServiceIntegration() *mockUserServiceIntegration {
	return &mockUserServiceIntegration{
		optInStatus: make(map[string]bool),
	}
}

func (m *mockUserServiceIntegration) SetOptInStatus(userID, guildID string, optedIn bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	m.optInStatus[key] = optedIn
	return nil
}

func (m *mockUserServiceIntegration) IsOptedIn(userID, guildID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, guildID)
	return m.optInStatus[key], nil
}

func (m *mockUserServiceIntegration) GetOptedInUsers(guildID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var users []string
	for key, optedIn := range m.optInStatus {
		if optedIn {
			parts := splitKey(key)
			if len(parts) == 2 && parts[1] == guildID {
				users = append(users, parts[0])
			}
		}
	}
	return users, nil
}

func (m *mockUserServiceIntegration) AutoOptIn(userID, guildID string) error {
	return m.SetOptInStatus(userID, guildID, true)
}

// mockConfigServiceIntegration provides a comprehensive mock for configuration management
type mockConfigServiceIntegration struct {
	configs map[string]*GuildTTSConfig
	mu      sync.RWMutex
}

func newMockConfigServiceIntegration() *mockConfigServiceIntegration {
	return &mockConfigServiceIntegration{
		configs: make(map[string]*GuildTTSConfig),
	}
}

func (m *mockConfigServiceIntegration) GetGuildConfig(guildID string) (*GuildTTSConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if config, exists := m.configs[guildID]; exists {
		return config, nil
	}

	// Return default config
	return &GuildTTSConfig{
		GuildID:       guildID,
		RequiredRoles: []string{},
		TTSSettings: TTSConfig{
			Voice:  DefaultVoice,
			Speed:  DefaultTTSSpeed,
			Volume: DefaultTTSVolume,
			Format: AudioFormatDCA,
		},
		MaxQueueSize: DefaultMaxQueueSize,
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *mockConfigServiceIntegration) SaveGuildConfig(config *GuildTTSConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.UpdatedAt = time.Now()
	m.configs[config.GuildID] = config
	return nil
}

func (m *mockConfigServiceIntegration) SetGuildConfig(guildID string, config *GuildTTSConfig) error {
	return m.SaveGuildConfig(config)
}

func (m *mockConfigServiceIntegration) SetRequiredRoles(guildID string, roleIDs []string) error {
	config, err := m.GetGuildConfig(guildID)
	if err != nil {
		return err
	}
	config.RequiredRoles = roleIDs
	return m.SaveGuildConfig(config)
}

func (m *mockConfigServiceIntegration) GetRequiredRoles(guildID string) ([]string, error) {
	config, err := m.GetGuildConfig(guildID)
	if err != nil {
		return nil, err
	}
	return config.RequiredRoles, nil
}

func (m *mockConfigServiceIntegration) SetTTSSettings(guildID string, settings TTSConfig) error {
	config, err := m.GetGuildConfig(guildID)
	if err != nil {
		return err
	}
	config.TTSSettings = settings
	return m.SaveGuildConfig(config)
}

func (m *mockConfigServiceIntegration) GetTTSSettings(guildID string) (*TTSConfig, error) {
	config, err := m.GetGuildConfig(guildID)
	if err != nil {
		return nil, err
	}
	return &config.TTSSettings, nil
}

func (m *mockConfigServiceIntegration) SetMaxQueueSize(guildID string, size int) error {
	config, err := m.GetGuildConfig(guildID)
	if err != nil {
		return err
	}
	config.MaxQueueSize = size
	return m.SaveGuildConfig(config)
}

func (m *mockConfigServiceIntegration) GetMaxQueueSize(guildID string) (int, error) {
	config, err := m.GetGuildConfig(guildID)
	if err != nil {
		return DefaultMaxQueueSize, err
	}
	return config.MaxQueueSize, nil
}

func (m *mockConfigServiceIntegration) ValidateConfig(config *GuildTTSConfig) error {
	if config.GuildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}
	if config.MaxQueueSize < 1 || config.MaxQueueSize > 100 {
		return fmt.Errorf("invalid max queue size: %d", config.MaxQueueSize)
	}
	return nil
}

// Helper function to split key
func splitKey(key string) []string {
	for i, char := range key {
		if char == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}
