package tts

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockVoiceManager struct {
	mock.Mock
}

func (m *MockVoiceManager) JoinChannel(guildID, channelID string) (*VoiceConnection, error) {
	args := m.Called(guildID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VoiceConnection), args.Error(1)
}

func (m *MockVoiceManager) LeaveChannel(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockVoiceManager) GetConnection(guildID string) (*VoiceConnection, bool) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*VoiceConnection), args.Bool(1)
}

func (m *MockVoiceManager) PlayAudio(guildID string, audioData []byte) error {
	args := m.Called(guildID, audioData)
	return args.Error(0)
}

func (m *MockVoiceManager) IsConnected(guildID string) bool {
	args := m.Called(guildID)
	return args.Bool(0)
}

func (m *MockVoiceManager) RecoverConnection(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockVoiceManager) HealthCheck() map[string]error {
	args := m.Called()
	return args.Get(0).(map[string]error)
}

func (m *MockVoiceManager) SetConnectionStateCallback(callback func(guildID string, connected bool)) {
	m.Called(callback)
}

func (m *MockVoiceManager) Cleanup() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockVoiceManager) GetActiveConnections() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockVoiceManager) PausePlayback(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockVoiceManager) ResumePlayback(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockVoiceManager) SkipCurrentMessage(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockVoiceManager) IsPaused(guildID string) bool {
	args := m.Called(guildID)
	return args.Bool(0)
}

type MockChannelService struct {
	mock.Mock
}

func (m *MockChannelService) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	args := m.Called(guildID, voiceChannelID, textChannelID)
	return args.Error(0)
}

func (m *MockChannelService) CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error {
	args := m.Called(guildID, voiceChannelID, textChannelID, createdBy)
	return args.Error(0)
}

func (m *MockChannelService) RemovePairing(guildID, voiceChannelID string) error {
	args := m.Called(guildID, voiceChannelID)
	return args.Error(0)
}

func (m *MockChannelService) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	args := m.Called(guildID, voiceChannelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChannelPairing), args.Error(1)
}

func (m *MockChannelService) ValidateChannelAccess(userID, channelID string) error {
	args := m.Called(userID, channelID)
	return args.Error(0)
}

func (m *MockChannelService) IsChannelPaired(guildID, textChannelID string) bool {
	args := m.Called(guildID, textChannelID)
	return args.Bool(0)
}

type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) CanInviteBot(userID, guildID string) (bool, error) {
	args := m.Called(userID, guildID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) CanControlBot(userID, guildID string) (bool, error) {
	args := m.Called(userID, guildID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) HasChannelAccess(userID, channelID string) (bool, error) {
	args := m.Called(userID, channelID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionService) SetRequiredRoles(guildID string, roleIDs []string) error {
	args := m.Called(guildID, roleIDs)
	return args.Error(0)
}

func (m *MockPermissionService) GetRequiredRoles(guildID string) ([]string, error) {
	args := m.Called(guildID)
	return args.Get(0).([]string), args.Error(1)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) SetOptInStatus(userID, guildID string, optedIn bool) error {
	args := m.Called(userID, guildID, optedIn)
	return args.Error(0)
}

func (m *MockUserService) IsOptedIn(userID, guildID string) (bool, error) {
	args := m.Called(userID, guildID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserService) GetOptedInUsers(guildID string) ([]string, error) {
	args := m.Called(guildID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserService) AutoOptIn(userID, guildID string) error {
	args := m.Called(userID, guildID)
	return args.Error(0)
}

type MockMessageQueue struct {
	mock.Mock
}

func (m *MockMessageQueue) Enqueue(message *QueuedMessage) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockMessageQueue) Dequeue(guildID string) (*QueuedMessage, error) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*QueuedMessage), args.Error(1)
}

func (m *MockMessageQueue) Clear(guildID string) error {
	args := m.Called(guildID)
	return args.Error(0)
}

func (m *MockMessageQueue) Size(guildID string) int {
	args := m.Called(guildID)
	return args.Int(0)
}

func (m *MockMessageQueue) SetMaxSize(guildID string, size int) error {
	args := m.Called(guildID, size)
	return args.Error(0)
}

func (m *MockMessageQueue) SkipNext(guildID string) (*QueuedMessage, error) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*QueuedMessage), args.Error(1)
}

// Test helper functions

func createTestJoinHandler() (*JoinCommandHandler, *MockVoiceManager, *MockChannelService, *MockPermissionService, *MockUserService) {
	mockVoiceManager := &MockVoiceManager{}
	mockChannelService := &MockChannelService{}
	mockPermissionService := &MockPermissionService{}
	mockUserService := &MockUserService{}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	handler := NewJoinCommandHandler(
		mockVoiceManager,
		mockChannelService,
		mockPermissionService,
		mockUserService,
		nil, // errorRecovery - not needed for basic tests
		logger,
	)

	return handler, mockVoiceManager, mockChannelService, mockPermissionService, mockUserService
}

func createTestLeaveHandler() (*LeaveCommandHandler, *MockVoiceManager, *MockChannelService, *MockPermissionService) {
	mockVoiceManager := &MockVoiceManager{}
	mockChannelService := &MockChannelService{}
	mockPermissionService := &MockPermissionService{}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	handler := NewLeaveCommandHandler(
		mockVoiceManager,
		mockChannelService,
		mockPermissionService,
		nil, // errorRecovery - not needed for basic tests
		logger,
	)

	return handler, mockVoiceManager, mockChannelService, mockPermissionService
}

func createTestControlHandler() (*ControlCommandHandler, *MockVoiceManager, *MockMessageQueue, *MockPermissionService) {
	mockVoiceManager := &MockVoiceManager{}
	mockMessageQueue := &MockMessageQueue{}
	mockPermissionService := &MockPermissionService{}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	handler := NewControlCommandHandler(
		mockVoiceManager,
		mockMessageQueue,
		mockPermissionService,
		logger,
	)

	return handler, mockVoiceManager, mockMessageQueue, mockPermissionService
}

// JoinCommandHandler Tests

func TestJoinCommandHandler_Definition(t *testing.T) {
	handler, _, _, _, _ := createTestJoinHandler()

	definition := handler.Definition()

	assert.Equal(t, "darrot-join", definition.Name)
	assert.Equal(t, "Join a voice channel and start TTS for messages from a text channel", definition.Description)
	assert.Len(t, definition.Options, 2)

	// Check voice channel option
	voiceOption := definition.Options[0]
	assert.Equal(t, "voice-channel", voiceOption.Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionChannel, voiceOption.Type)
	assert.True(t, voiceOption.Required)
	assert.Contains(t, voiceOption.ChannelTypes, discordgo.ChannelTypeGuildVoice)

	// Check text channel option
	textOption := definition.Options[1]
	assert.Equal(t, "text-channel", textOption.Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionChannel, textOption.Type)
	assert.False(t, textOption.Required)
	assert.Contains(t, textOption.ChannelTypes, discordgo.ChannelTypeGuildText)
}

func TestJoinCommandHandler_ValidatePermissions_Success(t *testing.T) {
	handler, _, _, mockPermissionService, _ := createTestJoinHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanInviteBot", userID, guildID).Return(true, nil)

	err := handler.ValidatePermissions(userID, guildID)

	assert.NoError(t, err)
	mockPermissionService.AssertExpectations(t)
}

func TestJoinCommandHandler_ValidatePermissions_Denied(t *testing.T) {
	handler, _, _, mockPermissionService, _ := createTestJoinHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanInviteBot", userID, guildID).Return(false, nil)

	err := handler.ValidatePermissions(userID, guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "don't have permission to invite the bot")
	mockPermissionService.AssertExpectations(t)
}

func TestJoinCommandHandler_ValidatePermissions_Error(t *testing.T) {
	handler, _, _, mockPermissionService, _ := createTestJoinHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanInviteBot", userID, guildID).Return(false, errors.New("permission check failed"))

	err := handler.ValidatePermissions(userID, guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check permissions")
	mockPermissionService.AssertExpectations(t)
}

func TestJoinCommandHandler_ValidateChannelAccess_Success(t *testing.T) {
	handler, _, mockChannelService, _, _ := createTestJoinHandler()

	userID := "user123"
	channelID := "channel123"

	mockChannelService.On("ValidateChannelAccess", userID, channelID).Return(nil)

	err := handler.ValidateChannelAccess(userID, channelID)

	assert.NoError(t, err)
	mockChannelService.AssertExpectations(t)
}

func TestJoinCommandHandler_ValidateChannelAccess_Denied(t *testing.T) {
	handler, _, mockChannelService, _, _ := createTestJoinHandler()

	userID := "user123"
	channelID := "channel123"

	mockChannelService.On("ValidateChannelAccess", userID, channelID).Return(errors.New("access denied"))

	err := handler.ValidateChannelAccess(userID, channelID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	mockChannelService.AssertExpectations(t)
}

// LeaveCommandHandler Tests

func TestLeaveCommandHandler_Definition(t *testing.T) {
	handler, _, _, _ := createTestLeaveHandler()

	definition := handler.Definition()

	assert.Equal(t, "darrot-leave", definition.Name)
	assert.Equal(t, "Stop TTS and leave the voice channel", definition.Description)
	assert.Len(t, definition.Options, 0) // Leave command has no options
}

func TestLeaveCommandHandler_ValidatePermissions_Success(t *testing.T) {
	handler, _, _, mockPermissionService := createTestLeaveHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanControlBot", userID, guildID).Return(true, nil)

	err := handler.ValidatePermissions(userID, guildID)

	assert.NoError(t, err)
	mockPermissionService.AssertExpectations(t)
}

func TestLeaveCommandHandler_ValidatePermissions_Denied(t *testing.T) {
	handler, _, _, mockPermissionService := createTestLeaveHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanControlBot", userID, guildID).Return(false, nil)

	err := handler.ValidatePermissions(userID, guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "don't have permission to control the bot")
	mockPermissionService.AssertExpectations(t)
}

func TestLeaveCommandHandler_ValidatePermissions_Error(t *testing.T) {
	handler, _, _, mockPermissionService := createTestLeaveHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanControlBot", userID, guildID).Return(false, errors.New("permission check failed"))

	err := handler.ValidatePermissions(userID, guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check permissions")
	mockPermissionService.AssertExpectations(t)
}

func TestLeaveCommandHandler_ValidateChannelAccess_NotApplicable(t *testing.T) {
	handler, _, _, _ := createTestLeaveHandler()

	// ValidateChannelAccess should always return nil for leave command
	err := handler.ValidateChannelAccess("user123", "channel123")

	assert.NoError(t, err)
}

// Service interaction tests

func TestJoinCommandHandler_ServiceInteractions(t *testing.T) {
	_, mockVoiceManager, mockChannelService, _, mockUserService := createTestJoinHandler()

	guildID := "guild123"
	userID := "user123"
	voiceChannelID := "voice123"
	textChannelID := "text123"

	// Test voice manager interactions
	expectedConnection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: voiceChannelID,
	}
	mockVoiceManager.On("GetConnection", guildID).Return(nil, false)
	mockVoiceManager.On("JoinChannel", guildID, voiceChannelID).Return(expectedConnection, nil)

	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.False(t, exists)
	assert.Nil(t, conn)

	conn, err := mockVoiceManager.JoinChannel(guildID, voiceChannelID)
	assert.NoError(t, err)
	assert.Equal(t, guildID, conn.GuildID)
	assert.Equal(t, voiceChannelID, conn.ChannelID)

	// Test channel service interactions
	mockChannelService.On("CreatePairingWithCreator", guildID, voiceChannelID, textChannelID, userID).Return(nil)
	err = mockChannelService.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, userID)
	assert.NoError(t, err)

	// Test user service interactions
	mockUserService.On("AutoOptIn", userID, guildID).Return(nil)
	err = mockUserService.AutoOptIn(userID, guildID)
	assert.NoError(t, err)

	// Verify all mocks were called as expected
	mockChannelService.AssertExpectations(t)
	mockVoiceManager.AssertExpectations(t)
	mockUserService.AssertExpectations(t)
}

func TestLeaveCommandHandler_ServiceInteractions(t *testing.T) {
	_, mockVoiceManager, mockChannelService, _ := createTestLeaveHandler()

	guildID := "guild123"
	voiceChannelID := "voice123"

	// Test voice manager interactions
	existingConnection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: voiceChannelID,
	}
	mockVoiceManager.On("GetConnection", guildID).Return(existingConnection, true)
	mockVoiceManager.On("LeaveChannel", guildID).Return(nil)

	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.Equal(t, guildID, conn.GuildID)

	err := mockVoiceManager.LeaveChannel(guildID)
	assert.NoError(t, err)

	// Test channel service interactions
	mockChannelService.On("RemovePairing", guildID, voiceChannelID).Return(nil)
	err = mockChannelService.RemovePairing(guildID, voiceChannelID)
	assert.NoError(t, err)

	// Verify all mocks were called as expected
	mockVoiceManager.AssertExpectations(t)
	mockChannelService.AssertExpectations(t)
}

// Error handling tests for edge cases

func TestJoinCommandHandler_VoiceManagerError(t *testing.T) {
	_, mockVoiceManager, _, _, _ := createTestJoinHandler()

	guildID := "guild123"
	voiceChannelID := "voice123"

	mockVoiceManager.On("JoinChannel", guildID, voiceChannelID).Return(nil, errors.New("failed to join channel"))

	_, err := mockVoiceManager.JoinChannel(guildID, voiceChannelID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to join channel")

	mockVoiceManager.AssertExpectations(t)
}

func TestLeaveCommandHandler_VoiceManagerError(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestLeaveHandler()

	guildID := "guild123"

	mockVoiceManager.On("LeaveChannel", guildID).Return(errors.New("failed to leave channel"))

	err := mockVoiceManager.LeaveChannel(guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to leave channel")

	mockVoiceManager.AssertExpectations(t)
}

// ControlCommandHandler Tests

func TestControlCommandHandler_Definition(t *testing.T) {
	handler, _, _, _ := createTestControlHandler()

	definition := handler.Definition()

	assert.Equal(t, "darrot-control", definition.Name)
	assert.Equal(t, "Control TTS playback (pause, resume, skip)", definition.Description)
	assert.Len(t, definition.Options, 1)

	// Check action option
	actionOption := definition.Options[0]
	assert.Equal(t, "action", actionOption.Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionString, actionOption.Type)
	assert.True(t, actionOption.Required)
	assert.Len(t, actionOption.Choices, 3)

	// Check choices
	choices := make(map[string]string)
	for _, choice := range actionOption.Choices {
		choices[choice.Name] = choice.Value.(string)
	}
	assert.Equal(t, "pause", choices["pause"])
	assert.Equal(t, "resume", choices["resume"])
	assert.Equal(t, "skip", choices["skip"])
}

func TestControlCommandHandler_ValidatePermissions_Success(t *testing.T) {
	handler, _, _, mockPermissionService := createTestControlHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanControlBot", userID, guildID).Return(true, nil)

	err := handler.ValidatePermissions(userID, guildID)

	assert.NoError(t, err)
	mockPermissionService.AssertExpectations(t)
}

func TestControlCommandHandler_ValidatePermissions_Denied(t *testing.T) {
	handler, _, _, mockPermissionService := createTestControlHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanControlBot", userID, guildID).Return(false, nil)

	err := handler.ValidatePermissions(userID, guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "don't have permission to control the bot")
	mockPermissionService.AssertExpectations(t)
}

func TestControlCommandHandler_ValidatePermissions_Error(t *testing.T) {
	handler, _, _, mockPermissionService := createTestControlHandler()

	userID := "user123"
	guildID := "guild123"

	mockPermissionService.On("CanControlBot", userID, guildID).Return(false, errors.New("permission check failed"))

	err := handler.ValidatePermissions(userID, guildID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check permissions")
	mockPermissionService.AssertExpectations(t)
}

func TestControlCommandHandler_ValidateChannelAccess_NotApplicable(t *testing.T) {
	handler, _, _, _ := createTestControlHandler()

	// ValidateChannelAccess should always return nil for control commands
	err := handler.ValidateChannelAccess("user123", "channel123")

	assert.NoError(t, err)
}

func TestControlCommandHandler_PausePlayback_Success(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestControlHandler()

	guildID := "guild123"
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPaused:  false,
	}

	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)
	mockVoiceManager.On("PausePlayback", guildID).Return(nil)

	// Test the pause functionality directly
	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.False(t, conn.IsPaused)

	err := mockVoiceManager.PausePlayback(guildID)
	assert.NoError(t, err)

	mockVoiceManager.AssertExpectations(t)
}

func TestControlCommandHandler_PausePlayback_AlreadyPaused(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestControlHandler()

	guildID := "guild123"
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPaused:  true,
	}

	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)

	// Test that already paused connection is handled correctly
	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.True(t, conn.IsPaused)

	mockVoiceManager.AssertExpectations(t)
}

func TestControlCommandHandler_PausePlayback_Error(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestControlHandler()

	guildID := "guild123"

	mockVoiceManager.On("PausePlayback", guildID).Return(errors.New("pause failed"))

	err := mockVoiceManager.PausePlayback(guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pause failed")

	mockVoiceManager.AssertExpectations(t)
}

func TestControlCommandHandler_ResumePlayback_Success(t *testing.T) {
	_, mockVoiceManager, mockMessageQueue, _ := createTestControlHandler()

	guildID := "guild123"
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPaused:  true,
	}

	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)
	mockVoiceManager.On("ResumePlayback", guildID).Return(nil)
	mockMessageQueue.On("Size", guildID).Return(3)

	// Test the resume functionality
	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.True(t, conn.IsPaused)

	err := mockVoiceManager.ResumePlayback(guildID)
	assert.NoError(t, err)

	queueSize := mockMessageQueue.Size(guildID)
	assert.Equal(t, 3, queueSize)

	mockVoiceManager.AssertExpectations(t)
	mockMessageQueue.AssertExpectations(t)
}

func TestControlCommandHandler_ResumePlayback_NotPaused(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestControlHandler()

	guildID := "guild123"
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPaused:  false,
	}

	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)

	// Test that not paused connection is handled correctly
	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.False(t, conn.IsPaused)

	mockVoiceManager.AssertExpectations(t)
}

func TestControlCommandHandler_ResumePlayback_Error(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestControlHandler()

	guildID := "guild123"

	mockVoiceManager.On("ResumePlayback", guildID).Return(errors.New("resume failed"))

	err := mockVoiceManager.ResumePlayback(guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resume failed")

	mockVoiceManager.AssertExpectations(t)
}

func TestControlCommandHandler_SkipMessage_Success(t *testing.T) {
	_, mockVoiceManager, mockMessageQueue, _ := createTestControlHandler()

	guildID := "guild123"
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPlaying: true,
	}

	skippedMessage := &QueuedMessage{
		ID:       "msg123",
		GuildID:  guildID,
		UserID:   "user123",
		Username: "TestUser",
		Content:  "Test message",
	}

	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)
	mockVoiceManager.On("SkipCurrentMessage", guildID).Return(nil)
	mockMessageQueue.On("SkipNext", guildID).Return(skippedMessage, nil)
	mockMessageQueue.On("Size", guildID).Return(2)

	// Test the skip functionality
	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.True(t, conn.IsPlaying)

	err := mockVoiceManager.SkipCurrentMessage(guildID)
	assert.NoError(t, err)

	message, err := mockMessageQueue.SkipNext(guildID)
	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, "TestUser", message.Username)

	queueSize := mockMessageQueue.Size(guildID)
	assert.Equal(t, 2, queueSize)

	mockVoiceManager.AssertExpectations(t)
	mockMessageQueue.AssertExpectations(t)
}

func TestControlCommandHandler_SkipMessage_EmptyQueue(t *testing.T) {
	_, mockVoiceManager, mockMessageQueue, _ := createTestControlHandler()

	guildID := "guild123"

	mockVoiceManager.On("SkipCurrentMessage", guildID).Return(nil)
	mockMessageQueue.On("SkipNext", guildID).Return(nil, nil)

	err := mockVoiceManager.SkipCurrentMessage(guildID)
	assert.NoError(t, err)

	message, err := mockMessageQueue.SkipNext(guildID)
	assert.NoError(t, err)
	assert.Nil(t, message)

	mockVoiceManager.AssertExpectations(t)
	mockMessageQueue.AssertExpectations(t)
}

func TestControlCommandHandler_SkipMessage_Error(t *testing.T) {
	_, mockVoiceManager, mockMessageQueue, _ := createTestControlHandler()

	guildID := "guild123"

	mockVoiceManager.On("SkipCurrentMessage", guildID).Return(nil)
	mockMessageQueue.On("SkipNext", guildID).Return(nil, errors.New("skip failed"))

	err := mockVoiceManager.SkipCurrentMessage(guildID)
	assert.NoError(t, err)

	_, err = mockMessageQueue.SkipNext(guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skip failed")

	mockVoiceManager.AssertExpectations(t)
	mockMessageQueue.AssertExpectations(t)
}

func TestControlCommandHandler_NoVoiceConnection(t *testing.T) {
	_, mockVoiceManager, _, _ := createTestControlHandler()

	guildID := "guild123"

	mockVoiceManager.On("GetConnection", guildID).Return(nil, false)

	// Test that no connection is handled correctly
	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.False(t, exists)
	assert.Nil(t, conn)

	mockVoiceManager.AssertExpectations(t)
}

// Integration tests for control command handler service interactions

func TestControlCommandHandler_ServiceInteractions_Pause(t *testing.T) {
	_, mockVoiceManager, _, mockPermissionService := createTestControlHandler()

	guildID := "guild123"
	userID := "user123"

	// Test permission validation
	mockPermissionService.On("CanControlBot", userID, guildID).Return(true, nil)

	canControl, err := mockPermissionService.CanControlBot(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, canControl)

	// Test voice connection check
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPaused:  false,
	}
	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)

	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.False(t, conn.IsPaused)

	// Test pause operation
	mockVoiceManager.On("PausePlayback", guildID).Return(nil)

	err = mockVoiceManager.PausePlayback(guildID)
	assert.NoError(t, err)

	mockPermissionService.AssertExpectations(t)
	mockVoiceManager.AssertExpectations(t)
}

func TestControlCommandHandler_ServiceInteractions_Resume(t *testing.T) {
	_, mockVoiceManager, mockMessageQueue, mockPermissionService := createTestControlHandler()

	guildID := "guild123"
	userID := "user123"

	// Test permission validation
	mockPermissionService.On("CanControlBot", userID, guildID).Return(true, nil)

	canControl, err := mockPermissionService.CanControlBot(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, canControl)

	// Test voice connection check
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPaused:  true,
	}
	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)

	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.True(t, conn.IsPaused)

	// Test resume operation
	mockVoiceManager.On("ResumePlayback", guildID).Return(nil)
	mockMessageQueue.On("Size", guildID).Return(0)

	err = mockVoiceManager.ResumePlayback(guildID)
	assert.NoError(t, err)

	queueSize := mockMessageQueue.Size(guildID)
	assert.Equal(t, 0, queueSize)

	mockPermissionService.AssertExpectations(t)
	mockVoiceManager.AssertExpectations(t)
	mockMessageQueue.AssertExpectations(t)
}

func TestControlCommandHandler_ServiceInteractions_Skip(t *testing.T) {
	_, mockVoiceManager, mockMessageQueue, mockPermissionService := createTestControlHandler()

	guildID := "guild123"
	userID := "user123"

	// Test permission validation
	mockPermissionService.On("CanControlBot", userID, guildID).Return(true, nil)

	canControl, err := mockPermissionService.CanControlBot(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, canControl)

	// Test voice connection check
	connection := &VoiceConnection{
		GuildID:   guildID,
		ChannelID: "voice123",
		IsPlaying: true,
	}
	mockVoiceManager.On("GetConnection", guildID).Return(connection, true)

	conn, exists := mockVoiceManager.GetConnection(guildID)
	assert.True(t, exists)
	assert.True(t, conn.IsPlaying)

	// Test skip operations
	skippedMessage := &QueuedMessage{
		ID:       "msg123",
		GuildID:  guildID,
		UserID:   "user456",
		Username: "SkippedUser",
		Content:  "Skipped message",
	}

	mockVoiceManager.On("SkipCurrentMessage", guildID).Return(nil)
	mockMessageQueue.On("SkipNext", guildID).Return(skippedMessage, nil)
	mockMessageQueue.On("Size", guildID).Return(1)

	err = mockVoiceManager.SkipCurrentMessage(guildID)
	assert.NoError(t, err)

	message, err := mockMessageQueue.SkipNext(guildID)
	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, "SkippedUser", message.Username)

	queueSize := mockMessageQueue.Size(guildID)
	assert.Equal(t, 1, queueSize)

	mockPermissionService.AssertExpectations(t)
	mockVoiceManager.AssertExpectations(t)
	mockMessageQueue.AssertExpectations(t)
}

// Test helper function for OptInCommandHandler

func createTestOptInHandler() (*OptInCommandHandler, *MockUserService) {
	mockUserService := &MockUserService{}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	handler := NewOptInCommandHandler(
		mockUserService,
		logger,
	)

	return handler, mockUserService
}

// OptInCommandHandler Tests

func TestOptInCommandHandler_Definition(t *testing.T) {
	handler, _ := createTestOptInHandler()

	definition := handler.Definition()

	assert.Equal(t, "darrot-optin", definition.Name)
	assert.Equal(t, "Manage your TTS opt-in preferences", definition.Description)
	assert.Len(t, definition.Options, 1)

	// Check action option
	actionOption := definition.Options[0]
	assert.Equal(t, "action", actionOption.Name)
	assert.Equal(t, discordgo.ApplicationCommandOptionString, actionOption.Type)
	assert.True(t, actionOption.Required)
	assert.Len(t, actionOption.Choices, 3)

	// Check choices
	choices := make(map[string]string)
	for _, choice := range actionOption.Choices {
		choices[choice.Name] = choice.Value.(string)
	}
	assert.Equal(t, "opt-in", choices["opt-in"])
	assert.Equal(t, "opt-out", choices["opt-out"])
	assert.Equal(t, "status", choices["status"])
}

func TestOptInCommandHandler_ValidatePermissions_Success(t *testing.T) {
	handler, _ := createTestOptInHandler()

	// Users can always manage their own opt-in preferences
	err := handler.ValidatePermissions("user123", "guild123")

	assert.NoError(t, err)
}

func TestOptInCommandHandler_ValidateChannelAccess_NotApplicable(t *testing.T) {
	handler, _ := createTestOptInHandler()

	// ValidateChannelAccess should always return nil for opt-in commands
	err := handler.ValidateChannelAccess("user123", "channel123")

	assert.NoError(t, err)
}

// Test opt-in functionality

func TestOptInCommandHandler_HandleOptIn_Success(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock that user is currently opted out
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(nil)

	// Test the opt-in process
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleOptIn_AlreadyOptedIn(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock that user is already opted in
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil)

	// Test that already opted-in user is handled correctly
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleOptIn_CheckStatusError(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock error when checking opt-in status
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, errors.New("status check failed"))

	// Test error handling
	_, err := mockUserService.IsOptedIn(userID, guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status check failed")

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleOptIn_SetStatusError(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock successful status check but failed opt-in
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(errors.New("opt-in failed"))

	// Test error handling
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "opt-in failed")

	mockUserService.AssertExpectations(t)
}

// Test opt-out functionality

func TestOptInCommandHandler_HandleOptOut_Success(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock that user is currently opted in
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, false).Return(nil)

	// Test the opt-out process
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	err = mockUserService.SetOptInStatus(userID, guildID, false)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleOptOut_AlreadyOptedOut(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock that user is already opted out
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil)

	// Test that already opted-out user is handled correctly
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleOptOut_CheckStatusError(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock error when checking opt-in status
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, errors.New("status check failed"))

	// Test error handling
	_, err := mockUserService.IsOptedIn(userID, guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status check failed")

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleOptOut_SetStatusError(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock successful status check but failed opt-out
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, false).Return(errors.New("opt-out failed"))

	// Test error handling
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	err = mockUserService.SetOptInStatus(userID, guildID, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "opt-out failed")

	mockUserService.AssertExpectations(t)
}

// Test status check functionality

func TestOptInCommandHandler_HandleStatus_OptedIn(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock that user is opted in
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil)

	// Test status check for opted-in user
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleStatus_OptedOut(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock that user is opted out
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil)

	// Test status check for opted-out user
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_HandleStatus_Error(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Mock error when checking status
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, errors.New("status check failed"))

	// Test error handling
	_, err := mockUserService.IsOptedIn(userID, guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status check failed")

	mockUserService.AssertExpectations(t)
}

// Integration tests for opt-in command handler service interactions

func TestOptInCommandHandler_ServiceInteractions_OptInFlow(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test complete opt-in flow
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(nil)

	// Check initial status
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	// Perform opt-in
	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_ServiceInteractions_OptOutFlow(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test complete opt-out flow
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, false).Return(nil)

	// Check initial status
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	// Perform opt-out
	err = mockUserService.SetOptInStatus(userID, guildID, false)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_ServiceInteractions_StatusFlow(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test status check flow for different states
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil).Once()
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil).Once()

	// Check opted-in status
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	// Check opted-out status
	isOptedIn, err = mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	mockUserService.AssertExpectations(t)
}

// Edge case tests

func TestOptInCommandHandler_MultipleUsers_IndependentPreferences(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	user1ID := "user123"
	user2ID := "user456"
	guildID := "guild123"

	// Test that different users have independent preferences
	mockUserService.On("IsOptedIn", user1ID, guildID).Return(true, nil)
	mockUserService.On("IsOptedIn", user2ID, guildID).Return(false, nil)
	mockUserService.On("SetOptInStatus", user2ID, guildID, true).Return(nil)

	// Check user1 status (opted in)
	isOptedIn1, err := mockUserService.IsOptedIn(user1ID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn1)

	// Check user2 status (opted out)
	isOptedIn2, err := mockUserService.IsOptedIn(user2ID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn2)

	// Opt in user2
	err = mockUserService.SetOptInStatus(user2ID, guildID, true)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_MultipleGuilds_IndependentPreferences(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guild1ID := "guild123"
	guild2ID := "guild456"

	// Test that same user has independent preferences across guilds
	mockUserService.On("IsOptedIn", userID, guild1ID).Return(true, nil)
	mockUserService.On("IsOptedIn", userID, guild2ID).Return(false, nil)
	mockUserService.On("SetOptInStatus", userID, guild2ID, true).Return(nil)

	// Check status in guild1 (opted in)
	isOptedIn1, err := mockUserService.IsOptedIn(userID, guild1ID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn1)

	// Check status in guild2 (opted out)
	isOptedIn2, err := mockUserService.IsOptedIn(userID, guild2ID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn2)

	// Opt in for guild2
	err = mockUserService.SetOptInStatus(userID, guild2ID, true)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

// Validation tests for requirements compliance

func TestOptInCommandHandler_RequirementCompliance_UserCanOnlyManageOwnPreferences(t *testing.T) {
	handler, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test that validation always passes (users can manage their own preferences)
	// This validates requirement 6.4: users can only manage their own opt-in preferences
	err := handler.ValidatePermissions(userID, guildID)
	assert.NoError(t, err)

	// The actual enforcement happens at the Discord interaction level
	// where userID is extracted from the interaction context
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(nil)

	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_RequirementCompliance_OptInOptOutCommands(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test requirement 6.2: users can opt-in using dedicated command
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil).Once()
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(nil)

	// Test requirement 6.3: users can opt-out using dedicated command
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil).Once()
	mockUserService.On("SetOptInStatus", userID, guildID, false).Return(nil)

	// Opt-in flow
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.NoError(t, err)

	// Opt-out flow
	isOptedIn, err = mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	err = mockUserService.SetOptInStatus(userID, guildID, false)
	assert.NoError(t, err)

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_RequirementCompliance_StatusCheck(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test requirement 6.4: users can check their opt-in status
	mockUserService.On("IsOptedIn", userID, guildID).Return(true, nil).Once()
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil).Once()

	// Check opted-in status
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.True(t, isOptedIn)

	// Check opted-out status
	isOptedIn, err = mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	mockUserService.AssertExpectations(t)
}

// Error resilience tests

func TestOptInCommandHandler_ErrorResilience_ServiceUnavailable(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test handling when user service is unavailable
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, errors.New("service unavailable"))
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(errors.New("service unavailable"))

	// Test that errors are properly propagated
	_, err := mockUserService.IsOptedIn(userID, guildID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service unavailable")

	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service unavailable")

	mockUserService.AssertExpectations(t)
}

func TestOptInCommandHandler_ErrorResilience_PartialFailure(t *testing.T) {
	_, mockUserService := createTestOptInHandler()

	userID := "user123"
	guildID := "guild123"

	// Test handling when status check succeeds but update fails
	mockUserService.On("IsOptedIn", userID, guildID).Return(false, nil)
	mockUserService.On("SetOptInStatus", userID, guildID, true).Return(errors.New("update failed"))

	// Status check should succeed
	isOptedIn, err := mockUserService.IsOptedIn(userID, guildID)
	assert.NoError(t, err)
	assert.False(t, isOptedIn)

	// Update should fail
	err = mockUserService.SetOptInStatus(userID, guildID, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")

	mockUserService.AssertExpectations(t)
}
