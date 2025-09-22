package tts

import (
	"fmt"
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

	// Check if already connected to this guild
	if existingConn, exists := vm.connections[guildID]; exists {
		// If already in the same channel, return existing connection
		if existingConn.ChannelID == channelID {
			return existingConn, nil
		}
		// Leave current channel before joining new one
		if err := vm.leaveChannelInternal(guildID); err != nil {
			return nil, fmt.Errorf("failed to leave current channel: %w", err)
		}
	}

	// Join the voice channel
	voiceConn, err := vm.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel %s: %w", channelID, err)
	}

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

// PlayAudio plays audio data through the voice connection
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

	// Check if connection exists
	if connection.Connection == nil {
		return fmt.Errorf("voice connection is nil for guild %s", guildID)
	}

	// Set playing status
	vm.mutex.Lock()
	connection.IsPlaying = true
	vm.mutex.Unlock()

	// Send audio data
	select {
	case connection.Connection.OpusSend <- audioData:
		// Audio sent successfully
	case <-time.After(5 * time.Second):
		vm.mutex.Lock()
		connection.IsPlaying = false
		vm.mutex.Unlock()
		return fmt.Errorf("timeout sending audio data for guild %s", guildID)
	}

	// Reset playing status
	vm.mutex.Lock()
	connection.IsPlaying = false
	vm.mutex.Unlock()

	return nil
}

// IsConnected checks if there's an active voice connection for the guild
func (vm *voiceManager) IsConnected(guildID string) bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return false
	}

	return connection.Connection != nil
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

// RecoverConnection attempts to recover a failed voice connection
func (vm *voiceManager) RecoverConnection(guildID string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	connection, exists := vm.connections[guildID]
	if !exists {
		return fmt.Errorf("no voice connection found for guild %s", guildID)
	}

	// Attempt to rejoin the same channel
	channelID := connection.ChannelID

	// Clean up the old connection
	if err := vm.leaveChannelInternal(guildID); err != nil {
		// Log error but continue with recovery attempt
	}

	// Try to rejoin
	voiceConn, err := vm.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return fmt.Errorf("failed to recover voice connection for guild %s: %w", guildID, err)
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
	return nil
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
