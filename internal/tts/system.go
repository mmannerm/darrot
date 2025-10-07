package tts

import (
	"fmt"
	"log"

	"darrot/internal/config"

	"github.com/bwmarrin/discordgo"
)

// TTSSystem is the main coordinator for all TTS functionality
type TTSSystem struct {
	// Core components
	ttsManager        TTSManager
	voiceManager      VoiceManager
	messageQueue      MessageQueue
	ttsProcessor      TTSProcessor
	messageMonitor    *MessageMonitor
	channelService    ChannelService
	permissionService PermissionService
	userService       UserService
	configService     ConfigService

	// Discord session
	session *discordgo.Session

	// Configuration
	config *config.Config
	logger *log.Logger

	// Command integration
	commandIntegration *TTSCommandIntegration

	// System state
	isRunning bool
}

// NewTTSSystem creates a new TTS system with all components initialized
func NewTTSSystem(session *discordgo.Session, cfg *config.Config, logger *log.Logger) (*TTSSystem, error) {
	if session == nil {
		return nil, fmt.Errorf("Discord session cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Initialize storage service
	storageService, err := NewStorageService("./data")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage service: %w", err)
	}

	// Initialize core services
	messageQueue := NewMessageQueue()
	userService := NewUserService(storageService)
	sessionWrapper := NewDiscordSessionWrapper(session)
	permissionService := NewPermissionService(sessionWrapper, storageService, logger)

	configService := NewConfigService(storageService, cfg.TTS)
	channelService := NewChannelService(storageService, sessionWrapper, permissionService)

	// Initialize TTS manager - using Google Cloud TTS
	ttsManager, err := NewGoogleTTSManager(messageQueue, cfg.TTS.GoogleCloudCredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TTS manager: %w", err)
	}
	logger.Println("Using Google Cloud TTS Manager")

	// Initialize voice manager - this will be shared with the integration
	voiceManager := NewVoiceManager(session)
	logger.Printf("Created shared voice manager instance: %p", voiceManager)

	// Initialize TTS processor
	ttsProcessor := NewTTSProcessor(ttsManager, voiceManager, messageQueue, configService, userService)

	// Initialize message monitor
	messageMonitor := NewMessageMonitor(session, channelService, userService, messageQueue, logger)

	// Create command integration (after TTS processor is created)
	commandIntegration, err := NewTTSCommandIntegration(session, storageService, voiceManager, ttsProcessor, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize command integration: %w", err)
	}

	system := &TTSSystem{
		ttsManager:         ttsManager,
		voiceManager:       voiceManager,
		messageQueue:       messageQueue,
		ttsProcessor:       ttsProcessor,
		messageMonitor:     messageMonitor,
		channelService:     channelService,
		permissionService:  permissionService,
		userService:        userService,
		configService:      configService,
		session:            session,
		config:             cfg,
		logger:             logger,
		commandIntegration: commandIntegration,
		isRunning:          false,
	}

	return system, nil
}

// Start initializes and starts all TTS system components
func (sys *TTSSystem) Start() error {
	if sys.isRunning {
		return fmt.Errorf("TTS system is already running")
	}

	sys.logger.Println("Starting TTS system...")

	// Clean up any stale pairings from previous sessions
	if err := sys.cleanupStalePairings(); err != nil {
		sys.logger.Printf("Warning: Failed to clean up stale pairings: %v", err)
	}

	// Start TTS processor
	if err := sys.ttsProcessor.Start(); err != nil {
		return fmt.Errorf("failed to start TTS processor: %w", err)
	}

	// Message monitor starts automatically when created

	sys.isRunning = true
	sys.logger.Println("TTS system started successfully")

	return nil
}

// Stop gracefully shuts down all TTS system components
func (sys *TTSSystem) Stop() error {
	if !sys.isRunning {
		return fmt.Errorf("TTS system is not running")
	}

	sys.logger.Println("Stopping TTS system...")

	// Stop message monitor
	sys.messageMonitor.Stop()

	// Stop TTS processor
	if err := sys.ttsProcessor.Stop(); err != nil {
		sys.logger.Printf("Error stopping TTS processor: %v", err)
	}

	// Disconnect from all voice channels
	activeConnections := sys.voiceManager.GetActiveConnections()
	for _, guildID := range activeConnections {
		if err := sys.voiceManager.LeaveChannel(guildID); err != nil {
			sys.logger.Printf("Error disconnecting from guild %s: %v", guildID, err)
		}
	}

	sys.isRunning = false
	sys.logger.Println("TTS system stopped successfully")

	return nil
}

// GetCommandIntegration returns the command integration for registering slash commands
func (sys *TTSSystem) GetCommandIntegration() *TTSCommandIntegration {
	return sys.commandIntegration
}

// GetVoiceManager returns the voice manager for direct access
func (sys *TTSSystem) GetVoiceManager() VoiceManager {
	return sys.voiceManager
}

// GetChannelService returns the channel service for direct access
func (sys *TTSSystem) GetChannelService() ChannelService {
	return sys.channelService
}

// GetUserService returns the user service for direct access
func (sys *TTSSystem) GetUserService() UserService {
	return sys.userService
}

// GetConfigService returns the config service for direct access
func (sys *TTSSystem) GetConfigService() ConfigService {
	return sys.configService
}

// GetTTSProcessor returns the TTS processor for direct access
func (sys *TTSSystem) GetTTSProcessor() TTSProcessor {
	return sys.ttsProcessor
}

// IsRunning returns whether the TTS system is currently running
func (sys *TTSSystem) IsRunning() bool {
	return sys.isRunning
}

// cleanupStalePairings removes channel pairings that exist but have no active voice connection
func (sys *TTSSystem) cleanupStalePairings() error {
	sys.logger.Println("Cleaning up stale channel pairings from previous sessions...")

	// Get all active voice connections
	activeConnections := sys.voiceManager.GetActiveConnections()
	activeGuilds := make(map[string]bool)
	for _, guildID := range activeConnections {
		activeGuilds[guildID] = true
	}

	// This is a simplified cleanup - in a full implementation, you'd want to
	// iterate through all stored pairings and check if they're still valid
	// For now, we'll just log that cleanup is happening
	sys.logger.Printf("Found %d active voice connections", len(activeConnections))

	return nil
}
