package tts

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

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
	joinHandler  *JoinCommandHandler
	leaveHandler *LeaveCommandHandler
	logger       *log.Logger
}

// NewTTSCommandIntegration creates a new TTS command integration instance
func NewTTSCommandIntegration(
	session *discordgo.Session,
	storage *StorageService,
	logger *log.Logger,
) (*TTSCommandIntegration, error) {
	// Create TTS services
	sessionWrapper := NewDiscordSessionWrapper(session)
	permissionService := NewPermissionService(sessionWrapper, storage, logger)
	channelService := NewChannelService(storage, sessionWrapper, permissionService)
	userService := NewUserService(storage)
	voiceManager := NewVoiceManager(session)

	// Create message queue and config service (needed for error recovery)
	messageQueue := NewMessageQueue()
	configService := &mockConfigServiceForIntegration{}

	// Create TTS manager (needed for error recovery)
	ttsManager, err := NewGoogleTTSManager(messageQueue)
	if err != nil {
		return nil, err
	}

	// Create error recovery manager
	errorRecovery := NewErrorRecoveryManager(voiceManager, ttsManager, messageQueue, configService)

	// Create command handlers
	joinHandler := NewJoinCommandHandler(
		voiceManager,
		channelService,
		permissionService,
		userService,
		errorRecovery,
		logger,
	)

	leaveHandler := NewLeaveCommandHandler(
		voiceManager,
		channelService,
		permissionService,
		errorRecovery,
		logger,
	)

	return &TTSCommandIntegration{
		joinHandler:  joinHandler,
		leaveHandler: leaveHandler,
		logger:       logger,
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
	}
}

// RegisterWithBot registers TTS command handlers with the bot's command router
// This would be called from the main bot initialization code
func (t *TTSCommandIntegration) RegisterWithBot(commandRouter interface {
	RegisterHandler(handler interface {
		Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
		Definition() *discordgo.ApplicationCommand
	}) error
}) error {
	// Register join command handler
	if err := commandRouter.RegisterHandler(t.joinHandler); err != nil {
		return err
	}

	// Register leave command handler
	if err := commandRouter.RegisterHandler(t.leaveHandler); err != nil {
		return err
	}

	t.logger.Println("Successfully registered TTS command handlers")
	return nil
}
