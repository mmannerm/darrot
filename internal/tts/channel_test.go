package tts

import (
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockChannelPermissionService is a mock implementation of PermissionService for testing
type MockChannelPermissionService struct {
	mock.Mock
}

func (m *MockChannelPermissionService) CanInviteBot(userID, guildID string) (bool, error) {
	args := m.Called(userID, guildID)
	return args.Bool(0), args.Error(1)
}

func (m *MockChannelPermissionService) CanControlBot(userID, guildID string) (bool, error) {
	args := m.Called(userID, guildID)
	return args.Bool(0), args.Error(1)
}

func (m *MockChannelPermissionService) HasChannelAccess(userID, channelID string) (bool, error) {
	args := m.Called(userID, channelID)
	return args.Bool(0), args.Error(1)
}

func (m *MockChannelPermissionService) SetRequiredRoles(guildID string, roleIDs []string) error {
	args := m.Called(guildID, roleIDs)
	return args.Error(0)
}

func (m *MockChannelPermissionService) GetRequiredRoles(guildID string) ([]string, error) {
	args := m.Called(guildID)
	return args.Get(0).([]string), args.Error(1)
}

func setupChannelServiceTest(t *testing.T) (*ChannelServiceImpl, *StorageService, *MockDiscordSession, *MockChannelPermissionService, string) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "channel_service_test_*")
	require.NoError(t, err)

	// Create storage service
	storage, err := NewStorageService(tempDir)
	require.NoError(t, err)

	// Create mock services
	mockSession := NewMockDiscordSession()
	mockPermissionService := &MockChannelPermissionService{}

	// Create channel service
	channelService := NewChannelService(storage, mockSession, mockPermissionService)

	return channelService, storage, mockSession, mockPermissionService, tempDir
}

func cleanupChannelServiceTest(tempDir string) {
	os.RemoveAll(tempDir)
}

func TestNewChannelService(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	assert.NotNil(t, channelService)
	assert.NotNil(t, channelService.storage)
	assert.NotNil(t, channelService.session)
	assert.NotNil(t, channelService.permissionService)
}

func TestCreatePairing_Success(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel)

	// Create pairing
	err := channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
	assert.NoError(t, err)

	// Verify pairing was created
	pairing, err := channelService.GetPairing(guildID, voiceChannelID)
	assert.NoError(t, err)
	assert.Equal(t, guildID, pairing.GuildID)
	assert.Equal(t, voiceChannelID, pairing.VoiceChannelID)
	assert.Equal(t, textChannelID, pairing.TextChannelID)
}

func TestCreatePairing_InvalidInput(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	tests := []struct {
		name           string
		guildID        string
		voiceChannelID string
		textChannelID  string
		expectedError  string
	}{
		{
			name:           "empty guild ID",
			guildID:        "",
			voiceChannelID: "voice123",
			textChannelID:  "text456",
			expectedError:  "guild ID is required",
		},
		{
			name:           "empty voice channel ID",
			guildID:        "guild123",
			voiceChannelID: "",
			textChannelID:  "text456",
			expectedError:  "voice channel ID is required",
		},
		{
			name:           "empty text channel ID",
			guildID:        "guild123",
			voiceChannelID: "voice123",
			textChannelID:  "",
			expectedError:  "text channel ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channelService.CreatePairing(tt.guildID, tt.voiceChannelID, tt.textChannelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCreatePairing_VoiceChannelAlreadyPaired(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID1 := "text789"
	textChannelID2 := "text101"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel1 := &discordgo.Channel{
		ID:      textChannelID1,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}
	textChannel2 := &discordgo.Channel{
		ID:      textChannelID2,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel1)
	mockSession.AddChannel(textChannel2)

	// Create first pairing
	err := channelService.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID1, "user123")
	assert.NoError(t, err)

	// Try to create second pairing with same voice channel
	err = channelService.CreatePairing(guildID, voiceChannelID, textChannelID2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "voice channel voice456 is already paired")
}

func TestCreatePairing_TextChannelAlreadyPaired(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID1 := "voice456"
	voiceChannelID2 := "voice789"
	textChannelID := "text101"

	// Setup mock channels
	voiceChannel1 := &discordgo.Channel{
		ID:      voiceChannelID1,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	voiceChannel2 := &discordgo.Channel{
		ID:      voiceChannelID2,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel1)
	mockSession.AddChannel(voiceChannel2)
	mockSession.AddChannel(textChannel)

	// Create first pairing
	err := channelService.CreatePairingWithCreator(guildID, voiceChannelID1, textChannelID, "user123")
	assert.NoError(t, err)

	// Try to create second pairing with same text channel
	err = channelService.CreatePairing(guildID, voiceChannelID2, textChannelID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "text channel text101 is already paired")
}

func TestCreatePairing_InvalidChannelTypes(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"

	tests := []struct {
		name          string
		voiceChannel  *discordgo.Channel
		textChannel   *discordgo.Channel
		expectedError string
	}{
		{
			name: "voice channel is text channel",
			voiceChannel: &discordgo.Channel{
				ID:      voiceChannelID,
				GuildID: guildID,
				Type:    discordgo.ChannelTypeGuildText,
			},
			textChannel: &discordgo.Channel{
				ID:      textChannelID,
				GuildID: guildID,
				Type:    discordgo.ChannelTypeGuildText,
			},
			expectedError: "is not a voice channel",
		},
		{
			name: "text channel is voice channel",
			voiceChannel: &discordgo.Channel{
				ID:      voiceChannelID,
				GuildID: guildID,
				Type:    discordgo.ChannelTypeGuildVoice,
			},
			textChannel: &discordgo.Channel{
				ID:      textChannelID,
				GuildID: guildID,
				Type:    discordgo.ChannelTypeGuildVoice,
			},
			expectedError: "is not a text channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous channels
			mockSession = NewMockDiscordSession()
			channelService.session = mockSession

			mockSession.AddChannel(tt.voiceChannel)
			mockSession.AddChannel(tt.textChannel)

			err := channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestRemovePairing_Success(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel)

	// Create pairing first
	err := channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
	assert.NoError(t, err)

	// Remove pairing
	err = channelService.RemovePairing(guildID, voiceChannelID)
	assert.NoError(t, err)

	// Verify pairing is removed
	_, err = channelService.GetPairing(guildID, voiceChannelID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel pairing not found")
}

func TestRemovePairing_InvalidInput(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	tests := []struct {
		name           string
		guildID        string
		voiceChannelID string
		expectedError  string
	}{
		{
			name:           "empty guild ID",
			guildID:        "",
			voiceChannelID: "voice123",
			expectedError:  "guild ID is required",
		},
		{
			name:           "empty voice channel ID",
			guildID:        "guild123",
			voiceChannelID: "",
			expectedError:  "voice channel ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channelService.RemovePairing(tt.guildID, tt.voiceChannelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestRemovePairing_NotFound(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"

	err := channelService.RemovePairing(guildID, voiceChannelID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel pairing not found")
}

func TestGetPairing_Success(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel)

	// Create pairing first
	err := channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
	assert.NoError(t, err)

	// Get pairing
	pairing, err := channelService.GetPairing(guildID, voiceChannelID)
	assert.NoError(t, err)
	assert.NotNil(t, pairing)
	assert.Equal(t, guildID, pairing.GuildID)
	assert.Equal(t, voiceChannelID, pairing.VoiceChannelID)
	assert.Equal(t, textChannelID, pairing.TextChannelID)
	assert.False(t, pairing.CreatedAt.IsZero())
}

func TestGetPairing_InvalidInput(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	tests := []struct {
		name           string
		guildID        string
		voiceChannelID string
		expectedError  string
	}{
		{
			name:           "empty guild ID",
			guildID:        "",
			voiceChannelID: "voice123",
			expectedError:  "guild ID is required",
		},
		{
			name:           "empty voice channel ID",
			guildID:        "guild123",
			voiceChannelID: "",
			expectedError:  "voice channel ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := channelService.GetPairing(tt.guildID, tt.voiceChannelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestGetPairing_NotFound(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"

	_, err := channelService.GetPairing(guildID, voiceChannelID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel pairing not found")
}

func TestValidateChannelAccess_Success(t *testing.T) {
	channelService, _, _, mockPermissionService, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	userID := "user123"
	channelID := "channel456"

	mockPermissionService.On("HasChannelAccess", userID, channelID).Return(true, nil)

	err := channelService.ValidateChannelAccess(userID, channelID)
	assert.NoError(t, err)

	mockPermissionService.AssertExpectations(t)
}

func TestValidateChannelAccess_NoAccess(t *testing.T) {
	channelService, _, _, mockPermissionService, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	userID := "user123"
	channelID := "channel456"

	mockPermissionService.On("HasChannelAccess", userID, channelID).Return(false, nil)

	err := channelService.ValidateChannelAccess(userID, channelID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have access to channel")

	mockPermissionService.AssertExpectations(t)
}

func TestValidateChannelAccess_InvalidInput(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	tests := []struct {
		name          string
		userID        string
		channelID     string
		expectedError string
	}{
		{
			name:          "empty user ID",
			userID:        "",
			channelID:     "channel123",
			expectedError: "user ID is required",
		},
		{
			name:          "empty channel ID",
			userID:        "user123",
			channelID:     "",
			expectedError: "channel ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := channelService.ValidateChannelAccess(tt.userID, tt.channelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestIsChannelPaired_True(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel)

	// Create pairing first
	err := channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
	assert.NoError(t, err)

	// Check if channel is paired
	isPaired := channelService.IsChannelPaired(guildID, textChannelID)
	assert.True(t, isPaired)
}

func TestIsChannelPaired_False(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	textChannelID := "text789"

	isPaired := channelService.IsChannelPaired(guildID, textChannelID)
	assert.False(t, isPaired)
}

func TestIsChannelPaired_InvalidInput(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	// Test empty guild ID
	isPaired := channelService.IsChannelPaired("", "text123")
	assert.False(t, isPaired)

	// Test empty text channel ID
	isPaired = channelService.IsChannelPaired("guild123", "")
	assert.False(t, isPaired)
}

func TestSetPairingCreator_Success(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"
	creatorID := "user101"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel)

	// Create pairing first
	err := channelService.CreatePairing(guildID, voiceChannelID, textChannelID)
	assert.NoError(t, err)

	// Set creator
	err = channelService.SetPairingCreator(guildID, voiceChannelID, creatorID)
	assert.NoError(t, err)

	// Verify creator was set
	pairing, err := channelService.GetPairing(guildID, voiceChannelID)
	assert.NoError(t, err)
	assert.Equal(t, creatorID, pairing.CreatedBy)
}

func TestListGuildPairings_Success(t *testing.T) {
	channelService, _, mockSession, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID1 := "voice456"
	voiceChannelID2 := "voice789"
	textChannelID1 := "text101"
	textChannelID2 := "text202"

	// Setup mock channels
	voiceChannel1 := &discordgo.Channel{
		ID:      voiceChannelID1,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	voiceChannel2 := &discordgo.Channel{
		ID:      voiceChannelID2,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel1 := &discordgo.Channel{
		ID:      textChannelID1,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}
	textChannel2 := &discordgo.Channel{
		ID:      textChannelID2,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel1)
	mockSession.AddChannel(voiceChannel2)
	mockSession.AddChannel(textChannel1)
	mockSession.AddChannel(textChannel2)

	// Create multiple pairings
	err := channelService.CreatePairing(guildID, voiceChannelID1, textChannelID1)
	assert.NoError(t, err)

	err = channelService.CreatePairing(guildID, voiceChannelID2, textChannelID2)
	assert.NoError(t, err)

	// List pairings
	pairings, err := channelService.ListGuildPairings(guildID)
	assert.NoError(t, err)
	assert.Len(t, pairings, 2)

	// Verify pairings
	pairingMap := make(map[string]*ChannelPairing)
	for _, pairing := range pairings {
		pairingMap[pairing.VoiceChannelID] = pairing
	}

	assert.Contains(t, pairingMap, voiceChannelID1)
	assert.Contains(t, pairingMap, voiceChannelID2)
	assert.Equal(t, textChannelID1, pairingMap[voiceChannelID1].TextChannelID)
	assert.Equal(t, textChannelID2, pairingMap[voiceChannelID2].TextChannelID)
}

func TestListGuildPairings_EmptyGuildID(t *testing.T) {
	channelService, _, _, _, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	_, err := channelService.ListGuildPairings("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "guild ID is required")
}

// Integration test to verify the complete workflow
func TestChannelService_IntegrationWorkflow(t *testing.T) {
	channelService, _, mockSession, mockPermissionService, tempDir := setupChannelServiceTest(t)
	defer cleanupChannelServiceTest(tempDir)

	guildID := "guild123"
	voiceChannelID := "voice456"
	textChannelID := "text789"
	userID := "user101"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:      voiceChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildVoice,
	}
	textChannel := &discordgo.Channel{
		ID:      textChannelID,
		GuildID: guildID,
		Type:    discordgo.ChannelTypeGuildText,
	}

	mockSession.AddChannel(voiceChannel)
	mockSession.AddChannel(textChannel)
	mockPermissionService.On("HasChannelAccess", userID, voiceChannelID).Return(true, nil)
	mockPermissionService.On("HasChannelAccess", userID, textChannelID).Return(true, nil)

	// Step 1: Validate channel access
	err := channelService.ValidateChannelAccess(userID, voiceChannelID)
	assert.NoError(t, err)

	err = channelService.ValidateChannelAccess(userID, textChannelID)
	assert.NoError(t, err)

	// Step 2: Verify text channel is not already paired
	isPaired := channelService.IsChannelPaired(guildID, textChannelID)
	assert.False(t, isPaired)

	// Step 3: Create pairing with creator
	err = channelService.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, userID)
	assert.NoError(t, err)

	// Step 4: Verify pairing exists
	pairing, err := channelService.GetPairing(guildID, voiceChannelID)
	assert.NoError(t, err)
	assert.Equal(t, userID, pairing.CreatedBy)

	// Step 5: Verify text channel is now paired
	isPaired = channelService.IsChannelPaired(guildID, textChannelID)
	assert.True(t, isPaired)

	// Step 6: List guild pairings
	pairings, err := channelService.ListGuildPairings(guildID)
	assert.NoError(t, err)
	assert.Len(t, pairings, 1)

	// Step 7: Remove pairing
	err = channelService.RemovePairing(guildID, voiceChannelID)
	assert.NoError(t, err)

	// Step 8: Verify pairing is removed
	_, err = channelService.GetPairing(guildID, voiceChannelID)
	assert.Error(t, err)

	// Step 9: Verify text channel is no longer paired
	isPaired = channelService.IsChannelPaired(guildID, textChannelID)
	assert.False(t, isPaired)

	mockPermissionService.AssertExpectations(t)
}
