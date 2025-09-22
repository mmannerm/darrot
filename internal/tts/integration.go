package tts

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

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

	// Create command handlers
	joinHandler := NewJoinCommandHandler(
		voiceManager,
		channelService,
		permissionService,
		userService,
		logger,
	)

	leaveHandler := NewLeaveCommandHandler(
		voiceManager,
		channelService,
		permissionService,
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
