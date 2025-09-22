package tts

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/jonas747/dca"
)

// GoogleTTSManager implements TTSManager using Google Cloud Text-to-Speech
type GoogleTTSManager struct {
	client        *texttospeech.Client
	messageQueue  MessageQueue
	voiceConfigs  map[string]TTSConfig
	errorRecovery *ErrorRecovery
	healthChecker *TTSHealthChecker
	mu            sync.RWMutex
}

// NewGoogleTTSManager creates a new Google TTS manager instance
func NewGoogleTTSManager(messageQueue MessageQueue) (*GoogleTTSManager, error) {
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %w", err)
	}

	manager := &GoogleTTSManager{
		client:        client,
		messageQueue:  messageQueue,
		voiceConfigs:  make(map[string]TTSConfig),
		errorRecovery: NewErrorRecovery(),
	}

	// Initialize health checker
	manager.healthChecker = NewTTSHealthChecker(manager)

	return manager, nil
}

// ConvertToSpeech converts text to speech using Google Cloud TTS
func (g *GoogleTTSManager) ConvertToSpeech(text, voice string, config TTSConfig) ([]byte, error) {
	if text == "" {
		return nil, ErrEmptyText
	}

	// Check text length
	if len(text) > MaxMessageLength {
		return nil, ErrTextTooLong
	}

	// Check if we have a valid client
	if g.client == nil {
		return nil, ErrTTSEngineUnavailable
	}

	// Use provided voice or fall back to config voice or default
	selectedVoice := voice
	if selectedVoice == "" {
		selectedVoice = config.Voice
	}
	if selectedVoice == "" {
		selectedVoice = DefaultVoice
	}

	// Validate and set speed
	speed := config.Speed
	if speed < MinTTSSpeed || speed > MaxTTSSpeed {
		speed = DefaultTTSSpeed
	}

	// Validate and set volume
	volume := config.Volume
	if volume < MinTTSVolume || volume > MaxTTSVolume {
		volume = DefaultTTSVolume
	}

	// Parse voice ID to extract language and name
	languageCode, voiceName := parseVoiceID(selectedVoice)

	// Create the TTS request
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: text,
			},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: languageCode,
			Name:         voiceName,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_LINEAR16,
			SpeakingRate:    float64(speed),
			VolumeGainDb:    volumeToDb(volume),
			SampleRateHertz: 48000, // Discord's preferred sample rate
		},
	}

	ctx := context.Background()
	resp, err := g.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		// Check if this is a retryable error
		if IsRetryableError(err) {
			return nil, fmt.Errorf("TTS synthesis failed (retryable): %w", err)
		}
		if IsFatalError(err) {
			return nil, fmt.Errorf("TTS synthesis failed (fatal): %w", err)
		}
		return nil, fmt.Errorf("TTS synthesis failed: %w", err)
	}

	// Convert audio to Discord-compatible format
	audioData, err := g.convertToDiscordFormat(resp.AudioContent, config.Format)
	if err != nil {
		return nil, fmt.Errorf("audio format conversion failed: %w", err)
	}

	return audioData, nil
}

// ProcessMessageQueue processes queued messages for a guild
func (g *GoogleTTSManager) ProcessMessageQueue(guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	// Get guild TTS configuration
	config := g.getVoiceConfig(guildID)

	for {
		message, err := g.messageQueue.Dequeue(guildID)
		if err != nil {
			// No more messages in queue
			break
		}

		if message == nil {
			break
		}

		// Prepare message text with author name
		messageText := fmt.Sprintf("%s says: %s", message.Username, message.Content)

		// Truncate message if too long
		if len(messageText) > MaxMessageLength {
			messageText = messageText[:MaxMessageLength-3] + "..."
		}

		// Check if we have a valid client before attempting conversion
		if g.client == nil {
			log.Printf("TTS client not available for guild %s, skipping message", guildID)
			continue // Skip this message and continue with next
		}

		// Convert to speech with error recovery
		audioData, err := g.ConvertToSpeech(messageText, "", config)
		if err != nil {
			log.Printf("Initial TTS conversion failed for guild %s: %v", guildID, err)

			// Try error recovery
			audioData, err = g.errorRecovery.HandleTTSFailure(g, messageText, "", config, guildID)
			if err != nil {
				log.Printf("TTS conversion failed after recovery attempts for guild %s: %v", guildID, err)
				continue // Skip this message and continue with next
			}
		}

		// TODO: This would typically send audio to VoiceManager for playback
		// For now, we'll just log that the conversion was successful
		log.Printf("Successfully converted message to speech for guild %s: %d bytes", guildID, len(audioData))
	}

	return nil
}

// SetVoiceConfig sets the TTS configuration for a guild
func (g *GoogleTTSManager) SetVoiceConfig(guildID string, config TTSConfig) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	// Validate configuration
	if err := g.validateTTSConfig(config); err != nil {
		return fmt.Errorf("invalid TTS config: %w", err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.voiceConfigs[guildID] = config
	return nil
}

// GetSupportedVoices returns a list of supported TTS voices
func (g *GoogleTTSManager) GetSupportedVoices() []Voice {
	// Return default voices if no client is available
	if g.client == nil {
		return getDefaultVoices()
	}

	ctx := context.Background()
	req := &texttospeechpb.ListVoicesRequest{}

	resp, err := g.client.ListVoices(ctx, req)
	if err != nil {
		log.Printf("Failed to list voices: %v", err)
		return getDefaultVoices()
	}

	voices := make([]Voice, 0, len(resp.Voices))
	for _, voice := range resp.Voices {
		for _, languageCode := range voice.LanguageCodes {
			voices = append(voices, Voice{
				ID:       voice.Name,
				Name:     voice.Name,
				Language: languageCode,
				Gender:   voice.SsmlGender.String(),
			})
		}
	}

	return voices
}

// StartHealthCheck starts the health monitoring for the TTS engine
func (g *GoogleTTSManager) StartHealthCheck() {
	if g.healthChecker != nil {
		g.healthChecker.StartHealthCheck()
	}
}

// Close closes the TTS client
func (g *GoogleTTSManager) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// Helper methods

// getVoiceConfig gets the TTS configuration for a guild
func (g *GoogleTTSManager) getVoiceConfig(guildID string) TTSConfig {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if config, exists := g.voiceConfigs[guildID]; exists {
		return config
	}

	// Return default configuration
	return TTSConfig{
		Voice:  DefaultVoice,
		Speed:  DefaultTTSSpeed,
		Volume: DefaultTTSVolume,
		Format: AudioFormatDCA,
	}
}

// validateTTSConfig validates TTS configuration parameters
func (g *GoogleTTSManager) validateTTSConfig(config TTSConfig) error {
	if config.Speed < MinTTSSpeed || config.Speed > MaxTTSSpeed {
		return fmt.Errorf("speed must be between %f and %f", MinTTSSpeed, MaxTTSSpeed)
	}

	if config.Volume < MinTTSVolume || config.Volume > MaxTTSVolume {
		return fmt.Errorf("volume must be between %f and %f", MinTTSVolume, MaxTTSVolume)
	}

	if config.Format != AudioFormatOpus && config.Format != AudioFormatDCA && config.Format != AudioFormatPCM {
		return fmt.Errorf("unsupported audio format: %s", config.Format)
	}

	return nil
}

// convertToDiscordFormat converts audio to Discord-compatible format
func (g *GoogleTTSManager) convertToDiscordFormat(audioData []byte, format AudioFormat) ([]byte, error) {
	switch format {
	case AudioFormatDCA:
		return g.convertToDCA(audioData)
	case AudioFormatOpus:
		return g.convertToOpus(audioData)
	case AudioFormatPCM:
		return audioData, nil // Already PCM from Google TTS
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", format)
	}
}

// convertToDCA converts PCM audio to DCA format for Discord
func (g *GoogleTTSManager) convertToDCA(pcmData []byte) ([]byte, error) {
	// Create a DCA encoder
	options := &dca.EncodeOptions{
		Volume:           256,
		Channels:         2,
		FrameRate:        48000,
		FrameDuration:    20,
		Bitrate:          64,
		Application:      dca.AudioApplicationAudio,
		CompressionLevel: 10,
		PacketLoss:       1,
		BufferedFrames:   100,
		VBR:              true,
	}

	// Create a reader from PCM data
	reader := &bytesReader{data: pcmData}

	// Encode to DCA
	encoder, err := dca.EncodeMem(reader, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create DCA encoder: %w", err)
	}
	defer encoder.Cleanup()

	// Read all encoded data
	var dcaData []byte
	for {
		frame, err := encoder.OpusFrame()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to encode DCA frame: %w", err)
		}
		dcaData = append(dcaData, frame...)
	}

	return dcaData, nil
}

// convertToOpus converts PCM audio to Opus format
func (g *GoogleTTSManager) convertToOpus(pcmData []byte) ([]byte, error) {
	// For now, use DCA conversion as it produces Opus-compatible output
	return g.convertToDCA(pcmData)
}

// parseVoiceID parses a voice ID to extract language code and voice name
func parseVoiceID(voiceID string) (languageCode, voiceName string) {
	// Default values
	languageCode = "en-US"
	voiceName = voiceID

	// Parse voice ID format: "en-US-Standard-A" or "en-US-Wavenet-A"
	if len(voiceID) >= 5 {
		if voiceID[2] == '-' && voiceID[5] == '-' {
			languageCode = voiceID[:5] // Extract "en-US"
		}
	}

	return languageCode, voiceName
}

// volumeToDb converts linear volume (0.0-2.0) to decibels
func volumeToDb(volume float32) float64 {
	if volume <= 0 {
		return -96.0 // Minimum volume
	}
	if volume >= 2.0 {
		return 6.0 // Maximum volume (+6dB)
	}
	if volume == 1.0 {
		return 0.0 // Unity gain
	}

	// Convert linear to dB: 20 * log10(volume)
	// Simplified approximation for common values
	if volume < 1.0 {
		return -20.0 * (1.0 - float64(volume)) // Approximate attenuation
	}
	return 6.0 * (float64(volume) - 1.0) // Approximate gain
}

// getDefaultVoices returns a list of default voices when API call fails
func getDefaultVoices() []Voice {
	return []Voice{
		{ID: "en-US-Standard-A", Name: "en-US-Standard-A", Language: "en-US", Gender: "FEMALE"},
		{ID: "en-US-Standard-B", Name: "en-US-Standard-B", Language: "en-US", Gender: "MALE"},
		{ID: "en-US-Standard-C", Name: "en-US-Standard-C", Language: "en-US", Gender: "FEMALE"},
		{ID: "en-US-Standard-D", Name: "en-US-Standard-D", Language: "en-US", Gender: "MALE"},
		{ID: "en-US-Wavenet-A", Name: "en-US-Wavenet-A", Language: "en-US", Gender: "MALE"},
		{ID: "en-US-Wavenet-B", Name: "en-US-Wavenet-B", Language: "en-US", Gender: "MALE"},
		{ID: "en-US-Wavenet-C", Name: "en-US-Wavenet-C", Language: "en-US", Gender: "FEMALE"},
		{ID: "en-US-Wavenet-D", Name: "en-US-Wavenet-D", Language: "en-US", Gender: "MALE"},
	}
}

// bytesReader implements io.Reader for byte slices
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
