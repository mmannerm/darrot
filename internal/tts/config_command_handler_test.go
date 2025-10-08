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

// MockConfigService implements ConfigService for testing
type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetGuildConfig(guildID string) (*GuildTTSConfig, error) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GuildTTSConfig), args.Error(1)
}

func (m *MockConfigService) SetGuildConfig(guildID string, config *GuildTTSConfig) error {
	args := m.Called(guildID, config)
	return args.Error(0)
}

func (m *MockConfigService) SetRequiredRoles(guildID string, roleIDs []string) error {
	args := m.Called(guildID, roleIDs)
	return args.Error(0)
}

func (m *MockConfigService) GetRequiredRoles(guildID string) ([]string, error) {
	args := m.Called(guildID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigService) SetTTSSettings(guildID string, settings TTSConfig) error {
	args := m.Called(guildID, settings)
	return args.Error(0)
}

func (m *MockConfigService) GetTTSSettings(guildID string) (*TTSConfig, error) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TTSConfig), args.Error(1)
}

func (m *MockConfigService) SetMaxQueueSize(guildID string, size int) error {
	args := m.Called(guildID, size)
	return args.Error(0)
}

func (m *MockConfigService) GetMaxQueueSize(guildID string) (int, error) {
	args := m.Called(guildID)
	return args.Int(0), args.Error(1)
}

func (m *MockConfigService) ValidateConfig(config *GuildTTSConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

// Test helper functions

func createTestConfigHandler() (*ConfigCommandHandler, *MockConfigService, *MockPermissionService, *MockTTSManager, *MockMessageQueue) {
	mockConfigService := &MockConfigService{}
	mockPermissionService := &MockPermissionService{}
	mockTTSManager := &MockTTSManager{}
	mockMessageQueue := &MockMessageQueue{}
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)

	handler := NewConfigCommandHandler(
		mockConfigService,
		mockPermissionService,
		mockTTSManager,
		mockMessageQueue,
		logger,
	)

	return handler, mockConfigService, mockPermissionService, mockTTSManager, mockMessageQueue
}

func createTestInteraction(guildID, userID string, options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			GuildID: guildID,
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: userID,
				},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Options: options,
			},
		},
	}
}

// Tests for ConfigCommandHandler

func TestConfigCommandHandler_Definition(t *testing.T) {
	handler, _, _, _, _ := createTestConfigHandler()

	definition := handler.Definition()

	assert.Equal(t, "darrot-config", definition.Name)
	assert.Equal(t, "Configure TTS settings for this server (Administrator only)", definition.Description)
	assert.Len(t, definition.Options, 4) // roles, voice, queue, show subcommands

	// Check subcommands exist
	subcommandNames := make(map[string]bool)
	for _, option := range definition.Options {
		subcommandNames[option.Name] = true
	}
	assert.True(t, subcommandNames["roles"])
	assert.True(t, subcommandNames["voice"])
	assert.True(t, subcommandNames["queue"])
	assert.True(t, subcommandNames["show"])
}

func TestConfigCommandHandler_ValidatePermissions(t *testing.T) {
	handler, _, mockPermissionService, _, _ := createTestConfigHandler()

	tests := []struct {
		name          string
		userID        string
		guildID       string
		canControl    bool
		permissionErr error
		expectedError string
	}{
		{
			name:       "valid admin permissions",
			userID:     "user123",
			guildID:    "guild123",
			canControl: true,
		},
		{
			name:          "no admin permissions",
			userID:        "user123",
			guildID:       "guild123",
			canControl:    false,
			expectedError: "you must have administrator permissions to configure TTS settings",
		},
		{
			name:          "permission check error",
			userID:        "user123",
			guildID:       "guild123",
			permissionErr: errors.New("permission service error"),
			expectedError: "failed to check permissions: permission service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPermissionService.On("CanControlBot", tt.userID, tt.guildID).Return(tt.canControl, tt.permissionErr).Once()

			err := handler.ValidatePermissions(tt.userID, tt.guildID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockPermissionService.AssertExpectations(t)
		})
	}
}

func TestConfigCommandHandler_GetRequiredRoles(t *testing.T) {
	_, mockConfigService, _, _, _ := createTestConfigHandler()

	// Test with no roles configured
	t.Run("no roles configured", func(t *testing.T) {
		mockConfigService.On("GetRequiredRoles", "guild123").Return([]string{}, nil).Once()

		roles, err := mockConfigService.GetRequiredRoles("guild123")

		assert.NoError(t, err)
		assert.Empty(t, roles)
		mockConfigService.AssertExpectations(t)
	})

	// Test with roles configured
	t.Run("roles configured", func(t *testing.T) {
		expectedRoles := []string{"role1", "role2"}
		mockConfigService.On("GetRequiredRoles", "guild123").Return(expectedRoles, nil).Once()

		roles, err := mockConfigService.GetRequiredRoles("guild123")

		assert.NoError(t, err)
		assert.Equal(t, expectedRoles, roles)
		mockConfigService.AssertExpectations(t)
	})

	// Test with service error
	t.Run("service error", func(t *testing.T) {
		mockConfigService.On("GetRequiredRoles", "guild123").Return([]string{}, errors.New("service error")).Once()

		roles, err := mockConfigService.GetRequiredRoles("guild123")

		assert.Error(t, err)
		assert.Empty(t, roles)
		assert.Contains(t, err.Error(), "service error")
		mockConfigService.AssertExpectations(t)
	})
}

func TestConfigCommandHandler_TTSSettings(t *testing.T) {
	_, mockConfigService, _, mockTTSManager, _ := createTestConfigHandler()

	// Test getting TTS settings
	t.Run("get TTS settings", func(t *testing.T) {
		expectedConfig := &TTSConfig{
			Voice:  "voice1",
			Speed:  1.0,
			Volume: 0.8,
		}
		mockConfigService.On("GetTTSSettings", "guild123").Return(expectedConfig, nil).Once()

		config, err := mockConfigService.GetTTSSettings("guild123")

		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
		mockConfigService.AssertExpectations(t)
	})

	// Test setting TTS settings
	t.Run("set TTS settings", func(t *testing.T) {
		newConfig := TTSConfig{
			Voice:  "voice2",
			Speed:  1.5,
			Volume: 0.9,
		}
		mockConfigService.On("SetTTSSettings", "guild123", newConfig).Return(nil).Once()
		mockTTSManager.On("SetVoiceConfig", "guild123", newConfig).Return(nil).Once()

		err := mockConfigService.SetTTSSettings("guild123", newConfig)
		assert.NoError(t, err)

		err = mockTTSManager.SetVoiceConfig("guild123", newConfig)
		assert.NoError(t, err)

		mockConfigService.AssertExpectations(t)
		mockTTSManager.AssertExpectations(t)
	})
}

func TestConfigCommandHandler_QueueConfig(t *testing.T) {
	_, mockConfigService, _, _, mockMessageQueue := createTestConfigHandler()

	// Test getting max queue size
	t.Run("get max queue size", func(t *testing.T) {
		mockConfigService.On("GetMaxQueueSize", "guild123").Return(10, nil).Once()

		size, err := mockConfigService.GetMaxQueueSize("guild123")

		assert.NoError(t, err)
		assert.Equal(t, 10, size)
		mockConfigService.AssertExpectations(t)
	})

	// Test setting max queue size
	t.Run("set max queue size", func(t *testing.T) {
		mockConfigService.On("SetMaxQueueSize", "guild123", 15).Return(nil).Once()
		mockMessageQueue.On("SetMaxSize", "guild123", 15).Return(nil).Once()

		err := mockConfigService.SetMaxQueueSize("guild123", 15)
		assert.NoError(t, err)

		err = mockMessageQueue.SetMaxSize("guild123", 15)
		assert.NoError(t, err)

		mockConfigService.AssertExpectations(t)
		mockMessageQueue.AssertExpectations(t)
	})

	// Test current queue size
	t.Run("get current queue size", func(t *testing.T) {
		mockMessageQueue.On("Size", "guild123").Return(5).Once()

		size := mockMessageQueue.Size("guild123")

		assert.Equal(t, 5, size)
		mockMessageQueue.AssertExpectations(t)
	})
}

func TestConfigCommandHandler_GuildConfig(t *testing.T) {
	_, mockConfigService, _, _, _ := createTestConfigHandler()

	// Test getting guild config
	t.Run("get guild config", func(t *testing.T) {
		expectedConfig := &GuildTTSConfig{
			GuildID:       "guild123",
			RequiredRoles: []string{"role1"},
			TTSSettings: TTSConfig{
				Voice:  "voice1",
				Speed:  1.0,
				Volume: 0.8,
			},
			MaxQueueSize: 10,
		}
		mockConfigService.On("GetGuildConfig", "guild123").Return(expectedConfig, nil).Once()

		config, err := mockConfigService.GetGuildConfig("guild123")

		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
		assert.Equal(t, "guild123", config.GuildID)
		assert.Len(t, config.RequiredRoles, 1)
		assert.Equal(t, "role1", config.RequiredRoles[0])
		mockConfigService.AssertExpectations(t)
	})

	// Test setting guild config
	t.Run("set guild config", func(t *testing.T) {
		newConfig := &GuildTTSConfig{
			GuildID:       "guild123",
			RequiredRoles: []string{"role1", "role2"},
			TTSSettings: TTSConfig{
				Voice:  "voice2",
				Speed:  1.5,
				Volume: 0.9,
			},
			MaxQueueSize: 15,
		}
		mockConfigService.On("SetGuildConfig", "guild123", newConfig).Return(nil).Once()

		err := mockConfigService.SetGuildConfig("guild123", newConfig)

		assert.NoError(t, err)
		mockConfigService.AssertExpectations(t)
	})
}

func TestParseFloat32(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    float32
		expectError bool
	}{
		{
			name:     "valid float",
			input:    "1.5",
			expected: 1.5,
		},
		{
			name:     "valid integer",
			input:    "2",
			expected: 2.0,
		},
		{
			name:        "invalid string",
			input:       "not_a_number",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFloat32(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
