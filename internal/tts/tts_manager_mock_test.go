package tts

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTTSManager implements TTSManager for testing
type MockTTSManager struct {
	mock.Mock
}

func (m *MockTTSManager) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	args := m.Called(text, voice, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTTSManager) ProcessMessageQueue(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockTTSManager) SetVoiceConfig(guildID string, config TTSConfig) error {
	args := m.Called(guildID, config)
	return args.Error(0)
}

func (m *MockTTSManager) GetSupportedVoices() []Voice {
	args := m.Called()
	return args.Get(0).([]Voice)
}

func TestMockTTSManager_ConvertToSpeech(t *testing.T) {
	mockManager := &MockTTSManager{}

	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	expectedAudio := []byte("mock audio data")
	mockManager.On("ConvertToSpeech", "Hello world", "", config).Return(expectedAudio, nil)

	result, err := mockManager.ConvertToSpeech("Hello world", "", config)

	assert.NoError(t, err)
	assert.Equal(t, expectedAudio, result)
	mockManager.AssertExpectations(t)
}

func TestMockTTSManager_ProcessMessageQueue(t *testing.T) {
	mockManager := &MockTTSManager{}

	mockManager.On("ProcessMessageQueue", "guild123").Return(nil)

	err := mockManager.ProcessMessageQueue("guild123")

	assert.NoError(t, err)
	mockManager.AssertExpectations(t)
}

func TestMockTTSManager_SetVoiceConfig(t *testing.T) {
	mockManager := &MockTTSManager{}

	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	mockManager.On("SetVoiceConfig", "guild123", config).Return(nil)

	err := mockManager.SetVoiceConfig("guild123", config)

	assert.NoError(t, err)
	mockManager.AssertExpectations(t)
}

func TestMockTTSManager_GetSupportedVoices(t *testing.T) {
	mockManager := &MockTTSManager{}

	expectedVoices := []Voice{
		{ID: "en-US-Standard-A", Name: "en-US-Standard-A", Language: "en-US", Gender: "FEMALE"},
		{ID: "en-US-Standard-B", Name: "en-US-Standard-B", Language: "en-US", Gender: "MALE"},
	}

	mockManager.On("GetSupportedVoices").Return(expectedVoices)

	voices := mockManager.GetSupportedVoices()

	assert.Equal(t, expectedVoices, voices)
	mockManager.AssertExpectations(t)
}

// Test the actual TTS manager functionality without requiring Google credentials
func TestTTSManager_ProcessMessageQueue_WithMockQueue(t *testing.T) {
	mockQueue := &MockMessageQueue{}

	// Create a manager with nil client for testing logic without Google API
	manager := &GoogleTTSManager{
		client:        nil, // No actual Google client
		messageQueue:  mockQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}

	tests := []struct {
		name     string
		guildID  string
		messages []*QueuedMessage
		wantErr  bool
	}{
		{
			name:    "empty guild ID",
			guildID: "",
			wantErr: true,
		},
		{
			name:     "empty queue",
			guildID:  "guild123",
			messages: []*QueuedMessage{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			if tt.guildID != "" {
				// Mock dequeue calls
				for _, msg := range tt.messages {
					mockQueue.On("Dequeue", tt.guildID).Return(msg, nil).Once()
				}
				// Final dequeue returns nil to end processing
				mockQueue.On("Dequeue", tt.guildID).Return(nil, nil).Once()
			}

			err := manager.ProcessMessageQueue(tt.guildID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

// Test error recovery functionality
func TestErrorRecovery_HandleTTSFailure_WithMockManager(t *testing.T) {
	mockQueue := &MockMessageQueue{}
	recovery := NewErrorRecovery()

	// Create a manager that will always fail TTS conversion
	manager := &GoogleTTSManager{
		client:        nil,
		messageQueue:  mockQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: recovery,
	}

	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	// This should fail because we don't have a real Google client
	audioData, err := recovery.HandleTTSFailure(manager, "test text", "", config, "guild123")

	assert.Error(t, err)
	assert.Nil(t, audioData)

	// Check that it's a TTS error
	var ttsErr *TTSError
	assert.True(t, errors.As(err, &ttsErr))
	assert.Equal(t, "conversion", ttsErr.Type)
	assert.Equal(t, "guild123", ttsErr.GuildID)
}

// Test configuration validation
func TestTTSManager_ConfigValidation(t *testing.T) {
	mockQueue := &MockMessageQueue{}

	manager := &GoogleTTSManager{
		client:        nil,
		messageQueue:  mockQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}

	tests := []struct {
		name    string
		guildID string
		config  TTSConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: false,
		},
		{
			name:    "invalid speed",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  10.0, // Too high
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetVoiceConfig(tt.guildID, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify config was stored
				storedConfig := manager.getVoiceConfig(tt.guildID)
				assert.Equal(t, tt.config, storedConfig)
			}
		})
	}
}

// Test audio format conversion logic (without actual conversion)
func TestTTSManager_AudioFormatValidation(t *testing.T) {
	manager := &GoogleTTSManager{}

	tests := []struct {
		name   string
		format AudioFormat
		valid  bool
	}{
		{
			name:   "DCA format",
			format: AudioFormatDCA,
			valid:  true,
		},
		{
			name:   "Opus format",
			format: AudioFormatOpus,
			valid:  true,
		},
		{
			name:   "PCM format",
			format: AudioFormatPCM,
			valid:  true,
		},
		{
			name:   "invalid format",
			format: "invalid",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: tt.format,
			}

			err := manager.validateTTSConfig(config)

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
