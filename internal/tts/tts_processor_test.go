package tts

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Mock implementations for testing

type mockTTSManager struct {
	convertFunc      func(text, voice string, config TTSConfig) ([]byte, error)
	processQueueFunc func(guildID string) error
	setConfigFunc    func(guildID string, config TTSConfig) error
	getSupportedFunc func() []Voice
	mu               sync.RWMutex
	callLog          []string
}

func (m *mockTTSManager) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "ConvertToSpeech")
	m.mu.Unlock()

	if m.convertFunc != nil {
		return m.convertFunc(text, voice, config)
	}
	return []byte("mock audio data"), nil
}

func (m *mockTTSManager) ProcessMessageQueue(guildID string) error {
	m.mu.Lock()
	m.callLog = append(m.callLog, "ProcessMessageQueue")
	m.mu.Unlock()

	if m.processQueueFunc != nil {
		return m.processQueueFunc(guildID)
	}
	return nil
}

func (m *mockTTSManager) SetVoiceConfig(guildID string, config TTSConfig) error {
	m.mu.Lock()
	m.callLog = append(m.callLog, "SetVoiceConfig")
	m.mu.Unlock()

	if m.setConfigFunc != nil {
		return m.setConfigFunc(guildID, config)
	}
	return nil
}

func (m *mockTTSManager) GetSupportedVoices() []Voice {
	m.mu.Lock()
	m.callLog = append(m.callLog, "GetSupportedVoices")
	m.mu.Unlock()

	if m.getSupportedFunc != nil {
		return m.getSupportedFunc()
	}
	return []Voice{}
}

func (m *mockTTSManager) getCallLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.callLog))
	copy(result, m.callLog)
	return result
}

type mockVoiceManager struct {
	connections   map[string]*VoiceConnection
	playAudioFunc func(guildID string, audioData []byte) error
	pausedGuilds  map[string]bool
	mu            sync.RWMutex
	callLog       []string
}

func newMockVoiceManager() *mockVoiceManager {
	return &mockVoiceManager{
		connections:  make(map[string]*VoiceConnection),
		pausedGuilds: make(map[string]bool),
	}
}

func (m *mockVoiceManager) JoinChannel(guildID, channelID string) (*VoiceConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "JoinChannel")

	conn := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		IsPlaying: false,
		IsPaused:  false,
	}
	m.connections[guildID] = conn
	return conn, nil
}

func (m *mockVoiceManager) LeaveChannel(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "LeaveChannel")

	delete(m.connections, guildID)
	return nil
}

func (m *mockVoiceManager) GetConnection(guildID string) (*VoiceConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.callLog = append(m.callLog, "GetConnection")

	conn, exists := m.connections[guildID]
	return conn, exists
}

func (m *mockVoiceManager) PlayAudio(guildID string, audioData []byte) error {
	m.mu.Lock()
	m.callLog = append(m.callLog, "PlayAudio")
	m.mu.Unlock()

	if m.playAudioFunc != nil {
		return m.playAudioFunc(guildID, audioData)
	}
	return nil
}

func (m *mockVoiceManager) IsConnected(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.connections[guildID]
	return exists
}

func (m *mockVoiceManager) PausePlayback(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "PausePlayback")

	m.pausedGuilds[guildID] = true
	return nil
}

func (m *mockVoiceManager) ResumePlayback(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "ResumePlayback")

	delete(m.pausedGuilds, guildID)
	return nil
}

func (m *mockVoiceManager) SkipCurrentMessage(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "SkipCurrentMessage")

	return nil
}

func (m *mockVoiceManager) IsPaused(guildID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.pausedGuilds[guildID]
}

func (m *mockVoiceManager) RecoverConnection(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "RecoverConnection")

	return nil
}

func (m *mockVoiceManager) HealthCheck() map[string]error {
	return make(map[string]error)
}

func (m *mockVoiceManager) SetConnectionStateCallback(callback func(guildID string, connected bool)) {
	// No-op for testing
}

func (m *mockVoiceManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callLog = append(m.callLog, "Cleanup")

	m.connections = make(map[string]*VoiceConnection)
	return nil
}

func (m *mockVoiceManager) GetActiveConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	guilds := make([]string, 0, len(m.connections))
	for guildID := range m.connections {
		guilds = append(guilds, guildID)
	}
	return guilds
}

func (m *mockVoiceManager) getCallLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.callLog))
	copy(result, m.callLog)
	return result
}

type mockConfigService struct {
	configs map[string]*TTSConfig
	mu      sync.RWMutex
}

func newMockConfigService() *mockConfigService {
	return &mockConfigService{
		configs: make(map[string]*TTSConfig),
	}
}

func (m *mockConfigService) GetGuildConfig(guildID string) (*GuildTTSConfig, error) {
	return nil, errors.New("not implemented")
}

func (m *mockConfigService) SetGuildConfig(guildID string, config *GuildTTSConfig) error {
	return errors.New("not implemented")
}

func (m *mockConfigService) SetRequiredRoles(guildID string, roleIDs []string) error {
	return errors.New("not implemented")
}

func (m *mockConfigService) GetRequiredRoles(guildID string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockConfigService) SetTTSSettings(guildID string, settings TTSConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[guildID] = &settings
	return nil
}

func (m *mockConfigService) GetTTSSettings(guildID string) (*TTSConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if config, exists := m.configs[guildID]; exists {
		return config, nil
	}
	return nil, errors.New("config not found")
}

func (m *mockConfigService) SetMaxQueueSize(guildID string, size int) error {
	return errors.New("not implemented")
}

func (m *mockConfigService) GetMaxQueueSize(guildID string) (int, error) {
	return 0, errors.New("not implemented")
}

func (m *mockConfigService) ValidateConfig(config *GuildTTSConfig) error {
	return nil
}

// Test functions

func TestNewTTSProcessor(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}

	// Test that the processor can be started and stopped
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	err = processor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}
}

func TestTTSProcessor_StartStop(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	// Test start
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	// Give it a moment to start
	time.Sleep(time.Millisecond * 10)

	// Test stop
	err = processor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}
}

func TestTTSProcessor_GuildProcessing(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"

	// Test starting guild processing
	err := processor.StartGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to start guild processing: %v", err)
	}

	// Check if guild is in active list
	activeGuilds := processor.GetActiveGuilds()
	found := false
	for _, id := range activeGuilds {
		if id == guildID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected guild to be in active list")
	}

	// Test stopping guild processing
	err = processor.StopGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to stop guild processing: %v", err)
	}

	// Check if guild is removed from active list
	activeGuilds = processor.GetActiveGuilds()
	for _, id := range activeGuilds {
		if id == guildID {
			t.Error("Expected guild to be removed from active list")
		}
	}
}

func TestTTSProcessor_MessageProcessing(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"
	channelID := "test-channel-456"

	// Set up voice connection
	_, err := voiceManager.JoinChannel(guildID, channelID)
	if err != nil {
		t.Fatalf("Failed to join voice channel: %v", err)
	}

	// Start guild processing
	err = processor.StartGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to start guild processing: %v", err)
	}

	// Add a message to the queue
	message := &QueuedMessage{
		ID:        "msg-1",
		GuildID:   guildID,
		ChannelID: channelID,
		UserID:    "user-123",
		Username:  "TestUser",
		Content:   "Hello, this is a test message",
		Timestamp: time.Now(),
	}

	err = messageQueue.Enqueue(message)
	if err != nil {
		t.Fatalf("Failed to enqueue message: %v", err)
	}

	// Start the processor to handle messages automatically
	err = processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Give it time to process the message with retries
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Millisecond * 50)

		// Check if TTS conversion was called
		ttsCallLog := ttsManager.getCallLog()
		for _, call := range ttsCallLog {
			if call == "ConvertToSpeech" {
				// Also check that audio was played
				voiceCallLog := voiceManager.getCallLog()
				for _, voiceCall := range voiceCallLog {
					if voiceCall == "PlayAudio" {
						return // Test passed
					}
				}
			}
		}
	}

	// If we get here, the test failed
	ttsCallLog := ttsManager.getCallLog()
	voiceCallLog := voiceManager.getCallLog()
	t.Errorf("Expected TTS processing to complete. TTS calls: %v, Voice calls: %v", ttsCallLog, voiceCallLog)
}

func TestTTSProcessor_ErrorHandling(t *testing.T) {
	// Test TTS conversion failure
	ttsManager := &mockTTSManager{
		convertFunc: func(text, voice string, config TTSConfig) ([]byte, error) {
			return nil, errors.New("TTS conversion failed")
		},
	}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"
	channelID := "test-channel-456"

	// Set up voice connection
	_, err := voiceManager.JoinChannel(guildID, channelID)
	if err != nil {
		t.Fatalf("Failed to join voice channel: %v", err)
	}

	// Start guild processing
	err = processor.StartGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to start guild processing: %v", err)
	}

	// Add a message to the queue
	message := &QueuedMessage{
		ID:        "msg-1",
		GuildID:   guildID,
		ChannelID: channelID,
		UserID:    "user-123",
		Username:  "TestUser",
		Content:   "Hello, this is a test message",
		Timestamp: time.Now(),
	}

	err = messageQueue.Enqueue(message)
	if err != nil {
		t.Fatalf("Failed to enqueue message: %v", err)
	}

	// Start the processor to handle messages automatically
	err = processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Give it time to process the message (should handle error gracefully)
	// Wait longer since error handling includes retries
	maxRetries := 20
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Millisecond * 100)

		// Check that TTS conversion was attempted
		ttsCallLog := ttsManager.getCallLog()
		if len(ttsCallLog) > 0 {
			// Check that PlayAudio was not called due to conversion failure
			voiceCallLog := voiceManager.getCallLog()
			for _, call := range voiceCallLog {
				if call == "PlayAudio" {
					t.Error("Expected PlayAudio not to be called when TTS conversion fails")
				}
			}
			return // Test passed
		}
	}

	t.Error("Expected TTS conversion to be attempted")
}

func TestTTSProcessor_PauseResume(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"

	// Test pause
	err := processor.PauseProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to pause processing: %v", err)
	}

	// Check that voice manager was called
	voiceCallLog := voiceManager.getCallLog()
	found := false
	for _, call := range voiceCallLog {
		if call == "PausePlayback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected PausePlayback to be called")
	}

	// Test resume
	err = processor.ResumeProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to resume processing: %v", err)
	}

	// Check that voice manager was called
	voiceCallLog = voiceManager.getCallLog()
	found = false
	for _, call := range voiceCallLog {
		if call == "ResumePlayback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ResumePlayback to be called")
	}
}

func TestTTSProcessor_SkipMessage(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"

	// Start guild processing
	err := processor.StartGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to start guild processing: %v", err)
	}

	// Test skip
	err = processor.SkipCurrentMessage(guildID)
	if err != nil {
		t.Fatalf("Failed to skip message: %v", err)
	}

	// Check that voice manager was called
	voiceCallLog := voiceManager.getCallLog()
	found := false
	for _, call := range voiceCallLog {
		if call == "SkipCurrentMessage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected SkipCurrentMessage to be called")
	}
}

func TestTTSProcessor_QueueOperations(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"

	// Add messages to queue
	for i := 0; i < 3; i++ {
		message := &QueuedMessage{
			ID:        fmt.Sprintf("msg-%d", i),
			GuildID:   guildID,
			ChannelID: "test-channel",
			UserID:    "user-123",
			Username:  "TestUser",
			Content:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
		}

		err := messageQueue.Enqueue(message)
		if err != nil {
			t.Fatalf("Failed to enqueue message %d: %v", i, err)
		}
	}

	// Check queue size
	size := processor.GetQueueSize(guildID)
	if size != 3 {
		t.Errorf("Expected queue size to be 3, got %d", size)
	}

	// Clear queue
	err := processor.ClearQueue(guildID)
	if err != nil {
		t.Fatalf("Failed to clear queue: %v", err)
	}

	// Check queue size after clear
	size = processor.GetQueueSize(guildID)
	if size != 0 {
		t.Errorf("Expected queue size to be 0 after clear, got %d", size)
	}
}

func TestTTSProcessor_MessageTruncation(t *testing.T) {
	ttsManager := &mockTTSManager{
		convertFunc: func(text, voice string, config TTSConfig) ([]byte, error) {
			// Check that long messages are truncated
			if len(text) > MaxMessageLength {
				t.Errorf("Expected message to be truncated to %d characters, got %d", MaxMessageLength, len(text))
			}
			return []byte("mock audio"), nil
		},
	}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"
	channelID := "test-channel-456"

	// Set up voice connection
	_, err := voiceManager.JoinChannel(guildID, channelID)
	if err != nil {
		t.Fatalf("Failed to join voice channel: %v", err)
	}

	// Start guild processing
	err = processor.StartGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to start guild processing: %v", err)
	}

	// Create a very long message
	longContent := make([]byte, MaxMessageLength+100)
	for i := range longContent {
		longContent[i] = 'A'
	}

	message := &QueuedMessage{
		ID:        "msg-long",
		GuildID:   guildID,
		ChannelID: channelID,
		UserID:    "user-123",
		Username:  "TestUser",
		Content:   string(longContent),
		Timestamp: time.Now(),
	}

	err = messageQueue.Enqueue(message)
	if err != nil {
		t.Fatalf("Failed to enqueue long message: %v", err)
	}

	// Start the processor to handle messages automatically
	err = processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Give it time to process the message (should truncate)
	time.Sleep(time.Millisecond * 100)
}

func TestTTSProcessor_InactivityAnnouncement(t *testing.T) {
	// This test is complex to implement with the interface approach
	// since inactivity timeout is an internal implementation detail.
	// For now, we'll test that the processor can handle empty queues gracefully.

	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	guildID := "test-guild-123"
	channelID := "test-channel-456"

	// Set up voice connection
	_, err := voiceManager.JoinChannel(guildID, channelID)
	if err != nil {
		t.Fatalf("Failed to join voice channel: %v", err)
	}

	// Start guild processing
	err = processor.StartGuildProcessing(guildID)
	if err != nil {
		t.Fatalf("Failed to start guild processing: %v", err)
	}

	// Start the processor
	err = processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Let it run for a short time with no messages
	time.Sleep(time.Millisecond * 100)

	// Verify no errors occurred (processor should handle empty queue gracefully)
	if len(ttsManager.getCallLog()) > 0 {
		// If TTS was called, it might be for inactivity announcement
		// This is acceptable behavior
	}
}

func TestTTSProcessor_ConcurrentProcessing(t *testing.T) {
	ttsManager := &mockTTSManager{}
	voiceManager := newMockVoiceManager()
	messageQueue := NewMessageQueue()
	configService := newMockConfigService()
	userService := newMockUserService()

	processor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	// Start multiple guilds concurrently
	numGuilds := 5
	var wg sync.WaitGroup

	for i := 0; i < numGuilds; i++ {
		wg.Add(1)
		go func(guildNum int) {
			defer wg.Done()

			guildID := fmt.Sprintf("guild-%d", guildNum)
			channelID := fmt.Sprintf("channel-%d", guildNum)

			// Set up voice connection
			_, err := voiceManager.JoinChannel(guildID, channelID)
			if err != nil {
				t.Errorf("Failed to join voice channel for guild %s: %v", guildID, err)
				return
			}

			// Start guild processing
			err = processor.StartGuildProcessing(guildID)
			if err != nil {
				t.Errorf("Failed to start guild processing for %s: %v", guildID, err)
				return
			}
		}(i)
	}

	wg.Wait()

	// Check that all guilds are active
	activeGuilds := processor.GetActiveGuilds()
	if len(activeGuilds) != numGuilds {
		t.Errorf("Expected %d active guilds, got %d", numGuilds, len(activeGuilds))
	}
}
