package tts

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// TTSCommandHandler extends the basic CommandHandler interface with TTS-specific validation
type TTSCommandHandler interface {
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
	Definition() *discordgo.ApplicationCommand
	ValidatePermissions(userID, guildID string) error
	ValidateChannelAccess(userID, channelID string) error
}

// JoinCommandHandler handles voice channel join commands for TTS
type JoinCommandHandler struct {
	voiceManager      VoiceManager
	channelService    ChannelService
	permissionService PermissionService
	userService       UserService
	ttsProcessor      TTSProcessor
	errorRecovery     *ErrorRecoveryManager
	logger            *log.Logger
}

// NewJoinCommandHandler creates a new join command handler
func NewJoinCommandHandler(
	voiceManager VoiceManager,
	channelService ChannelService,
	permissionService PermissionService,
	userService UserService,
	ttsProcessor TTSProcessor,
	errorRecovery *ErrorRecoveryManager,
	logger *log.Logger,
) *JoinCommandHandler {
	return &JoinCommandHandler{
		voiceManager:      voiceManager,
		channelService:    channelService,
		permissionService: permissionService,
		userService:       userService,
		ttsProcessor:      ttsProcessor,
		errorRecovery:     errorRecovery,
		logger:            logger,
	}
}

// Definition returns the Discord slash command definition for the join command
func (h *JoinCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "darrot-join",
		Description: "Join a voice channel and start TTS for messages from a text channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "voice-channel",
				Description: "The voice channel to join",
				Required:    true,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildVoice,
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "text-channel",
				Description: "The text channel to monitor (defaults to voice channel's text chat)",
				Required:    false,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildText,
				},
			},
		},
	}
}

// Handle processes the join command interaction
func (h *JoinCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Validate guild context
	if i.GuildID == "" {
		return h.respondError(s, i, "This command can only be used in a server.")
	}

	userID := i.Member.User.ID
	guildID := i.GuildID

	// Validate permissions
	if err := h.ValidatePermissions(userID, guildID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Permission denied: %v", err))
	}

	// Extract command options
	options := i.ApplicationCommandData().Options
	voiceChannelID := options[0].ChannelValue(s).ID

	var textChannelID string
	if len(options) > 1 && options[1].ChannelValue(s) != nil {
		textChannelID = options[1].ChannelValue(s).ID
	} else {
		// Default to the channel where the command was invoked
		textChannelID = i.ChannelID
	}

	// Validate channel access
	if err := h.ValidateChannelAccess(userID, voiceChannelID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Cannot access voice channel: %v", err))
	}

	if err := h.ValidateChannelAccess(userID, textChannelID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Cannot access text channel: %v", err))
	}

	// Check if bot is already connected to a different channel in this guild
	if existingConn, exists := h.voiceManager.GetConnection(guildID); exists {
		if existingConn.ChannelID != voiceChannelID {
			// Leave current channel first
			if err := h.voiceManager.LeaveChannel(guildID); err != nil {
				h.logger.Printf("Warning: Failed to leave current channel: %v", err)
			}
			// Stop TTS processing for the old connection
			if err := h.ttsProcessor.StopGuildProcessing(guildID); err != nil {
				h.logger.Printf("Warning: Failed to stop TTS processing for guild %s: %v", guildID, err)
			}
		} else {
			// Already connected to the same voice channel
			// Check if we need to update the text channel pairing
			existingPairing, err := h.channelService.GetPairing(guildID, voiceChannelID)
			if err == nil && existingPairing.TextChannelID == textChannelID {
				// Same pairing already exists and bot is connected
				// Just ensure TTS processing is started
				if err := h.ttsProcessor.StartGuildProcessing(guildID); err != nil {
					h.logger.Printf("Warning: Failed to start TTS processing for guild %s: %v", guildID, err)
				}

				voiceChannel, _ := s.Channel(voiceChannelID)
				textChannel, _ := s.Channel(textChannelID)

				voiceChannelName := voiceChannel.Name
				if voiceChannelName == "" {
					voiceChannelName = voiceChannelID
				}

				textChannelName := textChannel.Name
				if textChannelName == "" {
					textChannelName = textChannelID
				}

				responseMessage := fmt.Sprintf("✅ Already connected to voice channel **%s** and monitoring text channel **%s** for TTS messages.", voiceChannelName, textChannelName)
				return h.respondSuccess(s, i, responseMessage)
			}
		}
	}

	// Check for stale pairings (pairing exists but bot isn't connected)
	if _, err := h.channelService.GetPairing(guildID, voiceChannelID); err == nil {
		// Pairing exists but bot isn't connected - clean it up
		if !h.voiceManager.IsConnected(guildID) {
			h.logger.Printf("Found stale pairing for guild %s, voice channel %s - cleaning up", guildID, voiceChannelID)
			if err := h.channelService.RemovePairing(guildID, voiceChannelID); err != nil {
				h.logger.Printf("Warning: Failed to remove stale pairing: %v", err)
			}
			// Stop any stale TTS processing
			if err := h.ttsProcessor.StopGuildProcessing(guildID); err != nil {
				h.logger.Printf("Warning: Failed to stop stale TTS processing for guild %s: %v", guildID, err)
			}
		}
	}

	// Join the voice channel with error recovery
	_, err := h.voiceManager.JoinChannel(guildID, voiceChannelID)
	if err != nil {
		h.logger.Printf("Initial voice channel join failed for guild %s: %v", guildID, err)

		// Create user-friendly error message
		if h.errorRecovery != nil {
			userMessage := h.errorRecovery.CreateUserFriendlyErrorMessage(err, guildID)
			return h.respondError(s, i, userMessage)
		}

		return h.respondError(s, i, fmt.Sprintf("Failed to join voice channel: %v", err))
	}

	// Create channel pairing (this will now work since we cleaned up any stale pairings)
	if err := h.channelService.CreatePairingWithCreator(guildID, voiceChannelID, textChannelID, userID); err != nil {
		// If pairing creation fails, leave the voice channel
		_ = h.voiceManager.LeaveChannel(guildID)
		return h.respondError(s, i, fmt.Sprintf("Failed to create channel pairing: %v", err))
	}

	// Auto opt-in the user who invited the bot
	if err := h.userService.AutoOptIn(userID, guildID); err != nil {
		h.logger.Printf("Warning: Failed to auto opt-in user %s: %v", userID, err)
	}

	// Start TTS processing for this guild
	if err := h.ttsProcessor.StartGuildProcessing(guildID); err != nil {
		h.logger.Printf("Warning: Failed to start TTS processing for guild %s: %v", guildID, err)
	} else {
		h.logger.Printf("Started TTS processing for guild %s", guildID)
	}

	// Get channel names for response
	voiceChannel, _ := s.Channel(voiceChannelID)
	textChannel, _ := s.Channel(textChannelID)

	voiceChannelName := voiceChannel.Name
	if voiceChannelName == "" {
		voiceChannelName = voiceChannelID
	}

	textChannelName := textChannel.Name
	if textChannelName == "" {
		textChannelName = textChannelID
	}

	responseMessage := fmt.Sprintf("✅ Joined voice channel **%s** and monitoring text channel **%s** for TTS messages.\n\nUsers must opt-in to have their messages read aloud. You have been automatically opted-in.",
		voiceChannelName, textChannelName)

	return h.respondSuccess(s, i, responseMessage)
}

// ValidatePermissions validates that the user has permission to invite the bot
func (h *JoinCommandHandler) ValidatePermissions(userID, guildID string) error {
	canInvite, err := h.permissionService.CanInviteBot(userID, guildID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}

	if !canInvite {
		return fmt.Errorf("you don't have permission to invite the bot to voice channels")
	}

	return nil
}

// ValidateChannelAccess validates that the user has access to the specified channel
func (h *JoinCommandHandler) ValidateChannelAccess(userID, channelID string) error {
	return h.channelService.ValidateChannelAccess(userID, channelID)
}

// LeaveCommandHandler handles voice channel leave commands for TTS
type LeaveCommandHandler struct {
	voiceManager      VoiceManager
	channelService    ChannelService
	permissionService PermissionService
	ttsProcessor      TTSProcessor
	errorRecovery     *ErrorRecoveryManager
	logger            *log.Logger
}

// NewLeaveCommandHandler creates a new leave command handler
func NewLeaveCommandHandler(
	voiceManager VoiceManager,
	channelService ChannelService,
	permissionService PermissionService,
	ttsProcessor TTSProcessor,
	errorRecovery *ErrorRecoveryManager,
	logger *log.Logger,
) *LeaveCommandHandler {
	return &LeaveCommandHandler{
		voiceManager:      voiceManager,
		channelService:    channelService,
		permissionService: permissionService,
		ttsProcessor:      ttsProcessor,
		errorRecovery:     errorRecovery,
		logger:            logger,
	}
}

// Definition returns the Discord slash command definition for the leave command
func (h *LeaveCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "darrot-leave",
		Description: "Stop TTS and leave the voice channel",
	}
}

// Handle processes the leave command interaction
func (h *LeaveCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Validate guild context
	if i.GuildID == "" {
		return h.respondError(s, i, "This command can only be used in a server.")
	}

	userID := i.Member.User.ID
	guildID := i.GuildID

	// Validate permissions
	if err := h.ValidatePermissions(userID, guildID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Permission denied: %v", err))
	}

	// Check if bot is connected to a voice channel
	connection, exists := h.voiceManager.GetConnection(guildID)
	if !exists {
		return h.respondError(s, i, "I'm not currently in a voice channel in this server.")
	}

	voiceChannelID := connection.ChannelID

	// Leave the voice channel with error recovery
	if err := h.voiceManager.LeaveChannel(guildID); err != nil {
		h.logger.Printf("Failed to leave voice channel for guild %s: %v", guildID, err)

		// Create user-friendly error message
		if h.errorRecovery != nil {
			userMessage := h.errorRecovery.CreateUserFriendlyErrorMessage(err, guildID)
			return h.respondError(s, i, userMessage)
		}

		return h.respondError(s, i, fmt.Sprintf("Failed to leave voice channel: %v", err))
	}

	// Stop TTS processing for this guild
	if err := h.ttsProcessor.StopGuildProcessing(guildID); err != nil {
		h.logger.Printf("Warning: Failed to stop TTS processing for guild %s: %v", guildID, err)
	} else {
		h.logger.Printf("Stopped TTS processing for guild %s", guildID)
	}

	// Remove channel pairing
	if err := h.channelService.RemovePairing(guildID, voiceChannelID); err != nil {
		h.logger.Printf("Warning: Failed to remove channel pairing: %v", err)
	}

	// Get channel name for response
	voiceChannel, _ := s.Channel(voiceChannelID)
	channelName := voiceChannel.Name
	if channelName == "" {
		channelName = voiceChannelID
	}

	responseMessage := fmt.Sprintf("✅ Left voice channel **%s** and stopped TTS monitoring.", channelName)
	return h.respondSuccess(s, i, responseMessage)
}

// ValidatePermissions validates that the user has permission to control the bot
func (h *LeaveCommandHandler) ValidatePermissions(userID, guildID string) error {
	canControl, err := h.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}

	if !canControl {
		return fmt.Errorf("you don't have permission to control the bot")
	}

	return nil
}

// ValidateChannelAccess is not needed for leave command but required by interface
func (h *LeaveCommandHandler) ValidateChannelAccess(userID, channelID string) error {
	return nil // Not applicable for leave command
}

// Helper methods for response handling

func (h *JoinCommandHandler) respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

func (h *JoinCommandHandler) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *LeaveCommandHandler) respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

func (h *LeaveCommandHandler) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ControlCommandHandler handles TTS control commands (pause, resume, skip)
type ControlCommandHandler struct {
	voiceManager      VoiceManager
	messageQueue      MessageQueue
	permissionService PermissionService
	logger            *log.Logger
}

// NewControlCommandHandler creates a new control command handler
func NewControlCommandHandler(
	voiceManager VoiceManager,
	messageQueue MessageQueue,
	permissionService PermissionService,
	logger *log.Logger,
) *ControlCommandHandler {
	return &ControlCommandHandler{
		voiceManager:      voiceManager,
		messageQueue:      messageQueue,
		permissionService: permissionService,
		logger:            logger,
	}
}

// Definition returns the Discord slash command definition for TTS control commands
func (h *ControlCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "darrot-control",
		Description: "Control TTS playback (pause, resume, skip)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "action",
				Description: "The control action to perform",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "pause",
						Value: "pause",
					},
					{
						Name:  "resume",
						Value: "resume",
					},
					{
						Name:  "skip",
						Value: "skip",
					},
				},
			},
		},
	}
}

// Handle processes the control command interaction
func (h *ControlCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Validate guild context
	if i.GuildID == "" {
		return h.respondError(s, i, "This command can only be used in a server.")
	}

	userID := i.Member.User.ID
	guildID := i.GuildID

	// Validate permissions
	if err := h.ValidatePermissions(userID, guildID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Permission denied: %v", err))
	}

	// Check if bot is connected to a voice channel
	connection, exists := h.voiceManager.GetConnection(guildID)
	if !exists {
		return h.respondError(s, i, "I'm not currently in a voice channel in this server.")
	}

	// Extract command options
	options := i.ApplicationCommandData().Options
	action := options[0].StringValue()

	// Execute the requested action
	switch action {
	case "pause":
		return h.handlePause(s, i, guildID, connection)
	case "resume":
		return h.handleResume(s, i, guildID, connection)
	case "skip":
		return h.handleSkip(s, i, guildID, connection)
	default:
		return h.respondError(s, i, "Invalid action. Use pause, resume, or skip.")
	}
}

// handlePause pauses TTS playback
func (h *ControlCommandHandler) handlePause(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, connection *VoiceConnection) error {
	if connection.IsPaused {
		return h.respondError(s, i, "TTS is already paused.")
	}

	if err := h.voiceManager.PausePlayback(guildID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Failed to pause TTS: %v", err))
	}

	return h.respondSuccess(s, i, "⏸️ TTS playback paused. Use `/tts-control resume` to continue.")
}

// handleResume resumes TTS playback
func (h *ControlCommandHandler) handleResume(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, connection *VoiceConnection) error {
	if !connection.IsPaused {
		return h.respondError(s, i, "TTS is not currently paused.")
	}

	if err := h.voiceManager.ResumePlayback(guildID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Failed to resume TTS: %v", err))
	}

	queueSize := h.messageQueue.Size(guildID)
	var message string
	if queueSize > 0 {
		message = fmt.Sprintf("▶️ TTS playback resumed. %d message(s) in queue.", queueSize)
	} else {
		message = "▶️ TTS playback resumed. No messages currently in queue."
	}

	return h.respondSuccess(s, i, message)
}

// handleSkip skips the current message and proceeds to the next
func (h *ControlCommandHandler) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, connection *VoiceConnection) error {
	// Skip current playing message if any
	if err := h.voiceManager.SkipCurrentMessage(guildID); err != nil {
		h.logger.Printf("Warning: Failed to skip current message: %v", err)
	}

	// Skip next message in queue
	skippedMessage, err := h.messageQueue.SkipNext(guildID)
	if err != nil {
		return h.respondError(s, i, fmt.Sprintf("Failed to skip message: %v", err))
	}

	if skippedMessage == nil {
		return h.respondError(s, i, "No messages in queue to skip.")
	}

	queueSize := h.messageQueue.Size(guildID)
	var message string
	if queueSize > 0 {
		message = fmt.Sprintf("⏭️ Skipped message from **%s**. %d message(s) remaining in queue.", skippedMessage.Username, queueSize)
	} else {
		message = fmt.Sprintf("⏭️ Skipped message from **%s**. Queue is now empty.", skippedMessage.Username)
	}

	return h.respondSuccess(s, i, message)
}

// ValidatePermissions validates that the user has permission to control the bot
func (h *ControlCommandHandler) ValidatePermissions(userID, guildID string) error {
	canControl, err := h.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}

	if !canControl {
		return fmt.Errorf("you don't have permission to control the bot")
	}

	return nil
}

// ValidateChannelAccess is not needed for control commands but required by interface
func (h *ControlCommandHandler) ValidateChannelAccess(userID, channelID string) error {
	return nil // Not applicable for control commands
}

// Helper methods for response handling

func (h *ControlCommandHandler) respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

func (h *ControlCommandHandler) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// OptInCommandHandler handles user opt-in and opt-out commands for TTS
type OptInCommandHandler struct {
	userService UserService
	logger      *log.Logger
}

// NewOptInCommandHandler creates a new opt-in command handler
func NewOptInCommandHandler(
	userService UserService,
	logger *log.Logger,
) *OptInCommandHandler {
	return &OptInCommandHandler{
		userService: userService,
		logger:      logger,
	}
}

// Definition returns the Discord slash command definition for the opt-in command
func (h *OptInCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "darrot-optin",
		Description: "Manage your TTS opt-in preferences",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "action",
				Description: "The opt-in action to perform",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "opt-in",
						Value: "opt-in",
					},
					{
						Name:  "opt-out",
						Value: "opt-out",
					},
					{
						Name:  "status",
						Value: "status",
					},
				},
			},
		},
	}
}

// Handle processes the opt-in command interaction
func (h *OptInCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Validate guild context
	if i.GuildID == "" {
		return h.respondError(s, i, "This command can only be used in a server.")
	}

	userID := i.Member.User.ID
	guildID := i.GuildID

	// Extract command options
	options := i.ApplicationCommandData().Options
	action := options[0].StringValue()

	// Execute the requested action
	switch action {
	case "opt-in":
		return h.handleOptIn(s, i, userID, guildID)
	case "opt-out":
		return h.handleOptOut(s, i, userID, guildID)
	case "status":
		return h.handleStatus(s, i, userID, guildID)
	default:
		return h.respondError(s, i, "Invalid action. Use opt-in, opt-out, or status.")
	}
}

// handleOptIn opts the user in for TTS message reading
func (h *OptInCommandHandler) handleOptIn(s *discordgo.Session, i *discordgo.InteractionCreate, userID, guildID string) error {
	// Check current opt-in status
	isOptedIn, err := h.userService.IsOptedIn(userID, guildID)
	if err != nil {
		h.logger.Printf("Error checking opt-in status for user %s in guild %s: %v", userID, guildID, err)
		return h.respondError(s, i, "Failed to check your current opt-in status.")
	}

	if isOptedIn {
		return h.respondError(s, i, "You are already opted-in for TTS message reading in this server.")
	}

	// Opt the user in
	if err := h.userService.SetOptInStatus(userID, guildID, true); err != nil {
		h.logger.Printf("Error opting in user %s in guild %s: %v", userID, guildID, err)
		return h.respondError(s, i, "Failed to opt you in for TTS. Please try again.")
	}

	return h.respondSuccess(s, i, "✅ You have been opted-in for TTS message reading in this server. Your messages will now be read aloud when the bot is active in voice channels.")
}

// handleOptOut opts the user out of TTS message reading
func (h *OptInCommandHandler) handleOptOut(s *discordgo.Session, i *discordgo.InteractionCreate, userID, guildID string) error {
	// Check current opt-in status
	isOptedIn, err := h.userService.IsOptedIn(userID, guildID)
	if err != nil {
		h.logger.Printf("Error checking opt-in status for user %s in guild %s: %v", userID, guildID, err)
		return h.respondError(s, i, "Failed to check your current opt-in status.")
	}

	if !isOptedIn {
		return h.respondError(s, i, "You are already opted-out of TTS message reading in this server.")
	}

	// Opt the user out
	if err := h.userService.SetOptInStatus(userID, guildID, false); err != nil {
		h.logger.Printf("Error opting out user %s in guild %s: %v", userID, guildID, err)
		return h.respondError(s, i, "Failed to opt you out of TTS. Please try again.")
	}

	return h.respondSuccess(s, i, "✅ You have been opted-out of TTS message reading in this server. Your messages will no longer be read aloud.")
}

// handleStatus shows the user's current opt-in status
func (h *OptInCommandHandler) handleStatus(s *discordgo.Session, i *discordgo.InteractionCreate, userID, guildID string) error {
	// Check current opt-in status
	isOptedIn, err := h.userService.IsOptedIn(userID, guildID)
	if err != nil {
		h.logger.Printf("Error checking opt-in status for user %s in guild %s: %v", userID, guildID, err)
		return h.respondError(s, i, "Failed to check your current opt-in status.")
	}

	var statusMessage string
	if isOptedIn {
		statusMessage = "✅ **Opted-in**: Your messages will be read aloud when the bot is active in voice channels.\n\nUse `/tts-optin opt-out` to opt out of TTS message reading."
	} else {
		statusMessage = "❌ **Opted-out**: Your messages will not be read aloud.\n\nUse `/tts-optin opt-in` to opt in for TTS message reading."
	}

	return h.respondSuccess(s, i, statusMessage)
}

// ValidatePermissions validates user permissions (users can only manage their own preferences)
func (h *OptInCommandHandler) ValidatePermissions(userID, guildID string) error {
	// Users can always manage their own opt-in preferences
	// No additional permission validation needed
	return nil
}

// ValidateChannelAccess is not needed for opt-in commands but required by interface
func (h *OptInCommandHandler) ValidateChannelAccess(userID, channelID string) error {
	return nil // Not applicable for opt-in commands
}

// Helper methods for response handling

func (h *OptInCommandHandler) respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral, // Make responses private to the user
		},
	})
}

func (h *OptInCommandHandler) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ConfigCommandHandler handles administrator TTS configuration commands
type ConfigCommandHandler struct {
	configService     ConfigService
	permissionService PermissionService
	ttsManager        TTSManager
	messageQueue      MessageQueue
	logger            *log.Logger
}

// NewConfigCommandHandler creates a new configuration command handler
func NewConfigCommandHandler(
	configService ConfigService,
	permissionService PermissionService,
	ttsManager TTSManager,
	messageQueue MessageQueue,
	logger *log.Logger,
) *ConfigCommandHandler {
	return &ConfigCommandHandler{
		configService:     configService,
		permissionService: permissionService,
		ttsManager:        ttsManager,
		messageQueue:      messageQueue,
		logger:            logger,
	}
}

// Definition returns the Discord slash command definition for the config command
func (h *ConfigCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "darrot-config",
		Description: "Configure TTS settings for this server (Administrator only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "roles",
				Description: "Configure required roles for bot invitations",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "action",
						Description: "Action to perform",
						Required:    true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "set", Value: "set"},
							{Name: "add", Value: "add"},
							{Name: "remove", Value: "remove"},
							{Name: "clear", Value: "clear"},
							{Name: "list", Value: "list"},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionRole,
						Name:        "role",
						Description: "Role to add or remove",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "voice",
				Description: "Configure TTS voice settings",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "setting",
						Description: "Voice setting to configure",
						Required:    true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "voice", Value: "voice"},
							{Name: "speed", Value: "speed"},
							{Name: "volume", Value: "volume"},
							{Name: "list-voices", Value: "list-voices"},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "value",
						Description: "Value to set (voice name, speed 0.25-4.0, volume 0.0-1.0)",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "queue",
				Description: "Configure message queue settings",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "setting",
						Description: "Queue setting to configure",
						Required:    true,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "max-size", Value: "max-size"},
							{Name: "show", Value: "show"},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "value",
						Description: "Maximum queue size (1-50)",
						Required:    false,
						MinValue:    &[]float64{1}[0],
						MaxValue:    50,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "show",
				Description: "Show current TTS configuration",
			},
		},
	}
}

// Handle processes the config command interaction
func (h *ConfigCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Validate guild context
	if i.GuildID == "" {
		return h.respondError(s, i, "This command can only be used in a server.")
	}

	userID := i.Member.User.ID
	guildID := i.GuildID

	// Validate administrator permissions
	if err := h.ValidatePermissions(userID, guildID); err != nil {
		return h.respondError(s, i, fmt.Sprintf("Permission denied: %v", err))
	}

	// Extract subcommand
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return h.respondError(s, i, "No subcommand specified.")
	}

	subcommand := options[0]
	switch subcommand.Name {
	case "roles":
		return h.handleRolesConfig(s, i, guildID, subcommand.Options)
	case "voice":
		return h.handleVoiceConfig(s, i, guildID, subcommand.Options)
	case "queue":
		return h.handleQueueConfig(s, i, guildID, subcommand.Options)
	case "show":
		return h.handleShowConfig(s, i, guildID)
	default:
		return h.respondError(s, i, "Invalid subcommand.")
	}
}

// handleRolesConfig handles role configuration commands
func (h *ConfigCommandHandler) handleRolesConfig(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	if len(options) == 0 {
		return h.respondError(s, i, "No action specified for roles configuration.")
	}

	action := options[0].StringValue()

	switch action {
	case "list":
		return h.handleListRoles(s, i, guildID)
	case "clear":
		return h.handleClearRoles(s, i, guildID)
	case "set", "add", "remove":
		if len(options) < 2 {
			return h.respondError(s, i, fmt.Sprintf("Role parameter required for '%s' action.", action))
		}
		roleID := options[1].RoleValue(s, guildID).ID
		return h.handleRoleAction(s, i, guildID, action, roleID)
	default:
		return h.respondError(s, i, "Invalid action for roles configuration.")
	}
}

// handleRoleAction handles individual role actions (set, add, remove)
func (h *ConfigCommandHandler) handleRoleAction(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, action, roleID string) error {
	currentRoles, err := h.configService.GetRequiredRoles(guildID)
	if err != nil {
		h.logger.Printf("Error getting required roles for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to get current role configuration.")
	}

	var newRoles []string
	var actionMessage string

	switch action {
	case "set":
		newRoles = []string{roleID}
		actionMessage = "Required role set to"
	case "add":
		// Check if role already exists
		for _, existingRole := range currentRoles {
			if existingRole == roleID {
				return h.respondError(s, i, "Role is already in the required roles list.")
			}
		}
		newRoles = append(currentRoles, roleID)
		actionMessage = "Role added to required roles:"
	case "remove":
		// Remove role from list
		for _, existingRole := range currentRoles {
			if existingRole != roleID {
				newRoles = append(newRoles, existingRole)
			}
		}
		if len(newRoles) == len(currentRoles) {
			return h.respondError(s, i, "Role was not found in the required roles list.")
		}
		actionMessage = "Role removed from required roles:"
	}

	// Update the configuration
	if err := h.configService.SetRequiredRoles(guildID, newRoles); err != nil {
		h.logger.Printf("Error setting required roles for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to update role configuration.")
	}

	// Get role name for response
	role, err := s.State.Role(guildID, roleID)
	if err != nil {
		role = &discordgo.Role{ID: roleID, Name: roleID} // Fallback to ID
	}

	var responseMessage string
	if len(newRoles) == 0 {
		responseMessage = fmt.Sprintf("✅ %s **%s**\n\nNo roles are now required - any server member can invite the bot to voice channels.", actionMessage, role.Name)
	} else {
		responseMessage = fmt.Sprintf("✅ %s **%s**\n\nTotal required roles: %d", actionMessage, role.Name, len(newRoles))
	}

	return h.respondSuccess(s, i, responseMessage)
}

// handleListRoles lists current required roles
func (h *ConfigCommandHandler) handleListRoles(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	roles, err := h.configService.GetRequiredRoles(guildID)
	if err != nil {
		h.logger.Printf("Error getting required roles for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to get role configuration.")
	}

	if len(roles) == 0 {
		return h.respondSuccess(s, i, "📋 **Required Roles Configuration**\n\nNo roles are currently required - any server member can invite the bot to voice channels.")
	}

	var roleNames []string
	for _, roleID := range roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			roleNames = append(roleNames, fmt.Sprintf("Unknown Role (%s)", roleID))
		} else {
			roleNames = append(roleNames, role.Name)
		}
	}

	responseMessage := fmt.Sprintf("📋 **Required Roles Configuration**\n\nUsers must have one of these roles to invite the bot:\n• %s",
		fmt.Sprintf("**%s**", roleNames[0]))

	for _, roleName := range roleNames[1:] {
		responseMessage += fmt.Sprintf("\n• **%s**", roleName)
	}

	return h.respondSuccess(s, i, responseMessage)
}

// handleClearRoles clears all required roles
func (h *ConfigCommandHandler) handleClearRoles(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	if err := h.configService.SetRequiredRoles(guildID, []string{}); err != nil {
		h.logger.Printf("Error clearing required roles for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to clear role configuration.")
	}

	return h.respondSuccess(s, i, "✅ **All required roles cleared**\n\nAny server member can now invite the bot to voice channels.")
}

// handleVoiceConfig handles voice configuration commands
func (h *ConfigCommandHandler) handleVoiceConfig(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	if len(options) == 0 {
		return h.respondError(s, i, "No setting specified for voice configuration.")
	}

	setting := options[0].StringValue()

	switch setting {
	case "list-voices":
		return h.handleListVoices(s, i)
	case "voice", "speed", "volume":
		if len(options) < 2 {
			return h.handleShowVoiceSetting(s, i, guildID, setting)
		}
		value := options[1].StringValue()
		return h.handleSetVoiceSetting(s, i, guildID, setting, value)
	default:
		return h.respondError(s, i, "Invalid setting for voice configuration.")
	}
}

// handleListVoices lists available TTS voices
func (h *ConfigCommandHandler) handleListVoices(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	voices := h.ttsManager.GetSupportedVoices()
	if len(voices) == 0 {
		return h.respondError(s, i, "No voices are currently available.")
	}

	responseMessage := "🎤 **Available TTS Voices**\n\n"
	for _, voice := range voices {
		responseMessage += fmt.Sprintf("• **%s** (%s) - %s %s\n", voice.Name, voice.ID, voice.Language, voice.Gender)
	}

	return h.respondSuccess(s, i, responseMessage)
}

// handleShowVoiceSetting shows current voice setting value
func (h *ConfigCommandHandler) handleShowVoiceSetting(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, setting string) error {
	config, err := h.configService.GetTTSSettings(guildID)
	if err != nil {
		h.logger.Printf("Error getting TTS settings for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to get current voice settings.")
	}

	var currentValue string
	switch setting {
	case "voice":
		currentValue = config.Voice
		if currentValue == "" {
			currentValue = "default"
		}
	case "speed":
		currentValue = fmt.Sprintf("%.2f", config.Speed)
	case "volume":
		currentValue = fmt.Sprintf("%.2f", config.Volume)
	}

	responseMessage := fmt.Sprintf("🎤 **Current %s setting:** %s", setting, currentValue)
	return h.respondSuccess(s, i, responseMessage)
}

// handleSetVoiceSetting sets a voice configuration setting
func (h *ConfigCommandHandler) handleSetVoiceSetting(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, setting, value string) error {
	// Get current settings
	currentConfig, err := h.configService.GetTTSSettings(guildID)
	if err != nil {
		h.logger.Printf("Error getting TTS settings for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to get current voice settings.")
	}

	// Create new config with updated setting
	newConfig := *currentConfig

	switch setting {
	case "voice":
		// Validate voice exists
		voices := h.ttsManager.GetSupportedVoices()
		validVoice := false
		for _, voice := range voices {
			if voice.ID == value || voice.Name == value {
				newConfig.Voice = voice.ID
				validVoice = true
				break
			}
		}
		if !validVoice {
			return h.respondError(s, i, fmt.Sprintf("Invalid voice '%s'. Use `/tts-config voice list-voices` to see available voices.", value))
		}

	case "speed":
		speed, err := parseFloat32(value)
		if err != nil || speed < 0.25 || speed > 4.0 {
			return h.respondError(s, i, "Speed must be a number between 0.25 and 4.0")
		}
		newConfig.Speed = speed

	case "volume":
		volume, err := parseFloat32(value)
		if err != nil || volume < 0.0 || volume > 1.0 {
			return h.respondError(s, i, "Volume must be a number between 0.0 and 1.0")
		}
		newConfig.Volume = volume
	}

	// Update the configuration
	if err := h.configService.SetTTSSettings(guildID, newConfig); err != nil {
		h.logger.Printf("Error setting TTS settings for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to update voice settings.")
	}

	// Update TTS manager with new config
	if err := h.ttsManager.SetVoiceConfig(guildID, newConfig); err != nil {
		h.logger.Printf("Warning: Failed to update TTS manager config for guild %s: %v", guildID, err)
	}

	responseMessage := fmt.Sprintf("✅ **%s updated to:** %s", setting, value)
	return h.respondSuccess(s, i, responseMessage)
}

// handleQueueConfig handles queue configuration commands
func (h *ConfigCommandHandler) handleQueueConfig(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	if len(options) == 0 {
		return h.respondError(s, i, "No setting specified for queue configuration.")
	}

	setting := options[0].StringValue()

	switch setting {
	case "show":
		return h.handleShowQueueConfig(s, i, guildID)
	case "max-size":
		if len(options) < 2 {
			return h.handleShowQueueConfig(s, i, guildID)
		}
		size := int(options[1].IntValue())
		return h.handleSetMaxQueueSize(s, i, guildID, size)
	default:
		return h.respondError(s, i, "Invalid setting for queue configuration.")
	}
}

// handleShowQueueConfig shows current queue configuration
func (h *ConfigCommandHandler) handleShowQueueConfig(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	maxSize, err := h.configService.GetMaxQueueSize(guildID)
	if err != nil {
		h.logger.Printf("Error getting max queue size for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to get queue configuration.")
	}

	currentSize := h.messageQueue.Size(guildID)
	responseMessage := fmt.Sprintf("📋 **Message Queue Configuration**\n\nMax queue size: **%d**\nCurrent queue size: **%d**", maxSize, currentSize)

	return h.respondSuccess(s, i, responseMessage)
}

// handleSetMaxQueueSize sets the maximum queue size
func (h *ConfigCommandHandler) handleSetMaxQueueSize(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, size int) error {
	// Validate size range
	if size < 1 || size > 50 {
		return h.respondError(s, i, "Queue size must be between 1 and 50.")
	}

	// Update configuration
	if err := h.configService.SetMaxQueueSize(guildID, size); err != nil {
		h.logger.Printf("Error setting max queue size for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to update queue configuration.")
	}

	// Update message queue
	if err := h.messageQueue.SetMaxSize(guildID, size); err != nil {
		h.logger.Printf("Warning: Failed to update message queue max size for guild %s: %v", guildID, err)
	}

	responseMessage := fmt.Sprintf("✅ **Maximum queue size updated to:** %d", size)
	return h.respondSuccess(s, i, responseMessage)
}

// handleShowConfig shows complete TTS configuration
func (h *ConfigCommandHandler) handleShowConfig(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	config, err := h.configService.GetGuildConfig(guildID)
	if err != nil {
		h.logger.Printf("Error getting guild config for guild %s: %v", guildID, err)
		return h.respondError(s, i, "Failed to get server configuration.")
	}

	responseMessage := "⚙️ **TTS Configuration for this Server**\n\n"

	// Required roles
	if len(config.RequiredRoles) == 0 {
		responseMessage += "**Required Roles:** None (any member can invite bot)\n"
	} else {
		responseMessage += "**Required Roles:**\n"
		for _, roleID := range config.RequiredRoles {
			role, err := s.State.Role(guildID, roleID)
			if err != nil {
				responseMessage += fmt.Sprintf("• Unknown Role (%s)\n", roleID)
			} else {
				responseMessage += fmt.Sprintf("• %s\n", role.Name)
			}
		}
	}

	// TTS settings
	responseMessage += "\n**Voice Settings:**\n"
	responseMessage += fmt.Sprintf("• Voice: %s\n", config.TTSSettings.Voice)
	responseMessage += fmt.Sprintf("• Speed: %.2f\n", config.TTSSettings.Speed)
	responseMessage += fmt.Sprintf("• Volume: %.2f\n", config.TTSSettings.Volume)

	// Queue settings
	currentQueueSize := h.messageQueue.Size(guildID)
	responseMessage += "\n**Queue Settings:**\n"
	responseMessage += fmt.Sprintf("• Max Size: %d\n", config.MaxQueueSize)
	responseMessage += fmt.Sprintf("• Current Size: %d\n", currentQueueSize)

	return h.respondSuccess(s, i, responseMessage)
}

// ValidatePermissions validates that the user has administrator permissions
func (h *ConfigCommandHandler) ValidatePermissions(userID, guildID string) error {
	// Check if user has administrator permission in the guild
	// This would typically check Discord permissions, but for now we'll use a simple check
	// In a real implementation, this would verify the user has ADMINISTRATOR permission

	// For now, we'll assume the permission service can check admin permissions
	canControl, err := h.permissionService.CanControlBot(userID, guildID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}

	if !canControl {
		return fmt.Errorf("you must have administrator permissions to configure TTS settings")
	}

	return nil
}

// ValidateChannelAccess is not needed for config commands but required by interface
func (h *ConfigCommandHandler) ValidateChannelAccess(userID, channelID string) error {
	return nil // Not applicable for config commands
}

// Helper methods for response handling

func (h *ConfigCommandHandler) respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral, // Make config responses private to admin
		},
	})
}

func (h *ConfigCommandHandler) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// parseFloat32 is a helper function to parse string to float32
func parseFloat32(s string) (float32, error) {
	f, err := fmt.Sscanf(s, "%f", new(float32))
	if err != nil || f != 1 {
		return 0, fmt.Errorf("invalid float value")
	}
	var result float32
	_, _ = fmt.Sscanf(s, "%f", &result)
	return result, nil
}
