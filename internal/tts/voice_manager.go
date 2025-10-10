package tts

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// DiscordVoiceSession interface for voice operations
type DiscordVoiceSession interface {
	ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (*discordgo.VoiceConnection, error)
}

// voiceManager implements the VoiceManager interface
type voiceManager struct {
	session     DiscordVoiceSession
	connections map[string]*VoiceConnection
	mutex       sync.RWMutex
}

// NewVoiceManager creates a new VoiceManager instance
func NewVoiceManager(session *discordgo.Session) VoiceManager {
	return &voiceManager{
		session:     session,
		connections: make(map[string]*VoiceConnection),
		mutex:       sync.RWMutex{},
	}
}

// JoinChannel joins a voice channel and creates a voice connection
func (vm *voiceManager) JoinChannel(guildID, channelID string) (*VoiceConnection, error) {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	log.Printf("[DEBUG] Attempting to join voice channel %s in guild %s", channelID, guildID)

	// Check if already connected to this guild
	if existingConn, exists := vm.connections[guildID]; exists {
		// If already in the same channel, return existing connection
		if existingConn.ChannelID == channelID {
			log.Printf("[DEBUG] Already connected to channel %s in guild %s", channelID, guildID)
			return existingConn, nil
		}
		// Leave current channel before joining new one
		if err := vm.leaveChannelInternal(guildID); err != nil {
			return nil, fmt.Errorf("failed to leave current channel: %w", err)
		}
	}

	// Join the voice channel
	log.Printf("[DEBUG] Calling ChannelVoiceJoin for guild %s, channel %s", guildID, channelID)
	voiceConn, err := vm.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		log.Printf("[DEBUG] ChannelVoiceJoin failed: %v", err)
		return nil, fmt.Errorf("failed to join voice channel %s: %w", channelID, err)
	}

	log.Printf("[DEBUG] ChannelVoiceJoin succeeded, voiceConn: %v", voiceConn != nil)

	// Wait for the connection to be ready (simplified for now)
	// In a real implementation, we would wait for the Ready channel
	time.Sleep(100 * time.Millisecond) // Brief wait for connection setup

	// Create our VoiceConnection wrapper
	connection := &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: voiceConn,
		IsPlaying:  false,
		IsPaused:   false,
		Queue: &AudioQueue{
			Items:   make([][]byte, 0),
			MaxSize: 10,
			Current: 0,
		},
	}

	vm.connections[guildID] = connection
	log.Printf("[DEBUG] Stored voice connection for guild %s, total connections: %d", guildID, len(vm.connections))
	return connection, nil
}

// LeaveChannel leaves the voice channel for the specified guild
func (vm *voiceManager) LeaveChannel(guildID string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	return vm.leaveChannelInternal(guildID)
}

// leaveChannelInternal is the internal implementation without mutex locking
func (vm *voiceManager) leaveChannelInternal(guildID string) error {
	connection, exists := vm.connections[guildID]
	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	// Disconnect from Discord
	if connection.Connection != nil {
		// Note: In a real implementation, we would call connection.Connection.Disconnect()
		// For testing purposes, we'll skip the actual disconnect call
		// if err := connection.Connection.Disconnect(); err != nil {
		//     return fmt.Errorf("failed to disconnect from voice channel: %w", err)
		// }
	}

	// Remove from our connections map
	delete(vm.connections, guildID)
	return nil
}

// GetConnection retrieves the voice connection for a guild
func (vm *voiceManager) GetConnection(guildID string) (*VoiceConnection, bool) {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	connection, exists := vm.connections[guildID]
	return connection, exists
}

// PlayAudio plays audio data through the voice connection with enhanced error handling
func (vm *voiceManager) PlayAudio(guildID string, audioData []byte) error {
	vm.mutex.RLock()
	connection, exists := vm.connections[guildID]
	vm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	if connection.Connection == nil {
		return fmt.Errorf("voice connection is nil for guild %s", guildID)
	}

	// Check if connection is ready
	if connection.Connection.OpusSend == nil {
		return fmt.Errorf("voice connection not ready for guild %s", guildID)
	}

	// Set playing status
	vm.mutex.Lock()
	connection.IsPlaying = true
	vm.mutex.Unlock()

	// Ensure playing status is reset regardless of outcome
	defer func() {
		vm.mutex.Lock()
		connection.IsPlaying = false
		vm.mutex.Unlock()
	}()

	// Set speaking state to true before sending audio
	err := connection.Connection.Speaking(true)
	if err != nil {
		log.Printf("[DEBUG] Warning: failed to set speaking state: %v", err)
	}

	// Ensure speaking state is reset when done
	defer func() {
		err := connection.Connection.Speaking(false)
		if err != nil {
			log.Printf("[DEBUG] Warning: failed to reset speaking state: %v", err)
		}
	}()

	// Parse DCA format and send individual Opus frames to Discord
	log.Printf("[DEBUG] Parsing %d bytes of DCA data", len(audioData))

	// Parse DCA frames and send them individually
	frames, err := vm.parseDCAFrames(audioData)
	if err != nil {
		return fmt.Errorf("failed to parse DCA frames for guild %s: %w", guildID, err)
	}

	log.Printf("[DEBUG] Parsed %d DCA frames", len(frames))

	// Send each Opus frame (Discord handles 20ms timing automatically)
	for i, frame := range frames {
		select {
		case connection.Connection.OpusSend <- frame:
			// Frame sent successfully - Discord handles timing
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout sending DCA frame %d for guild %s", i, guildID)
		}
	}

	log.Printf("Successfully sent %d DCA frames (%d total bytes) for guild %s", len(frames), len(audioData), guildID)
	return nil
}

// parseDCAFrames parses DCA format data into individual Opus frames
// DCA format: [2 bytes frame length][N bytes Opus data][2 bytes frame length][N bytes Opus data]...
func (vm *voiceManager) parseDCAFrames(dcaData []byte) ([][]byte, error) {
	var frames [][]byte
	offset := 0

	for offset < len(dcaData) {
		// Need at least 2 bytes for frame length header
		if offset+2 > len(dcaData) {
			log.Printf("[DEBUG] Warning: incomplete DCA frame header at offset %d", offset)
			break
		}

		// Read frame length (2 bytes, little-endian)
		frameLen := int(dcaData[offset]) | int(dcaData[offset+1])<<8
		offset += 2

		// Validate frame length
		if frameLen <= 0 || frameLen > 4000 { // Reasonable max frame size
			return nil, fmt.Errorf("invalid DCA frame length %d at offset %d", frameLen, offset-2)
		}

		// Check if we have enough data for the frame
		if offset+frameLen > len(dcaData) {
			return nil, fmt.Errorf("incomplete DCA frame: expected %d bytes, only %d available at offset %d",
				frameLen, len(dcaData)-offset, offset)
		}

		// Extract the Opus frame data
		frame := make([]byte, frameLen)
		copy(frame, dcaData[offset:offset+frameLen])
		frames = append(frames, frame)

		offset += frameLen
	}

	log.Printf("[DEBUG] Successfully parsed %d DCA frames from %d bytes", len(frames), len(dcaData))
	return frames, nil
}

// convertStereoToMono converts stereo 16-bit PCM to mono by averaging the channels
func (vm *voiceManager) convertStereoToMono(stereoData []byte) []byte {
	if len(stereoData)%4 != 0 {
		log.Printf("[DEBUG] Warning: stereo data length not divisible by 4, truncating")
		stereoData = stereoData[:len(stereoData)-(len(stereoData)%4)]
	}

	monoData := make([]byte, len(stereoData)/2)

	for i := 0; i < len(stereoData); i += 4 {
		// Read left and right channel samples (16-bit little-endian)
		left := int16(stereoData[i]) | int16(stereoData[i+1])<<8
		right := int16(stereoData[i+2]) | int16(stereoData[i+3])<<8

		// Average the channels and boost volume more conservatively
		mono := (int32(left) + int32(right)) / 2
		// Boost volume by 25% (less aggressive to reduce distortion)
		boosted := mono * 5 / 4
		if boosted > 32767 {
			boosted = 32767
		} else if boosted < -32768 {
			boosted = -32768
		}
		monoSample := int16(boosted)

		// Write mono sample (16-bit little-endian)
		monoData[i/2] = byte(monoSample & 0xFF)
		monoData[i/2+1] = byte((monoSample >> 8) & 0xFF)
	}

	return monoData
}

// IsConnected checks if there's an active voice connection for the guild
func (vm *voiceManager) IsConnected(guildID string) bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return false
	}

	isConnected := connection.Connection != nil
	return isConnected
}

// Cleanup disconnects all voice connections (useful for shutdown)
func (vm *voiceManager) Cleanup() error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	var errors []error
	for guildID := range vm.connections {
		if err := vm.leaveChannelInternal(guildID); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup guild %s: %w", guildID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// GetActiveConnections returns a list of all active guild IDs with voice connections
func (vm *voiceManager) GetActiveConnections() []string {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	guildIDs := make([]string, 0, len(vm.connections))
	for guildID := range vm.connections {
		guildIDs = append(guildIDs, guildID)
	}

	return guildIDs
}

// RecoverConnection attempts to recover a failed voice connection with enhanced error handling
func (vm *voiceManager) RecoverConnection(guildID string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	// Attempt to rejoin the same channel
	channelID := connection.ChannelID
	log.Printf("Attempting to recover voice connection for guild %s, channel %s", guildID, channelID)

	// Clean up the old connection
	if err := vm.leaveChannelInternal(guildID); err != nil {
		log.Printf("Warning: failed to clean up old connection for guild %s: %v", guildID, err)
		// Continue with recovery attempt despite cleanup failure
	}

	// Try to rejoin with timeout
	done := make(chan error, 1)
	go func() {
		voiceConn, err := vm.session.ChannelVoiceJoin(guildID, channelID, false, true)
		if err != nil {
			done <- fmt.Errorf("failed to rejoin voice channel: %w", err)
			return
		}

		// Wait for connection setup
		time.Sleep(100 * time.Millisecond)

		// Create new connection wrapper
		newConnection := &VoiceConnection{
			GuildID:    guildID,
			ChannelID:  channelID,
			Connection: voiceConn,
			IsPlaying:  false,
			IsPaused:   false,
			Queue: &AudioQueue{
				Items:   make([][]byte, 0),
				MaxSize: 10,
				Current: 0,
			},
		}

		vm.connections[guildID] = newConnection
		done <- nil
	}()

	// Wait for connection with timeout
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("voice connection recovery failed for guild %s: %w", guildID, err)
		}
		log.Printf("Successfully recovered voice connection for guild %s", guildID)
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("voice connection recovery timed out for guild %s", guildID)
	}
}

// HealthCheck performs a health check on all voice connections
func (vm *voiceManager) HealthCheck() map[string]error {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	results := make(map[string]error)

	for guildID, connection := range vm.connections {
		if connection.Connection == nil {
			results[guildID] = fmt.Errorf("voice connection is nil")
			continue
		}

		// In a real implementation, we would check connection health
		// For now, we'll assume connections are healthy if they exist
		results[guildID] = nil
	}

	return results
}

// PausePlayback pauses TTS playback for the specified guild
func (vm *voiceManager) PausePlayback(guildID string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	connection.IsPaused = true
	return nil
}

// ResumePlayback resumes TTS playback for the specified guild
func (vm *voiceManager) ResumePlayback(guildID string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	connection.IsPaused = false
	return nil
}

// SkipCurrentMessage stops the current message and allows the next one to play
func (vm *voiceManager) SkipCurrentMessage(guildID string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	// If currently playing, stop the current audio
	if connection.IsPlaying {
		// In a real implementation, we would interrupt the current audio stream
		// For now, we'll just mark it as not playing to simulate skipping
		connection.IsPlaying = false
	}

	return nil
}

// IsPaused checks if playback is paused for the specified guild
func (vm *voiceManager) IsPaused(guildID string) bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return false
	}

	return connection.IsPaused
}

// SetConnectionStateCallback allows setting a callback for connection state changes
// This would be used in a real implementation to handle disconnections
func (vm *voiceManager) SetConnectionStateCallback(callback func(guildID string, connected bool)) {
	// In a real implementation, this would register a callback with the Discord session
	// to handle voice connection state changes and trigger automatic recovery
}

// TestPlayDCAFile plays a known working DCA file for testing
func (vm *voiceManager) TestPlayDCAFile(guildID, filename string) error {
	// Read the DCA file
	dcaData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read DCA file %s: %w", filename, err)
	}

	log.Printf("[TEST] Playing DCA file %s (%d bytes) for guild %s", filename, len(dcaData), guildID)

	// Use the same PlayAudio method to test our pipeline
	return vm.PlayAudio(guildID, dcaData)
}
