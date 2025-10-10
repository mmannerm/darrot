package tts

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ChannelServiceImpl implements the ChannelService interface
type ChannelServiceImpl struct {
	storage           *StorageService
	session           DiscordSession
	permissionService PermissionService
}

// NewChannelService creates a new channel service instance
func NewChannelService(storage *StorageService, session DiscordSession, permissionService PermissionService) *ChannelServiceImpl {
	return &ChannelServiceImpl{
		storage:           storage,
		session:           session,
		permissionService: permissionService,
	}
}

// CreatePairing creates a new voice-text channel pairing
func (c *ChannelServiceImpl) CreatePairing(guildID, voiceChannelID, textChannelID string) error {
	return c.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, "")
}

// CreatePairingWithCreator creates a new voice-text channel pairing with a specified creator
func (c *ChannelServiceImpl) CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, createdBy string) error {
	// Validate input parameters
	if guildID == "" {
		return fmt.Errorf("guild ID is required")
	}
	if voiceChannelID == "" {
		return fmt.Errorf("voice channel ID is required")
	}
	if textChannelID == "" {
		return fmt.Errorf("text channel ID is required")
	}

	// Check if voice channel already has a pairing
	existingPairing, err := c.storage.LoadChannelPairing(guildID, voiceChannelID)
	if err == nil && existingPairing.IsActive {
		return fmt.Errorf("voice channel %s is already paired with text channel %s", voiceChannelID, existingPairing.TextChannelID)
	}

	// Check if text channel is already paired with another voice channel
	if c.IsChannelPaired(guildID, textChannelID) {
		return fmt.Errorf("text channel %s is already paired with another voice channel", textChannelID)
	}

	// Validate that the channels exist and are of the correct type
	voiceChannel, err := c.session.Channel(voiceChannelID)
	if err != nil {
		return fmt.Errorf("failed to get voice channel: %w", err)
	}
	if voiceChannel.Type != discordgo.ChannelTypeGuildVoice {
		return fmt.Errorf("channel %s is not a voice channel", voiceChannelID)
	}

	textChannel, err := c.session.Channel(textChannelID)
	if err != nil {
		return fmt.Errorf("failed to get text channel: %w", err)
	}
	if textChannel.Type != discordgo.ChannelTypeGuildText {
		return fmt.Errorf("channel %s is not a text channel", textChannelID)
	}

	// Verify channels are in the same guild
	if voiceChannel.GuildID != guildID || textChannel.GuildID != guildID {
		return fmt.Errorf("channels must be in the specified guild")
	}

	// Create the pairing
	pairing := ChannelPairingStorage{
		GuildID:        guildID,
		VoiceChannelID: voiceChannelID,
		TextChannelID:  textChannelID,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		IsActive:       true,
	}

	return c.storage.SaveChannelPairing(pairing)
}

// RemovePairing removes a voice-text channel pairing
func (c *ChannelServiceImpl) RemovePairing(guildID, voiceChannelID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID is required")
	}
	if voiceChannelID == "" {
		return fmt.Errorf("voice channel ID is required")
	}

	// Check if pairing exists
	existingPairing, err := c.storage.LoadChannelPairing(guildID, voiceChannelID)
	if err != nil {
		return fmt.Errorf("channel pairing not found: %w", err)
	}

	// Mark as inactive and save
	existingPairing.IsActive = false
	if err := c.storage.SaveChannelPairing(*existingPairing); err != nil {
		return fmt.Errorf("failed to deactivate pairing: %w", err)
	}

	// Remove the pairing file
	return c.storage.RemoveChannelPairing(guildID, voiceChannelID)
}

// GetPairing retrieves a voice-text channel pairing
func (c *ChannelServiceImpl) GetPairing(guildID, voiceChannelID string) (*ChannelPairing, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild ID is required")
	}
	if voiceChannelID == "" {
		return nil, fmt.Errorf("voice channel ID is required")
	}

	storagePairing, err := c.storage.LoadChannelPairing(guildID, voiceChannelID)
	if err != nil {
		return nil, fmt.Errorf("channel pairing not found: %w", err)
	}

	if !storagePairing.IsActive {
		return nil, fmt.Errorf("channel pairing is not active")
	}

	// Convert storage format to interface format
	pairing := &ChannelPairing{
		GuildID:        storagePairing.GuildID,
		VoiceChannelID: storagePairing.VoiceChannelID,
		TextChannelID:  storagePairing.TextChannelID,
		CreatedBy:      storagePairing.CreatedBy,
		CreatedAt:      storagePairing.CreatedAt,
	}

	return pairing, nil
}

// ValidateChannelAccess validates that a user has access to a specific channel
func (c *ChannelServiceImpl) ValidateChannelAccess(userID, channelID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if channelID == "" {
		return fmt.Errorf("channel ID is required")
	}

	// Use the permission service to check channel access
	hasAccess, err := c.permissionService.HasChannelAccess(userID, channelID)
	if err != nil {
		return fmt.Errorf("failed to check channel access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("user %s does not have access to channel %s", userID, channelID)
	}

	return nil
}

// IsChannelPaired checks if a text channel is already paired with any voice channel
func (c *ChannelServiceImpl) IsChannelPaired(guildID, textChannelID string) bool {
	if guildID == "" || textChannelID == "" {
		return false
	}

	// Get all pairings for the guild
	pairings, err := c.storage.ListGuildPairings(guildID)
	if err != nil {
		return false
	}

	// Check if any active pairing uses this text channel
	for _, pairing := range pairings {
		if pairing.IsActive && pairing.TextChannelID == textChannelID {
			return true
		}
	}

	return false
}

// SetPairingCreator sets the creator of a channel pairing (used when creating pairings)
func (c *ChannelServiceImpl) SetPairingCreator(guildID, voiceChannelID, creatorID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID is required")
	}
	if voiceChannelID == "" {
		return fmt.Errorf("voice channel ID is required")
	}
	if creatorID == "" {
		return fmt.Errorf("creator ID is required")
	}

	// Load existing pairing
	pairing, err := c.storage.LoadChannelPairing(guildID, voiceChannelID)
	if err != nil {
		return fmt.Errorf("channel pairing not found: %w", err)
	}

	// Update creator
	pairing.CreatedBy = creatorID

	// Save updated pairing
	return c.storage.SaveChannelPairing(*pairing)
}

// ListGuildPairings returns all active channel pairings for a guild
func (c *ChannelServiceImpl) ListGuildPairings(guildID string) ([]*ChannelPairing, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild ID is required")
	}

	storagePairings, err := c.storage.ListGuildPairings(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild pairings: %w", err)
	}

	var pairings []*ChannelPairing
	for _, sp := range storagePairings {
		if sp.IsActive {
			pairing := &ChannelPairing{
				GuildID:        sp.GuildID,
				VoiceChannelID: sp.VoiceChannelID,
				TextChannelID:  sp.TextChannelID,
				CreatedBy:      sp.CreatedBy,
				CreatedAt:      sp.CreatedAt,
			}
			pairings = append(pairings, pairing)
		}
	}

	return pairings, nil
}
