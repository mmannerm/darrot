package tts

import (
	"fmt"
	"log"
	"math"
)

// MockTTSManager provides a simple TTS implementation for testing
type MockTTSManager struct {
	messageQueue MessageQueue
}

// NewMockTTSManager creates a new mock TTS manager
func NewMockTTSManager(messageQueue MessageQueue) (TTSManager, error) {
	return &MockTTSManager{
		messageQueue: messageQueue,
	}, nil
}

// ConvertToSpeech generates a simple audio beep instead of real TTS
func (m *MockTTSManager) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	log.Printf("Mock TTS: Converting text to speech: %s", text)

	// Generate a simple audio beep pattern
	// This creates a basic tone that Discord can play
	audioData := m.generateSimpleBeep(len(text))

	log.Printf("Mock TTS: Generated %d bytes of audio for text: %s", len(audioData), text)
	return audioData, nil
}

// generateSimpleBeep creates a basic audio beep in Discord-compatible format
func (m *MockTTSManager) generateSimpleBeep(textLength int) []byte {
	// Discord expects small audio frames (20ms at 48kHz = 960 samples)
	// Create a very short beep to avoid UDP packet size issues
	frameSize := 960 * 2 // 960 samples * 2 bytes per sample
	numFrames := 10      // Very short audio (200ms total)

	audioData := make([]byte, frameSize*numFrames)

	// Generate a simple beep pattern
	for frame := 0; frame < numFrames; frame++ {
		for sample := 0; sample < 960; sample++ {
			// Create a simple tone
			t := float64(frame*960+sample) / 48000.0
			amplitude := 0.1 // Very quiet
			frequency := 440.0

			value := int16(amplitude * 32767 * math.Sin(2*math.Pi*frequency*t))

			offset := frame*frameSize + sample*2
			audioData[offset] = byte(value & 0xFF)
			audioData[offset+1] = byte((value >> 8) & 0xFF)
		}
	}

	return audioData
}

// GetSupportedVoices returns mock voice options (required by TTSManager interface)
func (m *MockTTSManager) GetSupportedVoices() []Voice {
	return []Voice{
		{
			Name:     "mock-voice",
			Language: "en-US",
			Gender:   "neutral",
		},
	}
}

// ProcessMessageQueue processes queued messages (required by TTSManager interface)
func (m *MockTTSManager) ProcessMessageQueue(guildID string) error {
	// This is handled by the TTS processor, so we can just return nil
	return nil
}

// SetVoiceConfig sets voice configuration (required by TTSManager interface)
func (m *MockTTSManager) SetVoiceConfig(guildID string, config TTSConfig) error {
	log.Printf("Mock TTS: Setting voice config for guild %s: %+v", guildID, config)
	return nil
}

// ValidateVoice validates mock voice settings
func (m *MockTTSManager) ValidateVoice(voice string) error {
	if voice == "" || voice == "mock-voice" {
		return nil
	}
	return fmt.Errorf("mock TTS only supports 'mock-voice'")
}

// GetSupportedFormats returns supported audio formats
func (m *MockTTSManager) GetSupportedFormats() []AudioFormat {
	return []AudioFormat{AudioFormatPCM}
}

// Cleanup performs any necessary cleanup
func (m *MockTTSManager) Cleanup() error {
	log.Println("Mock TTS manager cleanup complete")
	return nil
}
