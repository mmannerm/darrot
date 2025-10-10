package tts

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// CommandRouter interface that matches the bot's CommandRouter
type CommandRouter interface {
	RegisterHandler(handler CommandHandler) error
}

// CommandHandler interface that matches the bot's CommandHandler
type CommandHandler interface {
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
	Definition() *discordgo.ApplicationCommand
}

// mockConfigServiceForIntegration provides a simple config service for integration
type mockConfigServiceForIntegration struct{}

func (m *mockConfigServiceForIntegration) GetGuildConfig(guildID string) (*GuildTTSConfig, error) {
	return nil, nil
}

func (m *mockConfigServiceForIntegration) SetGuildConfig(guildID string, config *GuildTTSConfig) error {
	return nil
}

func (m *mockConfigServiceForIntegration) SetRequiredRoles(guildID string, roleIDs []string) error {
	return nil
}

func (m *mockConfigServiceForIntegration) GetRequiredRoles(guildID string) ([]string, error) {
	return nil, nil
}

func (m *mockConfigServiceForIntegration) SetTTSSettings(guildID string, settings TTSConfig) error {
	return nil
}

func (m *mockConfigServiceForIntegration) GetTTSSettings(guildID string) (*TTSConfig, error) {
	return nil, nil
}

func (m *mockConfigServiceForIntegration) SetMaxQueueSize(guildID string, size int) error {
	return nil
}

func (m *mockConfigServiceForIntegration) GetMaxQueueSize(guildID string) (int, error) {
	return 10, nil
}

func (m *mockConfigServiceForIntegration) ValidateConfig(config *GuildTTSConfig) error {
	return nil
}

// TTSCommandIntegration provides methods to integrate TTS command handlers with the bot
type TTSCommandIntegration struct {
	joinHandler    *JoinCommandHandler
	leaveHandler   *LeaveCommandHandler
	controlHandler *ControlCommandHandler
	optInHandler   *OptInCommandHandler
	configHandler  *ConfigCommandHandler
	logger         *log.Logger
}

// NewTTSCommandIntegration creates a new TTS command integration instance
func NewTTSCommandIntegration(
	session *discordgo.Session,
	storage *StorageService,
	voiceManager VoiceManager,
	ttsProcessor TTSProcessor,
	logger *log.Logger,
) (*TTSCommandIntegration, error) {
	// Create TTS services
	sessionWrapper := NewDiscordSessionWrapper(session)
	permissionService := NewPermissionService(sessionWrapper, storage, logger)
	channelService := NewChannelService(storage, sessionWrapper, permissionService)
	userService := NewUserService(storage)

	logger.Printf("Using shared voice manager instance: %p", voiceManager)

	// Create message queue and config service (needed for error recovery)
	messageQueue := NewMessageQueue()
	configService := &mockConfigServiceForIntegration{}

	// Create TTS manager (needed for error recovery) - using Google Cloud TTS
	ttsManager, err := NewGoogleTTSManager(messageQueue, "")
	if err != nil {
		return nil, err
	}

	// Create error recovery manager
	errorRecovery := NewErrorRecoveryManager(voiceManager, ttsManager, messageQueue, configService)

	// Create all command handlers
	joinHandler := NewJoinCommandHandler(
		voiceManager,
		channelService,
		permissionService,
		userService,
		ttsProcessor,
		errorRecovery,
		logger,
	)

	leaveHandler := NewLeaveCommandHandler(
		voiceManager,
		channelService,
		permissionService,
		ttsProcessor,
		errorRecovery,
		logger,
	)

	controlHandler := NewControlCommandHandler(
		voiceManager,
		messageQueue,
		permissionService,
		logger,
	)

	optInHandler := NewOptInCommandHandler(
		userService,
		logger,
	)

	configHandler := NewConfigCommandHandler(
		configService,
		permissionService,
		ttsManager,
		messageQueue,
		logger,
	)

	return &TTSCommandIntegration{
		joinHandler:    joinHandler,
		leaveHandler:   leaveHandler,
		controlHandler: controlHandler,
		optInHandler:   optInHandler,
		configHandler:  configHandler,
		logger:         logger,
	}, nil
}

// GetJoinHandler returns the join command handler
func (t *TTSCommandIntegration) GetJoinHandler() *JoinCommandHandler {
	return t.joinHandler
}

// GetLeaveHandler returns the leave command handler
func (t *TTSCommandIntegration) GetLeaveHandler() *LeaveCommandHandler {
	return t.leaveHandler
}

// GetControlHandler returns the control command handler
func (t *TTSCommandIntegration) GetControlHandler() *ControlCommandHandler {
	return t.controlHandler
}

// GetOptInHandler returns the opt-in command handler
func (t *TTSCommandIntegration) GetOptInHandler() *OptInCommandHandler {
	return t.optInHandler
}

// GetConfigHandler returns the config command handler
func (t *TTSCommandIntegration) GetConfigHandler() *ConfigCommandHandler {
	return t.configHandler
}

// GetCommandHandlers returns all TTS command handlers for registration
func (t *TTSCommandIntegration) GetCommandHandlers() []interface {
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
	Definition() *discordgo.ApplicationCommand
} {
	return []interface {
		Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
		Definition() *discordgo.ApplicationCommand
	}{
		t.joinHandler,
		t.leaveHandler,
		t.controlHandler,
		t.optInHandler,
		t.configHandler,
	}
}

// RegisterWithBot registers TTS command handlers with the bot's command router
// This would be called from the main bot initialization code
func (t *TTSCommandIntegration) RegisterWithBot(commandRouter CommandRouter) error {
	// Register all TTS command handlers
	handlers := []struct {
		name    string
		handler CommandHandler
	}{
		{"join", t.joinHandler},
		{"leave", t.leaveHandler},
		{"control", t.controlHandler},
		{"opt-in", t.optInHandler},
		{"config", t.configHandler},
	}

	for _, h := range handlers {
		if err := commandRouter.RegisterHandler(h.handler); err != nil {
			return fmt.Errorf("failed to register %s command handler: %w", h.name, err)
		}
		t.logger.Printf("Registered TTS %s command handler", h.name)
	}

	t.logger.Println("Successfully registered all TTS command handlers")
	return nil
}
