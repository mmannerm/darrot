package tts

import (
	"log"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// MockDiscordSession provides a mock implementation of Discord session for testing
type MockDiscordSession struct {
	guilds       map[string]*discordgo.Guild
	members      map[string]map[string]*discordgo.Member // guildID -> userID -> member
	channels     map[string]*discordgo.Channel
	permissions  map[string]map[string]int64 // channelID -> userID -> permissions
	shouldError  bool
	errorMessage string
}

// Ensure MockDiscordSession implements DiscordSession interface
var _ DiscordSession = (*MockDiscordSession)(nil)

// NewMockDiscordSession creates a new mock Discord session
func NewMockDiscordSession() *MockDiscordSession {
	return &MockDiscordSession{
		guilds:      make(map[string]*discordgo.Guild),
		members:     make(map[string]map[string]*discordgo.Member),
		channels:    make(map[string]*discordgo.Channel),
		permissions: make(map[string]map[string]int64),
	}
}

// Guild mocks the Discord session Guild method
func (m *MockDiscordSession) Guild(guildID string) (*discordgo.Guild, error) {
	if m.shouldError {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Message: m.errorMessage}}
	}

	guild, exists := m.guilds[guildID]
	if !exists {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: discordgo.ErrCodeUnknownGuild}}
	}
	return guild, nil
}

// GuildMember mocks the Discord session GuildMember method
func (m *MockDiscordSession) GuildMember(guildID, userID string) (*discordgo.Member, error) {
	if m.shouldError {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Message: m.errorMessage}}
	}

	guildMembers, exists := m.members[guildID]
	if !exists {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: discordgo.ErrCodeUnknownMember}}
	}

	member, exists := guildMembers[userID]
	if !exists {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: discordgo.ErrCodeUnknownMember}}
	}

	return member, nil
}

// Channel mocks the Discord session Channel method
func (m *MockDiscordSession) Channel(channelID string) (*discordgo.Channel, error) {
	if m.shouldError {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Message: m.errorMessage}}
	}

	channel, exists := m.channels[channelID]
	if !exists {
		return nil, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: discordgo.ErrCodeUnknownChannel}}
	}
	return channel, nil
}

// UserChannelPermissions mocks the Discord session UserChannelPermissions method
func (m *MockDiscordSession) UserChannelPermissions(userID, channelID string) (int64, error) {
	if m.shouldError {
		return 0, &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Message: m.errorMessage}}
	}

	channelPerms, exists := m.permissions[channelID]
	if !exists {
		return 0, nil
	}

	perms, exists := channelPerms[userID]
	if !exists {
		return 0, nil
	}

	return perms, nil
}

// Helper methods for setting up mock data

func (m *MockDiscordSession) AddGuild(guild *discordgo.Guild) {
	m.guilds[guild.ID] = guild
	if m.members[guild.ID] == nil {
		m.members[guild.ID] = make(map[string]*discordgo.Member)
	}
}

func (m *MockDiscordSession) AddMember(guildID string, member *discordgo.Member) {
	if m.members[guildID] == nil {
		m.members[guildID] = make(map[string]*discordgo.Member)
	}
	m.members[guildID][member.User.ID] = member
}

func (m *MockDiscordSession) AddChannel(channel *discordgo.Channel) {
	m.channels[channel.ID] = channel
	if m.permissions[channel.ID] == nil {
		m.permissions[channel.ID] = make(map[string]int64)
	}
}

func (m *MockDiscordSession) SetUserChannelPermissions(userID, channelID string, permissions int64) {
	if m.permissions[channelID] == nil {
		m.permissions[channelID] = make(map[string]int64)
	}
	m.permissions[channelID][userID] = permissions
}

func (m *MockDiscordSession) SetError(shouldError bool, message string) {
	m.shouldError = shouldError
	m.errorMessage = message
}

// Test setup helper
func setupPermissionTest(t *testing.T) (*PermissionServiceImpl, *MockDiscordSession, *StorageService) {
	// Create temporary directory for test storage
	tempDir := t.TempDir()

	// Create storage service
	storage, err := NewStorageService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}

	// Create mock Discord session
	mockSession := NewMockDiscordSession()

	// Create logger
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	// Create permission service
	permService := NewPermissionService(mockSession, storage, logger)

	return permService, mockSession, storage
}

// Test CanInviteBot functionality
func TestCanInviteBot(t *testing.T) {
	permService, mockSession, _ := setupPermissionTest(t)

	// Test data
	guildID := "test-guild-123"
	userID := "test-user-456"
	adminUserID := "admin-user-789"
	ownerUserID := "owner-user-101"
	roleID := "test-role-111"

	// Setup mock guild with roles
	guild := &discordgo.Guild{
		ID:      guildID,
		OwnerID: ownerUserID,
		Roles: []*discordgo.Role{
			{
				ID:          roleID,
				Name:        "TTS User",
				Permissions: 0,
			},
			{
				ID:          "admin-role-222",
				Name:        "Admin",
				Permissions: discordgo.PermissionAdministrator,
			},
		},
	}
	mockSession.AddGuild(guild)

	// Setup mock members
	regularMember := &discordgo.Member{
		User:  &discordgo.User{ID: userID},
		Roles: []string{roleID},
	}
	mockSession.AddMember(guildID, regularMember)

	adminMember := &discordgo.Member{
		User:  &discordgo.User{ID: adminUserID},
		Roles: []string{"admin-role-222"},
	}
	mockSession.AddMember(guildID, adminMember)

	ownerMember := &discordgo.Member{
		User:  &discordgo.User{ID: ownerUserID},
		Roles: []string{},
	}
	mockSession.AddMember(guildID, ownerMember)

	t.Run("Empty parameters should return error", func(t *testing.T) {
		_, err := permService.CanInviteBot("", guildID)
		if err == nil {
			t.Error("Expected error for empty userID")
		}

		_, err = permService.CanInviteBot(userID, "")
		if err == nil {
			t.Error("Expected error for empty guildID")
		}
	})

	t.Run("No required roles configured - any member can invite", func(t *testing.T) {
		// Default config has no required roles
		canInvite, err := permService.CanInviteBot(userID, guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !canInvite {
			t.Error("Expected user to be able to invite bot when no roles required")
		}
	})

	t.Run("Required roles configured - user with role can invite", func(t *testing.T) {
		// Set required roles
		err := permService.SetRequiredRoles(guildID, []string{roleID})
		if err != nil {
			t.Fatalf("Failed to set required roles: %v", err)
		}

		canInvite, err := permService.CanInviteBot(userID, guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !canInvite {
			t.Error("Expected user with required role to be able to invite bot")
		}
	})

	t.Run("Administrator can always invite", func(t *testing.T) {
		canInvite, err := permService.CanInviteBot(adminUserID, guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !canInvite {
			t.Error("Expected administrator to be able to invite bot")
		}
	})

	t.Run("Guild owner can always invite", func(t *testing.T) {
		canInvite, err := permService.CanInviteBot(ownerUserID, guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !canInvite {
			t.Error("Expected guild owner to be able to invite bot")
		}
	})

	t.Run("User without required role cannot invite", func(t *testing.T) {
		// Add a user without the required role
		userWithoutRole := &discordgo.Member{
			User:  &discordgo.User{ID: "user-no-role-333"},
			Roles: []string{},
		}
		mockSession.AddMember(guildID, userWithoutRole)

		canInvite, err := permService.CanInviteBot("user-no-role-333", guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if canInvite {
			t.Error("Expected user without required role to not be able to invite bot")
		}
	})

	t.Run("Non-member cannot invite", func(t *testing.T) {
		canInvite, err := permService.CanInviteBot("non-member-444", guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if canInvite {
			t.Error("Expected non-member to not be able to invite bot")
		}
	})
}

// Test HasChannelAccess functionality
func TestHasChannelAccess(t *testing.T) {
	permService, mockSession, _ := setupPermissionTest(t)

	userID := "test-user-456"
	voiceChannelID := "voice-channel-123"
	textChannelID := "text-channel-456"

	// Setup mock channels
	voiceChannel := &discordgo.Channel{
		ID:   voiceChannelID,
		Type: discordgo.ChannelTypeGuildVoice,
		Name: "Test Voice Channel",
	}
	mockSession.AddChannel(voiceChannel)

	textChannel := &discordgo.Channel{
		ID:   textChannelID,
		Type: discordgo.ChannelTypeGuildText,
		Name: "Test Text Channel",
	}
	mockSession.AddChannel(textChannel)

	t.Run("Empty parameters should return error", func(t *testing.T) {
		_, err := permService.HasChannelAccess("", voiceChannelID)
		if err == nil {
			t.Error("Expected error for empty userID")
		}

		_, err = permService.HasChannelAccess(userID, "")
		if err == nil {
			t.Error("Expected error for empty channelID")
		}
	})

	t.Run("Voice channel access with proper permissions", func(t *testing.T) {
		// Set permissions for voice channel access
		permissions := int64(discordgo.PermissionVoiceConnect | discordgo.PermissionViewChannel)
		mockSession.SetUserChannelPermissions(userID, voiceChannelID, permissions)

		hasAccess, err := permService.HasChannelAccess(userID, voiceChannelID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !hasAccess {
			t.Error("Expected user to have voice channel access")
		}
	})

	t.Run("Voice channel access without connect permission", func(t *testing.T) {
		// Set permissions without voice connect
		permissions := int64(discordgo.PermissionViewChannel)
		mockSession.SetUserChannelPermissions(userID, voiceChannelID, permissions)

		hasAccess, err := permService.HasChannelAccess(userID, voiceChannelID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if hasAccess {
			t.Error("Expected user to not have voice channel access without connect permission")
		}
	})

	t.Run("Text channel access with proper permissions", func(t *testing.T) {
		// Set permissions for text channel access
		permissions := int64(discordgo.PermissionReadMessages | discordgo.PermissionViewChannel)
		mockSession.SetUserChannelPermissions(userID, textChannelID, permissions)

		hasAccess, err := permService.HasChannelAccess(userID, textChannelID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !hasAccess {
			t.Error("Expected user to have text channel access")
		}
	})

	t.Run("Text channel access without read permission", func(t *testing.T) {
		// Set permissions to 0 (no permissions)
		permissions := int64(0)
		mockSession.SetUserChannelPermissions(userID, textChannelID, permissions)

		hasAccess, err := permService.HasChannelAccess(userID, textChannelID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if hasAccess {
			t.Error("Expected user to not have text channel access without any permissions")
		}
	})

	t.Run("Non-existent channel should return error", func(t *testing.T) {
		_, err := permService.HasChannelAccess(userID, "non-existent-channel")
		if err == nil {
			t.Error("Expected error for non-existent channel")
		}
	})
}

// Test SetRequiredRoles and GetRequiredRoles functionality
func TestRequiredRolesManagement(t *testing.T) {
	permService, mockSession, _ := setupPermissionTest(t)

	guildID := "test-guild-123"
	roleID1 := "role-1"
	roleID2 := "role-2"
	invalidRoleID := "invalid-role"

	// Setup mock guild with roles
	guild := &discordgo.Guild{
		ID: guildID,
		Roles: []*discordgo.Role{
			{ID: roleID1, Name: "Role 1"},
			{ID: roleID2, Name: "Role 2"},
		},
	}
	mockSession.AddGuild(guild)

	t.Run("Set and get required roles", func(t *testing.T) {
		requiredRoles := []string{roleID1, roleID2}

		err := permService.SetRequiredRoles(guildID, requiredRoles)
		if err != nil {
			t.Fatalf("Failed to set required roles: %v", err)
		}

		retrievedRoles, err := permService.GetRequiredRoles(guildID)
		if err != nil {
			t.Fatalf("Failed to get required roles: %v", err)
		}

		if len(retrievedRoles) != len(requiredRoles) {
			t.Errorf("Expected %d roles, got %d", len(requiredRoles), len(retrievedRoles))
		}

		for i, role := range requiredRoles {
			if retrievedRoles[i] != role {
				t.Errorf("Expected role %s at index %d, got %s", role, i, retrievedRoles[i])
			}
		}
	})

	t.Run("Set required roles with invalid role ID should fail", func(t *testing.T) {
		invalidRoles := []string{roleID1, invalidRoleID}

		err := permService.SetRequiredRoles(guildID, invalidRoles)
		if err == nil {
			t.Error("Expected error when setting invalid role ID")
		}
	})

	t.Run("Empty guild ID should return error", func(t *testing.T) {
		err := permService.SetRequiredRoles("", []string{roleID1})
		if err == nil {
			t.Error("Expected error for empty guild ID")
		}

		_, err = permService.GetRequiredRoles("")
		if err == nil {
			t.Error("Expected error for empty guild ID")
		}
	})

	t.Run("Clear required roles", func(t *testing.T) {
		err := permService.SetRequiredRoles(guildID, []string{})
		if err != nil {
			t.Fatalf("Failed to clear required roles: %v", err)
		}

		retrievedRoles, err := permService.GetRequiredRoles(guildID)
		if err != nil {
			t.Fatalf("Failed to get required roles: %v", err)
		}

		if len(retrievedRoles) != 0 {
			t.Errorf("Expected 0 roles after clearing, got %d", len(retrievedRoles))
		}
	})
}

// Test CanControlBot functionality
func TestCanControlBot(t *testing.T) {
	permService, mockSession, _ := setupPermissionTest(t)

	guildID := "test-guild-123"
	userID := "test-user-456"
	roleID := "test-role-111"

	// Setup mock guild and member
	guild := &discordgo.Guild{
		ID: guildID,
		Roles: []*discordgo.Role{
			{ID: roleID, Name: "TTS User", Permissions: 0},
		},
	}
	mockSession.AddGuild(guild)

	member := &discordgo.Member{
		User:  &discordgo.User{ID: userID},
		Roles: []string{roleID},
	}
	mockSession.AddMember(guildID, member)

	t.Run("User with permissions can control bot", func(t *testing.T) {
		canControl, err := permService.CanControlBot(userID, guildID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !canControl {
			t.Error("Expected user to be able to control bot")
		}
	})

	t.Run("Empty parameters should return error", func(t *testing.T) {
		_, err := permService.CanControlBot("", guildID)
		if err == nil {
			t.Error("Expected error for empty userID")
		}

		_, err = permService.CanControlBot(userID, "")
		if err == nil {
			t.Error("Expected error for empty guildID")
		}
	})
}

// Test error handling
func TestPermissionServiceErrorHandling(t *testing.T) {
	permService, mockSession, _ := setupPermissionTest(t)

	guildID := "test-guild-123"
	userID := "test-user-456"
	channelID := "test-channel-789"

	t.Run("Discord API error should be handled", func(t *testing.T) {
		mockSession.SetError(true, "API Error")

		_, err := permService.CanInviteBot(userID, guildID)
		if err == nil {
			t.Error("Expected error when Discord API fails")
		}

		_, err = permService.HasChannelAccess(userID, channelID)
		if err == nil {
			t.Error("Expected error when Discord API fails")
		}

		// Reset error state
		mockSession.SetError(false, "")
	})
}
