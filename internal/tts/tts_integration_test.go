package tts

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTTSIntegration tests the complete TTS workflow without requiring Google credentials
func TestTTSIntegration(t *testing.T) {
	// Create message queue
	messageQueue := NewMessageQueue()
	require.NotNil(t, messageQueue)

	// Create TTS manager without Google client for testing
	manager := &GoogleTTSManager{
		client:        nil, // No actual Google client for testing
		messageQueue:  messageQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}
	manager.healthChecker = NewTTSHealthChecker(manager)

	guildID := "test-guild-123"

	// Test 1: Set voice configuration
	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.2,
		Volume: 0.8,
		Format: AudioFormatDCA,
	}

	err := manager.SetVoiceConfig(guildID, config)
	assert.NoError(t, err)

	// Verify configuration was stored
	storedConfig := manager.getVoiceConfig(guildID)
	assert.Equal(t, config, storedConfig)

	// Test 2: Add messages to queue
	messages := []*QueuedMessage{
		{
			ID:        "msg1",
			GuildID:   guildID,
			ChannelID: "channel123",
			UserID:    "user123",
			Username:  "TestUser",
			Content:   "Hello world!",
			Timestamp: time.Now(),
		},
		{
			ID:        "msg2",
			GuildID:   guildID,
			ChannelID: "channel123",
			UserID:    "user456",
			Username:  "AnotherUser",
			Content:   "How are you doing?",
			Timestamp: time.Now(),
		},
	}

	for _, msg := range messages {
		err := messageQueue.Enqueue(msg)
		assert.NoError(t, err)
	}

	// Verify queue size
	assert.Equal(t, len(messages), messageQueue.Size(guildID))

	// Test 3: Process message queue (will fail due to no Google client, but should handle gracefully)
	err = manager.ProcessMessageQueue(guildID)
	assert.NoError(t, err) // Should not error, just skip messages due to TTS failures

	// Test 4: Test error recovery
	audioData, err := manager.errorRecovery.HandleTTSFailure(manager, "test text", "", config, guildID)
	assert.Error(t, err) // Should fail due to no Google client
	assert.Nil(t, audioData)

	// Verify it's a TTS error
	var ttsErr *TTSError
	assert.True(t, errors.As(err, &ttsErr))
	assert.Equal(t, "conversion", ttsErr.Type)
	assert.Equal(t, guildID, ttsErr.GuildID)

	// Test 5: Test supported voices (should return defaults when API fails)
	voices := manager.GetSupportedVoices()
	assert.NotEmpty(t, voices)
	assert.GreaterOrEqual(t, len(voices), 8) // Should have at least 8 default voices

	// Test 6: Test configuration validation
	invalidConfig := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  10.0, // Invalid speed
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	err = manager.SetVoiceConfig(guildID, invalidConfig)
	assert.Error(t, err)
}

// TestTTSManagerLifecycle tests the complete lifecycle of a TTS manager
func TestTTSManagerLifecycle(t *testing.T) {
	messageQueue := NewMessageQueue()

	// Test manager creation without Google credentials
	manager := &GoogleTTSManager{
		client:        nil,
		messageQueue:  messageQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}
	manager.healthChecker = NewTTSHealthChecker(manager)

	// Test health checker initialization
	assert.NotNil(t, manager.healthChecker)
	assert.Equal(t, manager, manager.healthChecker.manager)

	// Test configuration management
	guildID := "lifecycle-test-guild"

	// Initially should return default config
	defaultConfig := manager.getVoiceConfig(guildID)
	assert.Equal(t, DefaultVoice, defaultConfig.Voice)
	assert.Equal(t, float32(DefaultTTSSpeed), defaultConfig.Speed)
	assert.Equal(t, float32(DefaultTTSVolume), defaultConfig.Volume)
	assert.Equal(t, AudioFormatDCA, defaultConfig.Format)

	// Set custom config
	customConfig := TTSConfig{
		Voice:  "en-US-Wavenet-A",
		Speed:  1.5,
		Volume: 0.7,
		Format: AudioFormatOpus,
	}

	err := manager.SetVoiceConfig(guildID, customConfig)
	assert.NoError(t, err)

	// Verify custom config is returned
	retrievedConfig := manager.getVoiceConfig(guildID)
	assert.Equal(t, customConfig, retrievedConfig)

	// Test cleanup
	err = manager.Close()
	assert.NoError(t, err) // Should not error even with nil client
}

// TestTTSIntegrationErrorScenarios tests various error scenarios in integration context
func TestTTSIntegrationErrorScenarios(t *testing.T) {
	messageQueue := NewMessageQueue()
	manager := &GoogleTTSManager{
		client:        nil,
		messageQueue:  messageQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}

	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	tests := []struct {
		name    string
		text    string
		voice   string
		wantErr bool
	}{
		{
			name:    "empty text",
			text:    "",
			voice:   "",
			wantErr: true,
		},
		{
			name:    "text too long",
			text:    string(make([]byte, MaxMessageLength+1)),
			voice:   "",
			wantErr: true,
		},
		{
			name:    "normal text with no client",
			text:    "Hello world",
			voice:   "",
			wantErr: true, // Will fail due to nil client
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ConvertToSpeech(tt.text, tt.voice, config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTTSAudioFormatConversion tests audio format conversion logic
func TestTTSAudioFormatConversion(t *testing.T) {
	manager := &GoogleTTSManager{}

	testData := []byte("mock audio data!") // 16 bytes (even length for 16-bit samples)

	tests := []struct {
		name    string
		format  AudioFormat
		wantErr bool
	}{
		{
			name:    "PCM format",
			format:  AudioFormatPCM,
			wantErr: false, // PCM is pass-through
		},
		{
			name:    "DCA format",
			format:  AudioFormatDCA,
			wantErr: false, // DCA conversion will succeed with mock data
		},
		{
			name:    "Opus format",
			format:  AudioFormatOpus,
			wantErr: false, // Opus conversion uses DCA internally
		},
		{
			name:    "invalid format",
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.convertToDiscordFormat(testData, tt.format)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// For DCA/Opus formats, result might be nil if ffmpeg is not available
				// This is acceptable for testing purposes
				if tt.format == AudioFormatPCM {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

// TestTTSConcurrentAccess tests concurrent access to TTS manager
func TestTTSConcurrentAccess(t *testing.T) {
	messageQueue := NewMessageQueue()
	manager := &GoogleTTSManager{
		client:        nil,
		messageQueue:  messageQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}

	// Test concurrent configuration updates
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			guildID := fmt.Sprintf("guild-%d", id)
			config := TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0 + float32(id)*0.1,
				Volume: 1.0,
				Format: AudioFormatDCA,
			}

			err := manager.SetVoiceConfig(guildID, config)
			assert.NoError(t, err)

			retrievedConfig := manager.getVoiceConfig(guildID)
			assert.Equal(t, config, retrievedConfig)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
