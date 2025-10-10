package tts

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

// mockDiscordVoiceSession provides a mock implementation of DiscordVoiceSession for testing
type mockDiscordVoiceSession struct {
	joinFunc func(guildID, channelID string, mute, deaf bool) (*discordgo.VoiceConnection, error)
}

func (m *mockDiscordVoiceSession) ChannelVoiceJoin(guildID, channelID string, mute, deaf bool) (*discordgo.VoiceConnection, error) {
	if m.joinFunc != nil {
		return m.joinFunc(guildID, channelID, mute, deaf)
	}
	return nil, nil
}

// createMockVoiceConnection creates a mock voice connection for testing
func createMockVoiceConnection(guildID, channelID string) *discordgo.VoiceConnection {
	return &discordgo.VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		OpusSend:  make(chan []byte, 2),
	}
}

func TestNewVoiceManager(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	assert.NotNil(t, vm)

	// Test that it implements the interface
	var _ VoiceManager = vm
}

func TestVoiceManager_JoinChannel_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Create a mock voice connection
	mockConn := createMockVoiceConnection(guildID, channelID)

	// Replace the session with a mock that returns our mock connection
	vm.session = &mockDiscordVoiceSession{
		joinFunc: func(gID, cID string, mute, deaf bool) (*discordgo.VoiceConnection, error) {
			assert.Equal(t, guildID, gID)
			assert.Equal(t, channelID, cID)
			assert.False(t, mute)
			assert.True(t, deaf)
			return mockConn, nil
		},
	}

	// Test joining a channel
	conn, err := vm.JoinChannel(guildID, channelID)

	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, guildID, conn.GuildID)
	assert.Equal(t, channelID, conn.ChannelID)
	assert.Equal(t, mockConn, conn.Connection)
	assert.False(t, conn.IsPlaying)
	assert.NotNil(t, conn.Queue)
	assert.Equal(t, 10, conn.Queue.MaxSize)
}

func TestVoiceManager_JoinChannel_AlreadyInSameChannel(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Create existing connection
	existingConn := &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  false,
		Queue: &AudioQueue{
			Items:   make([][]byte, 0),
			MaxSize: 10,
		},
	}
	vm.connections[guildID] = existingConn

	// Test joining the same channel
	conn, err := vm.JoinChannel(guildID, channelID)

	assert.NoError(t, err)
	assert.Equal(t, existingConn, conn)
}

func TestVoiceManager_LeaveChannel_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Create mock connection
	mockConn := createMockVoiceConnection(guildID, channelID)

	// Add connection to manager
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: mockConn,
	}

	// Test leaving channel
	err := vm.LeaveChannel(guildID)

	assert.NoError(t, err)

	// Verify connection was removed
	_, exists := vm.connections[guildID]
	assert.False(t, exists)
}

func TestVoiceManager_LeaveChannel_NotConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	guildID := "guild123"

	// Test leaving when not connected
	err := vm.LeaveChannel(guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no voice connection found")
}

func TestVoiceManager_GetConnection(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Test getting non-existent connection
	conn, exists := vm.GetConnection(guildID)
	assert.Nil(t, conn)
	assert.False(t, exists)

	// Add connection
	expectedConn := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
	}
	vm.connections[guildID] = expectedConn

	// Test getting existing connection
	conn, exists = vm.GetConnection(guildID)
	assert.Equal(t, expectedConn, conn)
	assert.True(t, exists)
}

func TestVoiceManager_PlayAudio_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"
	// Create valid DCA format test data
	// DCA format: [2 bytes frame length][N bytes Opus data]
	opusFrame := []byte("mock opus frame data") // 19 bytes
	frameLen := len(opusFrame)
	audioData := make([]byte, 2+frameLen)
	audioData[0] = byte(frameLen & 0xFF)        // Low byte of frame length
	audioData[1] = byte((frameLen >> 8) & 0xFF) // High byte of frame length
	copy(audioData[2:], opusFrame)              // Opus frame data

	// Create mock connection
	mockConn := createMockVoiceConnection(guildID, channelID)

	// Add connection to manager
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: mockConn,
		IsPlaying:  false,
	}

	// Test playing audio
	err := vm.PlayAudio(guildID, audioData)

	assert.NoError(t, err)

	// Verify audio was sent (should receive the Opus frame data, not the full DCA data)
	select {
	case receivedData := <-mockConn.OpusSend:
		assert.Equal(t, opusFrame, receivedData)
	case <-time.After(1 * time.Second):
		t.Fatal("Audio data was not sent")
	}
}

func TestVoiceManager_PlayAudio_NotConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	guildID := "guild123"
	audioData := []byte("test audio data")

	// Test playing audio when not connected
	err := vm.PlayAudio(guildID, audioData)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no voice connection found")
}

func TestVoiceManager_PlayAudio_ConnectionNil(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	audioData := []byte("test audio data")

	// Add connection with nil Connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  "channel456",
		Connection: nil,
		IsPlaying:  false,
	}

	// Test playing audio when connection is nil
	err := vm.PlayAudio(guildID, audioData)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "voice connection is nil")
}

func TestVoiceManager_IsConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Test when not connected
	assert.False(t, vm.IsConnected(guildID))

	// Add connection
	mockConn := createMockVoiceConnection(guildID, channelID)
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: mockConn,
	}

	// Test when connected
	assert.True(t, vm.IsConnected(guildID))

	// Test with nil connection
	vm.connections[guildID].Connection = nil
	assert.False(t, vm.IsConnected(guildID))
}

func TestVoiceManager_Cleanup(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	// Add multiple connections
	guild1 := "guild1"
	guild2 := "guild2"

	mockConn1 := createMockVoiceConnection(guild1, "channel1")
	mockConn2 := createMockVoiceConnection(guild2, "channel2")

	vm.connections[guild1] = &VoiceConnection{
		GuildID:    guild1,
		ChannelID:  "channel1",
		Connection: mockConn1,
	}
	vm.connections[guild2] = &VoiceConnection{
		GuildID:    guild2,
		ChannelID:  "channel2",
		Connection: mockConn2,
	}

	// Test cleanup
	err := vm.Cleanup()

	assert.NoError(t, err)
	assert.Empty(t, vm.connections)
}

func TestVoiceManager_GetActiveConnections(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	// Test with no connections
	connections := vm.GetActiveConnections()
	assert.Empty(t, connections)

	// Add connections
	guild1 := "guild1"
	guild2 := "guild2"

	vm.connections[guild1] = &VoiceConnection{GuildID: guild1}
	vm.connections[guild2] = &VoiceConnection{GuildID: guild2}

	// Test with connections
	connections = vm.GetActiveConnections()
	assert.Len(t, connections, 2)
	assert.Contains(t, connections, guild1)
	assert.Contains(t, connections, guild2)
}

func TestVoiceManager_ConcurrentAccess(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add a connection
	mockConn := createMockVoiceConnection(guildID, channelID)
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: mockConn,
	}

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Read operations
	go func() {
		for i := 0; i < 100; i++ {
			vm.GetConnection(guildID)
			vm.IsConnected(guildID)
			vm.GetActiveConnections()
		}
		done <- true
	}()

	// Goroutine 2: Read operations
	go func() {
		for i := 0; i < 100; i++ {
			vm.GetConnection(guildID)
			vm.IsConnected(guildID)
			vm.GetActiveConnections()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify connection still exists
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.NotNil(t, conn)
}

func TestVoiceManager_RecoverConnection_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add existing connection
	existingConn := &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
	}
	vm.connections[guildID] = existingConn

	// Create new mock connection for recovery
	newMockConn := createMockVoiceConnection(guildID, channelID)

	// Set up mock session for recovery
	vm.session = &mockDiscordVoiceSession{
		joinFunc: func(gID, cID string, mute, deaf bool) (*discordgo.VoiceConnection, error) {
			assert.Equal(t, guildID, gID)
			assert.Equal(t, channelID, cID)
			return newMockConn, nil
		},
	}

	// Test recovery
	err := vm.RecoverConnection(guildID)

	assert.NoError(t, err)

	// Verify new connection was created
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.Equal(t, newMockConn, conn.Connection)
}

func TestVoiceManager_RecoverConnection_NotConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	guildID := "guild123"

	// Test recovery when not connected
	err := vm.RecoverConnection(guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no voice connection found")
}

func TestVoiceManager_HealthCheck(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	// Test with no connections
	results := vm.HealthCheck()
	assert.Empty(t, results)

	// Add healthy connection
	guild1 := "guild1"
	vm.connections[guild1] = &VoiceConnection{
		GuildID:    guild1,
		ChannelID:  "channel1",
		Connection: createMockVoiceConnection(guild1, "channel1"),
	}

	// Add unhealthy connection (nil connection)
	guild2 := "guild2"
	vm.connections[guild2] = &VoiceConnection{
		GuildID:    guild2,
		ChannelID:  "channel2",
		Connection: nil,
	}

	// Test health check
	results = vm.HealthCheck()
	assert.Len(t, results, 2)
	assert.NoError(t, results[guild1])
	assert.Error(t, results[guild2])
	assert.Contains(t, results[guild2].Error(), "voice connection is nil")
}

func TestVoiceManager_SetConnectionStateCallback(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	// Test setting callback (should not panic)
	callbackCalled := false
	callback := func(guildID string, connected bool) {
		callbackCalled = true
	}

	vm.SetConnectionStateCallback(callback)

	// In a real implementation, this would trigger the callback
	// For now, we just verify it doesn't panic
	assert.False(t, callbackCalled) // Callback not called in mock implementation
}

// Tests for new control methods

func TestVoiceManager_PausePlayback_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  false,
		IsPaused:   false,
	}

	// Test pausing playback
	err := vm.PausePlayback(guildID)

	assert.NoError(t, err)

	// Verify connection is paused
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.True(t, conn.IsPaused)
}

func TestVoiceManager_PausePlayback_NotConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	guildID := "guild123"

	// Test pausing when not connected
	err := vm.PausePlayback(guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no voice connection found")
}

func TestVoiceManager_ResumePlayback_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add paused connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  false,
		IsPaused:   true,
	}

	// Test resuming playback
	err := vm.ResumePlayback(guildID)

	assert.NoError(t, err)

	// Verify connection is not paused
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.False(t, conn.IsPaused)
}

func TestVoiceManager_ResumePlayback_NotConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	guildID := "guild123"

	// Test resuming when not connected
	err := vm.ResumePlayback(guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no voice connection found")
}

func TestVoiceManager_SkipCurrentMessage_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add playing connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  true,
		IsPaused:   false,
	}

	// Test skipping current message
	err := vm.SkipCurrentMessage(guildID)

	assert.NoError(t, err)

	// Verify connection is no longer playing
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.False(t, conn.IsPlaying)
}

func TestVoiceManager_SkipCurrentMessage_NotPlaying(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add non-playing connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  false,
		IsPaused:   false,
	}

	// Test skipping when not playing
	err := vm.SkipCurrentMessage(guildID)

	assert.NoError(t, err)

	// Verify connection state unchanged
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.False(t, conn.IsPlaying)
}

func TestVoiceManager_SkipCurrentMessage_NotConnected(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session)

	guildID := "guild123"

	// Test skipping when not connected
	err := vm.SkipCurrentMessage(guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no voice connection found")
}

func TestVoiceManager_IsPaused_Success(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Test when not connected
	assert.False(t, vm.IsPaused(guildID))

	// Add unpaused connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  false,
		IsPaused:   false,
	}

	// Test when not paused
	assert.False(t, vm.IsPaused(guildID))

	// Set to paused
	vm.connections[guildID].IsPaused = true

	// Test when paused
	assert.True(t, vm.IsPaused(guildID))
}

func TestVoiceManager_ControlMethods_ConcurrentAccess(t *testing.T) {
	session := &discordgo.Session{}
	vm := NewVoiceManager(session).(*voiceManager)

	guildID := "guild123"
	channelID := "channel456"

	// Add connection
	vm.connections[guildID] = &VoiceConnection{
		GuildID:    guildID,
		ChannelID:  channelID,
		Connection: createMockVoiceConnection(guildID, channelID),
		IsPlaying:  false,
		IsPaused:   false,
	}

	// Test concurrent control operations
	done := make(chan bool, 3)

	// Goroutine 1: Pause/Resume operations
	go func() {
		for i := 0; i < 50; i++ {
			vm.PausePlayback(guildID)
			vm.ResumePlayback(guildID)
		}
		done <- true
	}()

	// Goroutine 2: Skip operations
	go func() {
		for i := 0; i < 50; i++ {
			vm.SkipCurrentMessage(guildID)
		}
		done <- true
	}()

	// Goroutine 3: Status checks
	go func() {
		for i := 0; i < 50; i++ {
			vm.IsPaused(guildID)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	<-done
	<-done
	<-done

	// Verify connection still exists
	conn, exists := vm.GetConnection(guildID)
	assert.True(t, exists)
	assert.NotNil(t, conn)
}
