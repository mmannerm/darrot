package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"
	"gopkg.in/hraban/opus.v2"
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
func NewGoogleTTSManager(messageQueue MessageQueue, credentialsPath string) (*GoogleTTSManager, error) {
	ctx := context.Background()

	var client *texttospeech.Client
	var err error

	if credentialsPath != "" {
		// Use service account credentials file
		client, err = texttospeech.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
	} else {
		// Use default credentials (Application Default Credentials)
		client, err = texttospeech.NewClient(ctx)
	}

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
			SampleRateHertz: 24000, // Use 24kHz (widely supported) then resample to 48kHz
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

	log.Printf("[DEBUG] Google TTS returned %d bytes of audio data for text: %s", len(resp.AudioContent), text)
	log.Printf("[DEBUG] TTS Request config - SampleRate: %d, Channels: %d, Encoding: %s",
		req.AudioConfig.SampleRateHertz,
		2, // We set channels to 2 in the config
		req.AudioConfig.AudioEncoding.String())

	// Debug: Check what Google TTS actually returned
	log.Printf("[DEBUG] TTS Response - AudioContent length: %d bytes", len(resp.AudioContent))
	if len(resp.AudioContent) >= 44 {
		// Check if it's a WAV file (has WAV header)
		header := resp.AudioContent[:44]
		if string(header[0:4]) == "RIFF" && string(header[8:12]) == "WAVE" {
			log.Printf("[DEBUG] Response contains WAV header")
			// Extract sample rate from WAV header (bytes 24-27, little-endian)
			actualSampleRate := uint32(header[24]) | uint32(header[25])<<8 | uint32(header[26])<<16 | uint32(header[27])<<24
			// Extract channels from WAV header (bytes 22-23, little-endian)
			actualChannels := uint16(header[22]) | uint16(header[23])<<8
			log.Printf("[DEBUG] WAV header indicates: %d Hz, %d channels", actualSampleRate, actualChannels)
		} else {
			log.Printf("[DEBUG] Response is raw PCM (no WAV header)")
			log.Printf("[DEBUG] First 16 bytes: %v", resp.AudioContent[:16])
		}
	}

	// Determine actual audio format from Google TTS response
	actualSampleRate := 24000 // Our request
	actualChannels := 1       // Google TTS typically returns mono for LINEAR16
	audioContent := resp.AudioContent

	// Skip WAV header if present
	if len(audioContent) >= 44 && string(audioContent[0:4]) == "RIFF" {
		log.Printf("[DEBUG] Skipping WAV header (44 bytes)")
		audioContent = audioContent[44:] // Skip WAV header
		// Extract actual format from WAV header
		header := resp.AudioContent[:44]
		actualSampleRate = int(uint32(header[24]) | uint32(header[25])<<8 | uint32(header[26])<<16 | uint32(header[27])<<24)
		actualChannels = int(uint16(header[22]) | uint16(header[23])<<8)
		log.Printf("[DEBUG] WAV header format: %d Hz, %d channels", actualSampleRate, actualChannels)
	}

	// Convert mono to stereo if needed, then resample to 48kHz stereo
	processedAudio := g.processAudioForDiscord(audioContent, actualSampleRate, actualChannels)
	log.Printf("[DEBUG] Processed audio: %d bytes -> %d bytes (%dHz %dch -> 48kHz 2ch)",
		len(audioContent), len(processedAudio), actualSampleRate, actualChannels)

	// Convert audio to Discord-compatible format
	audioData, err := g.convertToDiscordFormat(processedAudio, config.Format)
	if err != nil {
		return nil, fmt.Errorf("audio format conversion failed: %w", err)
	}

	log.Printf("[DEBUG] Audio conversion completed: %d bytes input -> %d bytes output (format: %s)", len(resp.AudioContent), len(audioData), config.Format)
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
		return g.convertToRawOpus(audioData)
	case AudioFormatPCM:
		return audioData, nil // Already PCM from Google TTS
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", format)
	}
}

// convertToDCA converts PCM audio to DCA format using native Opus encoding
func (g *GoogleTTSManager) convertToDCA(pcmData []byte) ([]byte, error) {
	log.Printf("[DEBUG] Converting PCM to DCA format using native Opus: %d bytes", len(pcmData))

	// Discord Opus specifications
	const (
		sampleRate      = 48000 // 48kHz
		channels        = 2     // Stereo
		bitrate         = 64000 // 64kbps
		frameDurationMs = 20    // 20ms frames
		application     = opus.AppAudio
	)

	// Calculate frame size in samples (per channel)
	frameSize := (sampleRate * frameDurationMs) / 1000 // 960 samples per channel

	// Create Opus encoder
	encoder, err := opus.NewEncoder(sampleRate, channels, application)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %w", err)
	}

	// Set encoding parameters for Discord compatibility
	if err := encoder.SetBitrate(bitrate); err != nil {
		return nil, fmt.Errorf("failed to set bitrate: %w", err)
	}

	// Convert byte data to int16 samples
	if len(pcmData)%2 != 0 {
		return nil, fmt.Errorf("PCM data length must be even (16-bit samples)")
	}

	samples := make([]int16, len(pcmData)/2)
	for i := 0; i < len(samples); i++ {
		// Convert little-endian bytes to int16
		samples[i] = int16(pcmData[i*2]) | int16(pcmData[i*2+1])<<8
	}

	log.Printf("[DEBUG] Converted %d bytes to %d samples for Opus encoding", len(pcmData), len(samples))

	// Process samples in frames and encode to DCA
	var dcaBuffer bytes.Buffer
	frameCount := 0
	samplesPerFrame := frameSize * channels // Total samples per frame (both channels)

	for offset := 0; offset < len(samples); offset += samplesPerFrame {
		end := offset + samplesPerFrame
		if end > len(samples) {
			// Pad the last frame with silence
			lastFrame := make([]int16, samplesPerFrame)
			copy(lastFrame, samples[offset:])
			// The rest is already zero (silence)
			samples = append(samples[:offset], lastFrame...)
			end = offset + samplesPerFrame
		}

		frame := samples[offset:end]

		// Encode frame to Opus
		opusFrame := make([]byte, 4000) // Max Opus frame size
		n, err := encoder.Encode(frame, opusFrame)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Opus frame %d: %w", frameCount, err)
		}

		opusFrame = opusFrame[:n] // Trim to actual size

		// Write DCA frame header (2 bytes: frame length as int16 little-endian)
		frameLen := int16(len(opusFrame))
		if err := binary.Write(&dcaBuffer, binary.LittleEndian, frameLen); err != nil {
			return nil, fmt.Errorf("failed to write DCA frame header: %w", err)
		}

		// Write Opus frame data
		if _, err := dcaBuffer.Write(opusFrame); err != nil {
			return nil, fmt.Errorf("failed to write DCA frame data: %w", err)
		}

		frameCount++
	}

	totalSize := dcaBuffer.Len()
	avgFrameSize := 0
	if frameCount > 0 {
		avgFrameSize = totalSize / frameCount
	}

	log.Printf("[DEBUG] Native Opus encoding completed: %d frames, %d bytes total (avg %d bytes/frame)",
		frameCount, totalSize, avgFrameSize)

	return dcaBuffer.Bytes(), nil
}

// convertToRawOpus converts PCM audio to raw Opus format using native Opus encoding
func (g *GoogleTTSManager) convertToRawOpus(pcmData []byte) ([]byte, error) {
	log.Printf("[DEBUG] Converting PCM to raw Opus format using native library: %d bytes", len(pcmData))

	// Discord Opus specifications
	const (
		sampleRate      = 48000  // 48kHz
		channels        = 2      // Stereo
		bitrate         = 128000 // 128kbps for higher quality raw Opus
		frameDurationMs = 20     // 20ms frames
		application     = opus.AppAudio
	)

	// Calculate frame size in samples (per channel)
	frameSize := (sampleRate * frameDurationMs) / 1000 // 960 samples per channel

	// Create Opus encoder
	encoder, err := opus.NewEncoder(sampleRate, channels, application)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %w", err)
	}

	// Set encoding parameters
	if err := encoder.SetBitrate(bitrate); err != nil {
		return nil, fmt.Errorf("failed to set bitrate: %w", err)
	}

	// Convert byte data to int16 samples
	if len(pcmData)%2 != 0 {
		return nil, fmt.Errorf("PCM data length must be even (16-bit samples)")
	}

	samples := make([]int16, len(pcmData)/2)
	for i := 0; i < len(samples); i++ {
		// Convert little-endian bytes to int16
		samples[i] = int16(pcmData[i*2]) | int16(pcmData[i*2+1])<<8
	}

	// Encode all samples to raw Opus (not DCA format)
	var opusBuffer bytes.Buffer
	samplesPerFrame := frameSize * channels // Total samples per frame (both channels)

	for offset := 0; offset < len(samples); offset += samplesPerFrame {
		end := offset + samplesPerFrame
		if end > len(samples) {
			// Pad the last frame with silence
			lastFrame := make([]int16, samplesPerFrame)
			copy(lastFrame, samples[offset:])
			samples = append(samples[:offset], lastFrame...)
			end = offset + samplesPerFrame
		}

		frame := samples[offset:end]

		// Encode frame to Opus
		opusFrame := make([]byte, 4000) // Max Opus frame size
		n, err := encoder.Encode(frame, opusFrame)
		if err != nil {
			return nil, fmt.Errorf("failed to encode raw Opus frame: %w", err)
		}

		// Append raw Opus data (no DCA headers for raw format)
		opusBuffer.Write(opusFrame[:n])
	}

	opusData := opusBuffer.Bytes()
	log.Printf("[DEBUG] Native raw Opus encoding completed: %d bytes input -> %d bytes output", len(pcmData), len(opusData))

	return opusData, nil
}

// parseOpusStreamToDCA parses raw Opus stream into proper DCA format
// This respects Opus frame boundaries for better audio quality
func (g *GoogleTTSManager) parseOpusStreamToDCA(opusData []byte) ([]byte, error) {
	var dcaBuffer bytes.Buffer
	var frameCount int

	// Parse Opus frames from the raw stream
	// Opus frames start with a TOC (Table of Contents) byte that indicates frame structure
	offset := 0
	for offset < len(opusData) {
		if offset >= len(opusData) {
			break
		}

		// Find the next Opus frame by looking for frame patterns
		// For simplicity, we'll use a heuristic based on the working airhorn
		frameSize := g.estimateOpusFrameSize(opusData[offset:])
		if frameSize <= 0 || offset+frameSize > len(opusData) {
			// If we can't determine frame size, use remaining data
			frameSize = len(opusData) - offset
		}

		frameData := opusData[offset : offset+frameSize]

		// Write DCA frame header (2 bytes: frame length as int16 little-endian)
		frameLen := int16(len(frameData))
		if err := binary.Write(&dcaBuffer, binary.LittleEndian, frameLen); err != nil {
			return nil, fmt.Errorf("failed to write DCA frame header: %w", err)
		}

		// Write Opus frame data
		if _, err := dcaBuffer.Write(frameData); err != nil {
			return nil, fmt.Errorf("failed to write DCA frame data: %w", err)
		}

		offset += frameSize
		frameCount++

		// Safety check to prevent infinite loops
		if frameCount > 1000 {
			log.Printf("[DEBUG] Warning: Too many frames, stopping at %d", frameCount)
			break
		}
	}

	log.Printf("[DEBUG] Parsed %d bytes of Opus data into %d DCA frames", len(opusData), frameCount)
	return dcaBuffer.Bytes(), nil
}

// estimateOpusFrameSize estimates the size of an Opus frame based on patterns
// This is a heuristic approach to avoid complex Opus parsing
func (g *GoogleTTSManager) estimateOpusFrameSize(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// For 20ms frames at 64kbps, typical Opus frame sizes are:
	// - 160-200 bytes for speech
	// - 120-180 bytes for audio
	// The airhorn averages ~163 bytes per frame

	// Use a simple heuristic: look for the next potential frame start
	// or use a reasonable default size
	const defaultFrameSize = 160
	const maxFrameSize = 300

	if len(data) <= defaultFrameSize {
		return len(data)
	}

	// Look for patterns that might indicate frame boundaries
	// This is a simplified approach - in practice, proper Opus parsing would be better
	for i := defaultFrameSize; i < len(data) && i < maxFrameSize; i++ {
		// Look for potential TOC byte patterns (simplified heuristic)
		if data[i] >= 0x00 && data[i] <= 0xFF {
			// This could be a frame boundary, but it's just a heuristic
			if i > defaultFrameSize/2 { // Ensure reasonable frame size
				return i
			}
		}
	}

	// Default to a reasonable frame size
	if len(data) > defaultFrameSize {
		return defaultFrameSize
	}
	return len(data)
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

	// Debug logging to see if data is being read
	if n > 0 {
		log.Printf("[DEBUG] bytesReader: Read %d bytes, position now %d/%d", n, r.pos, len(r.data))
	}

	return n, nil
}

// processAudioForDiscord converts audio to Discord format (48kHz stereo)
// Handles mono->stereo conversion and sample rate conversion
func (g *GoogleTTSManager) processAudioForDiscord(pcmData []byte, fromRate, fromChannels int) []byte {
	const (
		targetRate     = 48000
		targetChannels = 2
	)

	log.Printf("[DEBUG] Processing audio: %dHz %dch -> %dHz %dch", fromRate, fromChannels, targetRate, targetChannels)

	// Convert bytes to int16 samples
	if len(pcmData)%2 != 0 {
		log.Printf("[DEBUG] Warning: PCM data length not even, truncating")
		pcmData = pcmData[:len(pcmData)-1]
	}

	inputSamples := make([]int16, len(pcmData)/2)
	for i := 0; i < len(inputSamples); i++ {
		inputSamples[i] = int16(pcmData[i*2]) | int16(pcmData[i*2+1])<<8
	}

	// Step 1: Convert mono to stereo if needed
	var stereoSamples []int16
	if fromChannels == 1 {
		// Mono to stereo: duplicate each sample
		stereoSamples = make([]int16, len(inputSamples)*2)
		for i, sample := range inputSamples {
			stereoSamples[i*2] = sample   // Left channel
			stereoSamples[i*2+1] = sample // Right channel (same as left)
		}
		log.Printf("[DEBUG] Converted mono to stereo: %d -> %d samples", len(inputSamples), len(stereoSamples))
	} else {
		// Already stereo
		stereoSamples = inputSamples
	}

	// Step 2: Resample to target rate if needed
	var finalSamples []int16
	if fromRate != targetRate {
		finalSamples = g.resampleStereo(stereoSamples, fromRate, targetRate)
		log.Printf("[DEBUG] Resampled: %d samples (%dHz) -> %d samples (%dHz)",
			len(stereoSamples), fromRate, len(finalSamples), targetRate)
	} else {
		finalSamples = stereoSamples
	}

	// Convert back to bytes
	outputData := make([]byte, len(finalSamples)*2)
	for i, sample := range finalSamples {
		outputData[i*2] = byte(sample & 0xFF)
		outputData[i*2+1] = byte((sample >> 8) & 0xFF)
	}

	return outputData
}

// resampleStereo resamples stereo PCM audio using linear interpolation
func (g *GoogleTTSManager) resampleStereo(stereoSamples []int16, fromRate, toRate int) []int16 {
	if fromRate == toRate {
		return stereoSamples // No resampling needed
	}

	// Calculate resampling ratio
	ratio := float64(toRate) / float64(fromRate)
	inputFrames := len(stereoSamples) / 2 // Stereo frames (left+right pairs)
	outputFrames := int(float64(inputFrames) * ratio)
	outputSamples := make([]int16, outputFrames*2) // 2 samples per frame

	// Simple linear interpolation resampling for stereo
	for i := 0; i < outputFrames; i++ {
		// Calculate the corresponding position in the input
		srcPos := float64(i) / ratio
		srcIndex := int(srcPos)

		if srcIndex >= inputFrames-1 {
			// Use the last frame if we're at the end
			outputSamples[i*2] = stereoSamples[(inputFrames-1)*2]     // Left
			outputSamples[i*2+1] = stereoSamples[(inputFrames-1)*2+1] // Right
		} else {
			// Linear interpolation between two stereo frames
			frac := srcPos - float64(srcIndex)

			// Left channel
			left1 := float64(stereoSamples[srcIndex*2])
			left2 := float64(stereoSamples[(srcIndex+1)*2])
			leftInterpolated := left1 + frac*(left2-left1)
			outputSamples[i*2] = int16(leftInterpolated)

			// Right channel
			right1 := float64(stereoSamples[srcIndex*2+1])
			right2 := float64(stereoSamples[(srcIndex+1)*2+1])
			rightInterpolated := right1 + frac*(right2-right1)
			outputSamples[i*2+1] = int16(rightInterpolated)
		}
	}

	return outputSamples
}
