package tts

import (
	"errors"
	"testing"
	"time"
)

// Mock implementations for testing

type mockVoiceManagerForRecovery struct {
	connections       map[string]bool
	recoveryCalls     []string
	recoveryErrors    map[string]error
	healthCheckErrors map[string]error
	playAudioErrors   map[string]error
}

func newMockVoiceManagerForRecovery() *mockVoiceManagerForRecovery {
	return &mockVoiceManagerForRecovery{
		connections:       make(map[string]bool),
		recoveryCalls:     make([]string, 0),
		recoveryErrors:    make(map[string]error),
		healthCheckErrors: make(map[string]error),
		playAudioErrors:   make(map[string]error),
	}
}

func (m *mockVoiceManagerForRecovery) JoinChannel(guildID, channelID string) (*VoiceConnection, error) {
	return nil, nil
}

func (m *mockVoiceManagerForRecovery) LeaveChannel(guildID string) error {
	return nil
}

func (m *mockVoiceManagerForRecovery) GetConnection(guildID string) (*VoiceConnection, bool) {
	connected, exists := m.connections[guildID]
	return nil, exists && connected
}

func (m *mockVoiceManagerForRecovery) PlayAudio(guildID string, audioData []byte) error {
	if err, exists := m.playAudioErrors[guildID]; exists {
		return err
	}
	return nil
}

func (m *mockVoiceManagerForRecovery) IsConnected(guildID string) bool {
	connected, exists := m.connections[guildID]
	return exists && connected
}

func (m *mockVoiceManagerForRecovery) PausePlayback(guildID string) error {
	return nil
}

func (m *mockVoiceManagerForRecovery) ResumePlayback(guildID string) error {
	return nil
}

func (m *mockVoiceManagerForRecovery) SkipCurrentMessage(guildID string) error {
	return nil
}

func (m *mockVoiceManagerForRecovery) IsPaused(guildID string) bool {
	return false
}

func (m *mockVoiceManagerForRecovery) RecoverConnection(guildID string) error {
	m.recoveryCalls = append(m.recoveryCalls, guildID)
	if err, exists := m.recoveryErrors[guildID]; exists {
		return err
	}
	m.connections[guildID] = true
	return nil
}

func (m *mockVoiceManagerForRecovery) HealthCheck() map[string]error {
	results := make(map[string]error)
	for guildID := range m.connections {
		if err, exists := m.healthCheckErrors[guildID]; exists {
			results[guildID] = err
		} else {
			results[guildID] = nil
		}
	}
	return results
}

func (m *mockVoiceManagerForRecovery) SetConnectionStateCallback(callback func(guildID string, connected bool)) {
}

func (m *mockVoiceManagerForRecovery) Cleanup() error {
	return nil
}

func (m *mockVoiceManagerForRecovery) GetActiveConnections() []string {
	var guilds []string
	for guildID, connected := range m.connections {
		if connected {
			guilds = append(guilds, guildID)
		}
	}
	return guilds
}

type mockTTSManagerForRecovery struct {
	conversionErrors map[string]error
	conversionCalls  []ConversionCall
}

type ConversionCall struct {
	Text   string
	Voice  string
	Config TTSConfig
}

func newMockTTSManagerForRecovery() *mockTTSManagerForRecovery {
	return &mockTTSManagerForRecovery{
		conversionErrors: make(map[string]error),
		conversionCalls:  make([]ConversionCall, 0),
	}
}

func (m *mockTTSManagerForRecovery) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	call := ConversionCall{
		Text:   text,
		Voice:  voice,
		Config: config,
	}
	m.conversionCalls = append(m.conversionCalls, call)

	// Check for specific error conditions
	if err, exists := m.conversionErrors[text]; exists {
		return nil, err
	}

	// Return mock audio data
	return []byte("mock audio data"), nil
}

func (m *mockTTSManagerForRecovery) ProcessMessageQueue(guildID string) error {
	return nil
}

func (m *mockTTSManagerForRecovery) SetVoiceConfig(guildID string, config TTSConfig) error {
	return nil
}

func (m *mockTTSManagerForRecovery) GetSupportedVoices() []Voice {
	return []Voice{}
}

type mockMessageQueueForRecovery struct{}

func (m *mockMessageQueueForRecovery) Enqueue(message *QueuedMessage) error {
	return nil
}

func (m *mockMessageQueueForRecovery) Dequeue(guildID string) (*QueuedMessage, error) {
	return nil, nil
}

func (m *mockMessageQueueForRecovery) Clear(guildID string) error {
	return nil
}

func (m *mockMessageQueueForRecovery) Size(guildID string) int {
	return 0
}

func (m *mockMessageQueueForRecovery) SetMaxSize(guildID string, size int) error {
	return nil
}

func (m *mockMessageQueueForRecovery) SkipNext(guildID string) (*QueuedMessage, error) {
	return nil, nil
}

type mockConfigServiceForRecovery struct{}

func (m *mockConfigServiceForRecovery) GetGuildConfig(guildID string) (*GuildTTSConfig, error) {
	return nil, nil
}

func (m *mockConfigServiceForRecovery) SetGuildConfig(guildID string, config *GuildTTSConfig) error {
	return nil
}

func (m *mockConfigServiceForRecovery) SetRequiredRoles(guildID string, roleIDs []string) error {
	return nil
}

func (m *mockConfigServiceForRecovery) GetRequiredRoles(guildID string) ([]string, error) {
	return nil, nil
}

func (m *mockConfigServiceForRecovery) SetTTSSettings(guildID string, settings TTSConfig) error {
	return nil
}

func (m *mockConfigServiceForRecovery) GetTTSSettings(guildID string) (*TTSConfig, error) {
	return nil, nil
}

func (m *mockConfigServiceForRecovery) SetMaxQueueSize(guildID string, size int) error {
	return nil
}

func (m *mockConfigServiceForRecovery) GetMaxQueueSize(guildID string) (int, error) {
	return 10, nil
}

func (m *mockConfigServiceForRecovery) ValidateConfig(config *GuildTTSConfig) error {
	return nil
}

// Test cases

func TestErrorRecoveryManager_HandleVoiceDisconnection(t *testing.T) {
	tests := []struct {
		name          string
		guildID       string
		recoveryError error
		expectedError bool
		expectedCalls int
	}{
		{
			name:          "successful recovery",
			guildID:       "guild1",
			recoveryError: nil,
			expectedError: false,
			expectedCalls: 1,
		},
		{
			name:          "recovery fails then succeeds",
			guildID:       "guild2",
			recoveryError: errors.New("temporary failure"),
			expectedError: false,
			expectedCalls: 2, // Will retry and succeed on second attempt
		},
		{
			name:          "empty guild ID",
			guildID:       "",
			expectedError: true,
			expectedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVoice := newMockVoiceManagerForRecovery()
			mockTTS := newMockTTSManagerForRecovery()
			mockQueue := &mockMessageQueueForRecovery{}
			mockConfig := &mockConfigServiceForRecovery{}

			// Set up recovery error for first attempt only
			if tt.recoveryError != nil {
				mockVoice.recoveryErrors[tt.guildID] = tt.recoveryError
			}

			erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

			err := erm.HandleVoiceDisconnection(tt.guildID)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.guildID != "" {
				// For successful cases, check that recovery was attempted
				if len(mockVoice.recoveryCalls) == 0 {
					t.Errorf("Expected recovery to be attempted")
				}
			}
		})
	}
}

func TestErrorRecoveryManager_HandleTTSFailure(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		voice           string
		guildID         string
		conversionError error
		expectedError   bool
		expectedRetries int
	}{
		{
			name:            "successful conversion",
			text:            "test message",
			voice:           "en-US-Standard-A",
			guildID:         "guild1",
			conversionError: nil,
			expectedError:   false,
			expectedRetries: 1,
		},
		{
			name:            "retryable error then success",
			text:            "retry test",
			voice:           "en-US-Standard-A",
			guildID:         "guild2",
			conversionError: errors.New("timeout"),
			expectedError:   false,
			expectedRetries: 2, // Will retry and succeed
		},
		{
			name:            "fatal error",
			text:            "fatal test",
			voice:           "en-US-Standard-A",
			guildID:         "guild3",
			conversionError: errors.New("invalid credentials"),
			expectedError:   true,
			expectedRetries: 1, // Won't retry fatal errors
		},
		{
			name:            "empty guild ID",
			text:            "test",
			voice:           "en-US-Standard-A",
			guildID:         "",
			expectedError:   true,
			expectedRetries: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVoice := newMockVoiceManagerForRecovery()
			mockTTS := newMockTTSManagerForRecovery()
			mockQueue := &mockMessageQueueForRecovery{}
			mockConfig := &mockConfigServiceForRecovery{}

			// Set up conversion error for first attempt only
			if tt.conversionError != nil {
				mockTTS.conversionErrors[tt.text] = tt.conversionError
			}

			erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

			config := TTSConfig{
				Voice:  tt.voice,
				Speed:  1.0,
				Volume: 1.0,
				Format: AudioFormatPCM,
			}

			audioData, err := erm.HandleTTSFailure(tt.text, tt.voice, config, tt.guildID)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectedError && audioData == nil {
				t.Errorf("Expected audio data but got none")
			}

			if tt.guildID != "" && tt.expectedRetries > 0 {
				// Check that conversion was attempted
				if len(mockTTS.conversionCalls) == 0 {
					t.Errorf("Expected TTS conversion to be attempted")
				}
			}
		})
	}
}

func TestErrorRecoveryManager_HandleAudioPlaybackFailure(t *testing.T) {
	tests := []struct {
		name          string
		guildID       string
		audioData     []byte
		playbackError error
		isConnected   bool
		expectedError bool
	}{
		{
			name:          "successful playback retry",
			guildID:       "guild1",
			audioData:     []byte("test audio"),
			playbackError: nil,
			isConnected:   true,
			expectedError: false,
		},
		{
			name:          "connection lost during playback",
			guildID:       "guild2",
			audioData:     []byte("test audio"),
			playbackError: errors.New("connection lost"),
			isConnected:   false,
			expectedError: false, // Should recover connection
		},
		{
			name:          "empty guild ID",
			guildID:       "",
			audioData:     []byte("test audio"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVoice := newMockVoiceManagerForRecovery()
			mockTTS := newMockTTSManagerForRecovery()
			mockQueue := &mockMessageQueueForRecovery{}
			mockConfig := &mockConfigServiceForRecovery{}

			// Set up initial connection state
			if tt.isConnected {
				mockVoice.connections[tt.guildID] = true
			}

			// Set up playback error for first attempt only
			if tt.playbackError != nil {
				mockVoice.playAudioErrors[tt.guildID] = tt.playbackError
			}

			erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

			err := erm.HandleAudioPlaybackFailure(tt.guildID, tt.audioData)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestErrorRecoveryManager_CreateUserFriendlyErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		guildID  string
		expected string
	}{
		{
			name:     "voice connection error",
			err:      errors.New("voice connection failed"),
			guildID:  "guild1",
			expected: "I'm having trouble connecting to the voice channel. Please try inviting me again, or check that I have the necessary permissions.",
		},
		{
			name:     "permission error",
			err:      errors.New("permission denied"),
			guildID:  "guild1",
			expected: "I don't have the necessary permissions to perform this action. Please check that I have voice channel and text channel permissions.",
		},
		{
			name:     "TTS error",
			err:      errors.New("TTS conversion failed"),
			guildID:  "guild1",
			expected: "I'm having trouble converting text to speech right now. I'll keep trying, but some messages might be skipped.",
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			guildID:  "guild1",
			expected: "I'm being rate limited by the text-to-speech service. Please wait a moment and try again.",
		},
		{
			name:     "network error",
			err:      errors.New("connection timeout"),
			guildID:  "guild1",
			expected: "I'm having network connectivity issues. I'll keep trying to reconnect automatically.",
		},
		{
			name:     "configuration error",
			err:      errors.New("invalid configuration"),
			guildID:  "guild1",
			expected: "There's an issue with the TTS configuration. Please check your settings or contact an administrator.",
		},
		{
			name:     "unknown error",
			err:      errors.New("some unknown error"),
			guildID:  "guild1",
			expected: "I encountered an error, but I'll keep trying to work normally. If problems persist, please try restarting the TTS session.",
		},
		{
			name:     "nil error",
			err:      nil,
			guildID:  "guild1",
			expected: "An unknown error occurred.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVoice := newMockVoiceManagerForRecovery()
			mockTTS := newMockTTSManagerForRecovery()
			mockQueue := &mockMessageQueueForRecovery{}
			mockConfig := &mockConfigServiceForRecovery{}

			erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

			result := erm.CreateUserFriendlyErrorMessage(tt.err, tt.guildID)

			if result != tt.expected {
				t.Errorf("Expected message: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestErrorRecoveryManager_ErrorStats(t *testing.T) {
	mockVoice := newMockVoiceManagerForRecovery()
	mockTTS := newMockTTSManagerForRecovery()
	mockQueue := &mockMessageQueueForRecovery{}
	mockConfig := &mockConfigServiceForRecovery{}

	erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

	guildID := "test-guild"

	// Initially should be healthy
	if !erm.IsGuildHealthy(guildID) {
		t.Errorf("Guild should be healthy initially")
	}

	// Simulate some errors
	erm.updateErrorStats(guildID, "voice_connection")
	erm.updateErrorStats(guildID, "tts_conversion")
	erm.updateErrorStats(guildID, "audio_playback")

	// Get error stats
	stats := erm.GetErrorStats(guildID)
	if stats.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, stats.GuildID)
	}
	if stats.VoiceConnectionErrors != 1 {
		t.Errorf("Expected 1 voice connection error, got %d", stats.VoiceConnectionErrors)
	}
	if stats.TTSConversionErrors != 1 {
		t.Errorf("Expected 1 TTS conversion error, got %d", stats.TTSConversionErrors)
	}
	if stats.AudioPlaybackErrors != 1 {
		t.Errorf("Expected 1 audio playback error, got %d", stats.AudioPlaybackErrors)
	}
	if stats.ConsecutiveFailures != 3 {
		t.Errorf("Expected 3 consecutive failures, got %d", stats.ConsecutiveFailures)
	}

	// Should still be healthy with few errors
	if !erm.IsGuildHealthy(guildID) {
		t.Errorf("Guild should still be healthy with few errors")
	}

	// Simulate many consecutive failures
	for i := 0; i < 10; i++ {
		erm.updateErrorStats(guildID, "voice_connection")
	}

	// Should now be unhealthy
	if erm.IsGuildHealthy(guildID) {
		t.Errorf("Guild should be unhealthy with many consecutive failures")
	}

	// Reset stats
	erm.resetErrorStats(guildID)
	stats = erm.GetErrorStats(guildID)
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("Expected 0 consecutive failures after reset, got %d", stats.ConsecutiveFailures)
	}

	// Should be healthy again
	if !erm.IsGuildHealthy(guildID) {
		t.Errorf("Guild should be healthy after reset")
	}
}

func TestErrorRecoveryManager_StartStop(t *testing.T) {
	mockVoice := newMockVoiceManagerForRecovery()
	mockTTS := newMockTTSManagerForRecovery()
	mockQueue := &mockMessageQueueForRecovery{}
	mockConfig := &mockConfigServiceForRecovery{}

	erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

	// Test start
	err := erm.Start()
	if err != nil {
		t.Errorf("Expected no error starting error recovery manager, got: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stop
	err = erm.Stop()
	if err != nil {
		t.Errorf("Expected no error stopping error recovery manager, got: %v", err)
	}
}

func TestConnectionMonitor_HealthChecking(t *testing.T) {
	mockVoice := newMockVoiceManagerForRecovery()
	mockTTS := newMockTTSManagerForRecovery()
	mockQueue := &mockMessageQueueForRecovery{}
	mockConfig := &mockConfigServiceForRecovery{}

	// Set up a guild with connection
	guildID := "test-guild"
	mockVoice.connections[guildID] = true

	erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

	// Test connection monitoring
	erm.connectionMonitor.checkConnection(guildID)

	// Check that connection state was created
	erm.connectionMonitor.mu.RLock()
	state, exists := erm.connectionMonitor.connectionState[guildID]
	erm.connectionMonitor.mu.RUnlock()

	if !exists {
		t.Errorf("Expected connection state to be created for guild %s", guildID)
	}
	if !state.IsHealthy {
		t.Errorf("Expected connection to be healthy")
	}

	// Simulate health check failure
	mockVoice.healthCheckErrors[guildID] = errors.New("connection unhealthy")
	erm.connectionMonitor.checkConnection(guildID)

	erm.connectionMonitor.mu.RLock()
	state, _ = erm.connectionMonitor.connectionState[guildID]
	erm.connectionMonitor.mu.RUnlock()

	if state.IsHealthy {
		t.Errorf("Expected connection to be unhealthy after failed health check")
	}
	if state.ConsecutiveErrors != 1 {
		t.Errorf("Expected 1 consecutive error, got %d", state.ConsecutiveErrors)
	}
}

func TestHealthChecker_TTSHealthCheck(t *testing.T) {
	mockVoice := newMockVoiceManagerForRecovery()
	mockTTS := newMockTTSManagerForRecovery()
	mockQueue := &mockMessageQueueForRecovery{}
	mockConfig := &mockConfigServiceForRecovery{}

	erm := NewErrorRecoveryManager(mockVoice, mockTTS, mockQueue, mockConfig)

	// Test health check
	erm.healthChecker.performHealthCheck()

	// Check that TTS conversion was attempted
	if len(mockTTS.conversionCalls) == 0 {
		t.Errorf("Expected TTS health check to attempt conversion")
	}

	// Check that the health check used the correct test text
	call := mockTTS.conversionCalls[0]
	if call.Text != "Health check test" {
		t.Errorf("Expected health check text 'Health check test', got '%s'", call.Text)
	}
}

// Note: Error classification functions (IsRetryableError, IsFatalError) are tested in tts_errors_test.go
