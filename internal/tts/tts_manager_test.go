package tts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGoogleTTSManager(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful creation with mock queue",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockMessageQueue{}

			// Note: This test will fail in CI without Google Cloud credentials
			// In a real environment, you'd mock the Google TTS client
			manager, err := NewGoogleTTSManager(mockQueue, "")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				// Skip this test if we don't have credentials
				if err != nil && err.Error() == "failed to create TTS client: google: could not find default credentials. See https://cloud.google.com/docs/authentication/external/set-up-adc for more information" {
					t.Skip("Skipping test - no Google Cloud credentials available")
				}

				assert.NoError(t, err)
				assert.NotNil(t, manager)

				if manager != nil {
					manager.Close()
				}
			}
		})
	}
}

func TestGoogleTTSManager_SetVoiceConfig(t *testing.T) {
	mockQueue := &MockMessageQueue{}

	// Create a manager without Google client for testing config operations
	manager := &GoogleTTSManager{
		client:       nil, // We'll test without actual Google client
		messageQueue: mockQueue,
		voiceConfigs: make(map[string]TTSConfig),
	}

	tests := []struct {
		name    string
		guildID string
		config  TTSConfig
		wantErr bool
	}{
		{
			name:    "valid configuration",
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
			name:    "empty guild ID",
			guildID: "",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name:    "invalid speed - too low",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  0.1,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name:    "invalid speed - too high",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  5.0,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name:    "invalid volume - too low",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: -0.1,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name:    "invalid volume - too high",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 3.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name:    "invalid audio format",
			guildID: "guild123",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: "invalid",
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

func TestGoogleTTSManager_ProcessMessageQueue(t *testing.T) {
	mockQueue := &MockMessageQueue{}

	manager := &GoogleTTSManager{
		client:       nil,
		messageQueue: mockQueue,
		voiceConfigs: make(map[string]TTSConfig),
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
		{
			name:    "queue with messages",
			guildID: "guild123",
			messages: []*QueuedMessage{
				{
					ID:        "msg1",
					GuildID:   "guild123",
					ChannelID: "channel123",
					UserID:    "user123",
					Username:  "TestUser",
					Content:   "Hello world",
					Timestamp: time.Now(),
				},
				{
					ID:        "msg2",
					GuildID:   "guild123",
					ChannelID: "channel123",
					UserID:    "user456",
					Username:  "AnotherUser",
					Content:   "How are you?",
					Timestamp: time.Now(),
				},
			},
			wantErr: false,
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
				// For tests with messages but no Google client, we expect no error
				// but the messages will be skipped due to TTS conversion failures
				assert.NoError(t, err)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestGoogleTTSManager_GetVoiceConfig(t *testing.T) {
	mockQueue := &MockMessageQueue{}

	manager := &GoogleTTSManager{
		client:       nil,
		messageQueue: mockQueue,
		voiceConfigs: make(map[string]TTSConfig),
	}

	// Set a custom config for one guild
	customConfig := TTSConfig{
		Voice:  "en-US-Wavenet-A",
		Speed:  1.5,
		Volume: 0.8,
		Format: AudioFormatOpus,
	}
	manager.voiceConfigs["guild123"] = customConfig

	tests := []struct {
		name           string
		guildID        string
		expectedConfig TTSConfig
	}{
		{
			name:           "guild with custom config",
			guildID:        "guild123",
			expectedConfig: customConfig,
		},
		{
			name:    "guild with default config",
			guildID: "guild456",
			expectedConfig: TTSConfig{
				Voice:  DefaultVoice,
				Speed:  DefaultTTSSpeed,
				Volume: DefaultTTSVolume,
				Format: AudioFormatDCA,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := manager.getVoiceConfig(tt.guildID)
			assert.Equal(t, tt.expectedConfig, config)
		})
	}
}

func TestValidateTTSConfig(t *testing.T) {
	manager := &GoogleTTSManager{}

	tests := []struct {
		name    string
		config  TTSConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: false,
		},
		{
			name: "speed too low",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  0.1,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name: "speed too high",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  5.0,
				Volume: 1.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name: "volume too low",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: -0.1,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name: "volume too high",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 3.0,
				Format: AudioFormatDCA,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: TTSConfig{
				Voice:  "en-US-Standard-A",
				Speed:  1.0,
				Volume: 1.0,
				Format: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateTTSConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseVoiceID(t *testing.T) {
	tests := []struct {
		name              string
		voiceID           string
		expectedLangCode  string
		expectedVoiceName string
	}{
		{
			name:              "standard voice format",
			voiceID:           "en-US-Standard-A",
			expectedLangCode:  "en-US",
			expectedVoiceName: "en-US-Standard-A",
		},
		{
			name:              "wavenet voice format",
			voiceID:           "en-GB-Wavenet-B",
			expectedLangCode:  "en-GB",
			expectedVoiceName: "en-GB-Wavenet-B",
		},
		{
			name:              "short voice ID",
			voiceID:           "test",
			expectedLangCode:  "en-US",
			expectedVoiceName: "test",
		},
		{
			name:              "empty voice ID",
			voiceID:           "",
			expectedLangCode:  "en-US",
			expectedVoiceName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			langCode, voiceName := parseVoiceID(tt.voiceID)
			assert.Equal(t, tt.expectedLangCode, langCode)
			assert.Equal(t, tt.expectedVoiceName, voiceName)
		})
	}
}

func TestVolumeToDb(t *testing.T) {
	tests := []struct {
		name     string
		volume   float32
		expected float64
	}{
		{
			name:     "zero volume",
			volume:   0.0,
			expected: -96.0,
		},
		{
			name:     "unity volume",
			volume:   1.0,
			expected: 0.0,
		},
		{
			name:     "maximum volume",
			volume:   2.0,
			expected: 6.0,
		},
		{
			name:     "half volume",
			volume:   0.5,
			expected: -10.0,
		},
		{
			name:     "1.5x volume",
			volume:   1.5,
			expected: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := volumeToDb(tt.volume)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultVoices(t *testing.T) {
	voices := getDefaultVoices()

	assert.NotEmpty(t, voices)
	assert.GreaterOrEqual(t, len(voices), 8)

	// Check that all voices have required fields
	for _, voice := range voices {
		assert.NotEmpty(t, voice.ID)
		assert.NotEmpty(t, voice.Name)
		assert.NotEmpty(t, voice.Language)
		assert.NotEmpty(t, voice.Gender)
	}

	// Check for specific expected voices
	voiceIDs := make(map[string]bool)
	for _, voice := range voices {
		voiceIDs[voice.ID] = true
	}

	assert.True(t, voiceIDs["en-US-Standard-A"])
	assert.True(t, voiceIDs["en-US-Wavenet-A"])
}

func TestBytesReader(t *testing.T) {
	data := []byte("Hello, World!")
	reader := &bytesReader{data: data}

	// Test reading all data
	buffer := make([]byte, len(data))
	n, err := reader.Read(buffer)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buffer)

	// Test reading beyond end
	_, err = reader.Read(buffer)
	assert.Error(t, err)
	assert.Equal(t, "EOF", err.Error())
}

func TestBytesReader_PartialRead(t *testing.T) {
	data := []byte("Hello, World!")
	reader := &bytesReader{data: data}

	// Read first 5 bytes
	buffer := make([]byte, 5)
	n, err := reader.Read(buffer)

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("Hello"), buffer)

	// Read remaining bytes
	buffer = make([]byte, 10)
	n, err = reader.Read(buffer)

	assert.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, []byte(", World!"), buffer[:n])
}

// Integration test for ConvertToSpeech (requires Google Cloud credentials)
func TestGoogleTTSManager_ConvertToSpeech_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockQueue := &MockMessageQueue{}
	manager, err := NewGoogleTTSManager(mockQueue, "")
	if err != nil {
		t.Skip("Skipping integration test - no Google Cloud credentials available")
	}
	defer manager.Close()

	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatPCM,
	}

	tests := []struct {
		name    string
		text    string
		voice   string
		wantErr bool
	}{
		{
			name:    "simple text conversion",
			text:    "Hello, this is a test.",
			voice:   "",
			wantErr: false,
		},
		{
			name:    "empty text",
			text:    "",
			voice:   "",
			wantErr: true,
		},
		{
			name:    "custom voice",
			text:    "Testing custom voice",
			voice:   "en-US-Standard-B",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audioData, err := manager.ConvertToSpeech(tt.text, tt.voice, config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, audioData)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, audioData)
				assert.Greater(t, len(audioData), 0)
			}
		})
	}
}
