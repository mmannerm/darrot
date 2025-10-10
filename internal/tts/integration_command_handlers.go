package tts

import (
	"fmt"
)

// Mock command handlers for integration testing

// mockJoinHandlerIntegration provides a mock join command handler for integration tests
type mockJoinHandlerIntegration struct {
	env *integrationTestEnvironment
}

func newMockJoinHandlerIntegration(env *integrationTestEnvironment) *mockJoinHandlerIntegration {
	return &mockJoinHandlerIntegration{env: env}
}

func (h *mockJoinHandlerIntegration) handleJoinCommand(guildID, voiceChannelID, textChannelID, userID string) error {
	// Check permissions
	canInvite, err := h.env.permissionService.CanInviteBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canInvite {
		return fmt.Errorf("user %s cannot invite bot in guild %s", userID, guildID)
	}

	// Validate channel access
	err = h.env.channelService.ValidateChannelAccess(userID, voiceChannelID)
	if err != nil {
		return fmt.Errorf("user %s cannot access voice channel %s: %v", userID, voiceChannelID, err)
	}

	err = h.env.channelService.ValidateChannelAccess(userID, textChannelID)
	if err != nil {
		return fmt.Errorf("user %s cannot access text channel %s: %v", userID, textChannelID, err)
	}

	// Join voice channel
	_, err = h.env.voiceManager.JoinChannel(guildID, voiceChannelID)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %v", err)
	}

	// Create channel pairing
	err = h.env.channelService.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, userID)
	if err != nil {
		return fmt.Errorf("failed to create channel pairing: %v", err)
	}

	// Auto opt-in the user who invited the bot
	err = h.env.userService.AutoOptIn(userID, guildID)
	if err != nil {
		return fmt.Errorf("failed to auto opt-in user: %v", err)
	}

	return nil
}

// mockLeaveHandlerIntegration provides a mock leave command handler for integration tests
type mockLeaveHandlerIntegration struct {
	env *integrationTestEnvironment
}

func newMockLeaveHandlerIntegration(env *integrationTestEnvironment) *mockLeaveHandlerIntegration {
	return &mockLeaveHandlerIntegration{env: env}
}

func (h *mockLeaveHandlerIntegration) handleLeaveCommand(guildID, userID string) error {
	// Check permissions
	canControl, err := h.env.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canControl {
		return fmt.Errorf("user %s cannot control bot in guild %s", userID, guildID)
	}

	// Leave voice channel
	err = h.env.voiceManager.LeaveChannel(guildID)
	if err != nil {
		return fmt.Errorf("failed to leave voice channel: %v", err)
	}

	// Clear message queue
	err = h.env.messageQueue.Clear(guildID)
	if err != nil {
		return fmt.Errorf("failed to clear message queue: %v", err)
	}

	return nil
}

// mockControlHandlerIntegration provides a mock control command handler for integration tests
type mockControlHandlerIntegration struct {
	env *integrationTestEnvironment
}

func newMockControlHandlerIntegration(env *integrationTestEnvironment) *mockControlHandlerIntegration {
	return &mockControlHandlerIntegration{env: env}
}

func (h *mockControlHandlerIntegration) handlePauseCommand(guildID, userID string) error {
	// Check permissions
	canControl, err := h.env.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canControl {
		return fmt.Errorf("user %s cannot control bot in guild %s", userID, guildID)
	}

	// Pause playback
	err = h.env.voiceManager.PausePlayback(guildID)
	if err != nil {
		return fmt.Errorf("failed to pause playback: %v", err)
	}

	return nil
}

func (h *mockControlHandlerIntegration) handleResumeCommand(guildID, userID string) error {
	// Check permissions
	canControl, err := h.env.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canControl {
		return fmt.Errorf("user %s cannot control bot in guild %s", userID, guildID)
	}

	// Resume playback
	err = h.env.voiceManager.ResumePlayback(guildID)
	if err != nil {
		return fmt.Errorf("failed to resume playback: %v", err)
	}

	return nil
}

func (h *mockControlHandlerIntegration) handleSkipCommand(guildID, userID string) error {
	// Check permissions
	canControl, err := h.env.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canControl {
		return fmt.Errorf("user %s cannot control bot in guild %s", userID, guildID)
	}

	// Skip current message
	err = h.env.voiceManager.SkipCurrentMessage(guildID)
	if err != nil {
		return fmt.Errorf("failed to skip current message: %v", err)
	}

	return nil
}

// mockConfigHandlerIntegration provides a mock config command handler for integration tests
type mockConfigHandlerIntegration struct {
	env *integrationTestEnvironment
}

func newMockConfigHandlerIntegration(env *integrationTestEnvironment) *mockConfigHandlerIntegration {
	return &mockConfigHandlerIntegration{env: env}
}

func (h *mockConfigHandlerIntegration) handleSetVoiceCommand(guildID, userID string, config TTSConfig) error {
	// Check permissions (assume admin permissions for config changes)
	canInvite, err := h.env.permissionService.CanInviteBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canInvite {
		return fmt.Errorf("user %s cannot configure bot in guild %s", userID, guildID)
	}

	// Set TTS configuration
	err = h.env.ttsManager.SetVoiceConfig(guildID, config)
	if err != nil {
		return fmt.Errorf("failed to set voice config: %v", err)
	}

	return nil
}

func (h *mockConfigHandlerIntegration) handleSetRequiredRolesCommand(guildID, userID string, roleIDs []string) error {
	// Check permissions (assume admin permissions for config changes)
	canInvite, err := h.env.permissionService.CanInviteBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canInvite {
		return fmt.Errorf("user %s cannot configure bot in guild %s", userID, guildID)
	}

	// Set required roles
	err = h.env.permissionService.SetRequiredRoles(guildID, roleIDs)
	if err != nil {
		return fmt.Errorf("failed to set required roles: %v", err)
	}

	return nil
}

func (h *mockConfigHandlerIntegration) handleSetQueueLimitCommand(guildID, userID string, maxSize int) error {
	// Check permissions (assume admin permissions for config changes)
	canInvite, err := h.env.permissionService.CanInviteBot(userID, guildID)
	if err != nil {
		return err
	}
	if !canInvite {
		return fmt.Errorf("user %s cannot configure bot in guild %s", userID, guildID)
	}

	// Set queue limit
	err = h.env.messageQueue.SetMaxSize(guildID, maxSize)
	if err != nil {
		return fmt.Errorf("failed to set queue limit: %v", err)
	}

	return nil
}
