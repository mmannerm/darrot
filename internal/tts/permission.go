package tts

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// DiscordSession interface defines the Discord API methods needed by PermissionService
type DiscordSession interface {
	Guild(guildID string) (*discordgo.Guild, error)
	GuildMember(guildID, userID string) (*discordgo.Member, error)
	Channel(channelID string) (*discordgo.Channel, error)
	UserChannelPermissions(userID, channelID string) (int64, error)
}

// PermissionServiceImpl implements the PermissionService interface
type PermissionServiceImpl struct {
	session DiscordSession
	storage *StorageService
	logger  *log.Logger
}

// NewPermissionService creates a new permission service instance
func NewPermissionService(session DiscordSession, storage *StorageService, logger *log.Logger) *PermissionServiceImpl {
	return &PermissionServiceImpl{
		session: session,
		storage: storage,
		logger:  logger,
	}
}

// CanInviteBot checks if a user has permission to invite the bot to voice channels
// Requirements: 7.1, 7.2, 7.3, 7.4, 7.5
func (p *PermissionServiceImpl) CanInviteBot(userID, guildID string) (bool, error) {
	if userID == "" || guildID == "" {
		return false, fmt.Errorf("userID and guildID cannot be empty")
	}

	// First check if user is a guild member
	isMember, err := p.isGuildMember(userID, guildID)
	if err != nil {
		return false, fmt.Errorf("failed to check guild membership: %w", err)
	}
	if !isMember {
		return false, nil
	}

	// Get guild configuration to check required roles
	guildConfig, err := p.storage.LoadGuildConfig(guildID)
	if err != nil {
		return false, fmt.Errorf("failed to load guild config: %w", err)
	}

	// If no roles are configured, any server member can invite the bot (Requirement 7.3)
	if len(guildConfig.RequiredRoles) == 0 {
		return true, nil
	}

	// Check if user has administrator permissions (Requirement 7.5)
	hasAdminPerms, err := p.hasAdministratorPermissions(userID, guildID)
	if err != nil {
		return false, fmt.Errorf("failed to check administrator permissions: %w", err)
	}
	if hasAdminPerms {
		return true, nil
	}

	// Check if user has any of the required roles (Requirement 7.1)
	hasRequiredRole, err := p.hasAnyRole(userID, guildID, guildConfig.RequiredRoles)
	if err != nil {
		return false, fmt.Errorf("failed to check user roles: %w", err)
	}

	return hasRequiredRole, nil
}

// CanControlBot checks if a user has permission to control bot functions (pause, resume, skip)
// Requirements: 7.1, 7.2, 7.4, 7.5
func (p *PermissionServiceImpl) CanControlBot(userID, guildID string) (bool, error) {
	if userID == "" || guildID == "" {
		return false, fmt.Errorf("userID and guildID cannot be empty")
	}

	// For now, use the same permission logic as bot invitation
	// This can be extended later if different control permissions are needed
	return p.CanInviteBot(userID, guildID)
}

// HasChannelAccess validates if a user has access to a specific channel
// Requirements: 8.1, 8.2, 8.3, 8.4, 8.5
func (p *PermissionServiceImpl) HasChannelAccess(userID, channelID string) (bool, error) {
	if userID == "" || channelID == "" {
		return false, fmt.Errorf("userID and channelID cannot be empty")
	}

	// Get channel information
	channel, err := p.session.Channel(channelID)
	if err != nil {
		return false, fmt.Errorf("failed to get channel information: %w", err)
	}

	// Check if user has access to the channel based on channel type
	switch channel.Type {
	case discordgo.ChannelTypeGuildVoice:
		return p.hasVoiceChannelAccess(userID, channel)
	case discordgo.ChannelTypeGuildText:
		return p.hasTextChannelAccess(userID, channel)
	default:
		return false, fmt.Errorf("unsupported channel type: %v", channel.Type)
	}
}

// SetRequiredRoles sets the required roles for bot invitation in a guild
// Requirements: 7.1, 7.4
func (p *PermissionServiceImpl) SetRequiredRoles(guildID string, roleIDs []string) error {
	if guildID == "" {
		return fmt.Errorf("guildID cannot be empty")
	}

	// Validate that all role IDs exist in the guild
	for _, roleID := range roleIDs {
		if err := p.validateRoleExists(guildID, roleID); err != nil {
			return fmt.Errorf("invalid role ID %s: %w", roleID, err)
		}
	}

	// Load current guild configuration
	guildConfig, err := p.storage.LoadGuildConfig(guildID)
	if err != nil {
		return fmt.Errorf("failed to load guild config: %w", err)
	}

	// Update required roles
	guildConfig.RequiredRoles = roleIDs

	// Save updated configuration
	if err := p.storage.SaveGuildConfig(*guildConfig); err != nil {
		return fmt.Errorf("failed to save guild config: %w", err)
	}

	p.logger.Printf("Updated required roles for guild %s: %v", guildID, roleIDs)
	return nil
}

// GetRequiredRoles retrieves the required roles for bot invitation in a guild
// Requirements: 7.1, 7.4
func (p *PermissionServiceImpl) GetRequiredRoles(guildID string) ([]string, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guildID cannot be empty")
	}

	guildConfig, err := p.storage.LoadGuildConfig(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to load guild config: %w", err)
	}

	return guildConfig.RequiredRoles, nil
}

// Helper methods

// isGuildMember checks if a user is a member of the specified guild
func (p *PermissionServiceImpl) isGuildMember(userID, guildID string) (bool, error) {
	_, err := p.session.GuildMember(guildID, userID)
	if err != nil {
		if restErr, ok := err.(*discordgo.RESTError); ok && restErr.Message != nil {
			if restErr.Message.Code == discordgo.ErrCodeUnknownMember {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check guild membership: %w", err)
	}
	return true, nil
}

// hasAdministratorPermissions checks if a user has administrator permissions in a guild
func (p *PermissionServiceImpl) hasAdministratorPermissions(userID, guildID string) (bool, error) {
	member, err := p.session.GuildMember(guildID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get guild member: %w", err)
	}

	guild, err := p.session.Guild(guildID)
	if err != nil {
		return false, fmt.Errorf("failed to get guild: %w", err)
	}

	// Check if user is the guild owner
	if member.User.ID == guild.OwnerID {
		return true, nil
	}

	// Check roles for administrator permission
	for _, roleID := range member.Roles {
		role, err := p.getRoleByID(guildID, roleID)
		if err != nil {
			continue // Skip roles that can't be retrieved
		}

		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true, nil
		}
	}

	return false, nil
}

// hasAnyRole checks if a user has any of the specified roles
func (p *PermissionServiceImpl) hasAnyRole(userID, guildID string, requiredRoles []string) (bool, error) {
	member, err := p.session.GuildMember(guildID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get guild member: %w", err)
	}

	// Check if user has any of the required roles
	for _, userRoleID := range member.Roles {
		for _, requiredRoleID := range requiredRoles {
			if userRoleID == requiredRoleID {
				return true, nil
			}
		}
	}

	return false, nil
}

// hasVoiceChannelAccess checks if a user has access to a voice channel
func (p *PermissionServiceImpl) hasVoiceChannelAccess(userID string, channel *discordgo.Channel) (bool, error) {
	// Get user permissions for the channel
	permissions, err := p.session.UserChannelPermissions(userID, channel.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get user channel permissions: %w", err)
	}

	// Check if user can connect to voice channel
	canConnect := permissions&discordgo.PermissionVoiceConnect != 0
	canView := permissions&discordgo.PermissionViewChannel != 0

	return canConnect && canView, nil
}

// hasTextChannelAccess checks if a user has read access to a text channel
func (p *PermissionServiceImpl) hasTextChannelAccess(userID string, channel *discordgo.Channel) (bool, error) {
	// Get user permissions for the channel
	permissions, err := p.session.UserChannelPermissions(userID, channel.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get user channel permissions: %w", err)
	}

	// In Discord, ViewChannel permission includes the ability to read messages in text channels
	// So we only need to check for ViewChannel permission
	canView := permissions&discordgo.PermissionViewChannel != 0

	return canView, nil
}

// validateRoleExists checks if a role exists in the specified guild
func (p *PermissionServiceImpl) validateRoleExists(guildID, roleID string) error {
	_, err := p.getRoleByID(guildID, roleID)
	return err
}

// getRoleByID retrieves a role by ID from a guild
func (p *PermissionServiceImpl) getRoleByID(guildID, roleID string) (*discordgo.Role, error) {
	guild, err := p.session.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild: %w", err)
	}

	for _, role := range guild.Roles {
		if role.ID == roleID {
			return role, nil
		}
	}

	return nil, fmt.Errorf("role not found")
}
