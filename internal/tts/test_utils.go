package tts

import (
	"testing"
)

// integrationTestEnvironment provides a complete test environment for integration tests
type integrationTestEnvironment struct {
	voiceManager      *mockVoiceManagerIntegration
	ttsManager        *mockTTSManagerIntegration
	messageQueue      *mockMessageQueueIntegration
	channelService    *mockChannelServiceIntegration
	permissionService *mockPermissionServiceIntegration
	userService       *mockUserServiceIntegration
	configService     *mockConfigServiceIntegration

	joinHandler    *mockJoinHandlerIntegration
	leaveHandler   *mockLeaveHandlerIntegration
	controlHandler *mockControlHandlerIntegration
	configHandler  *mockConfigHandlerIntegration

	cleanup func()
}

// setupIntegrationTestEnvironment creates a complete test environment
func setupIntegrationTestEnvironment(t *testing.T) *integrationTestEnvironment {
	messageQueue := newMockMessageQueueIntegration()
	ttsManager := newMockTTSManagerIntegration()
	ttsManager.setMessageQueue(messageQueue)

	env := &integrationTestEnvironment{
		voiceManager:      newMockVoiceManagerIntegration(),
		ttsManager:        ttsManager,
		messageQueue:      messageQueue,
		channelService:    newMockChannelServiceIntegration(),
		permissionService: newMockPermissionServiceIntegration(),
		userService:       newMockUserServiceIntegration(),
		configService:     newMockConfigServiceIntegration(),
	}

	// Create command handlers
	env.joinHandler = newMockJoinHandlerIntegration(env)
	env.leaveHandler = newMockLeaveHandlerIntegration(env)
	env.controlHandler = newMockControlHandlerIntegration(env)
	env.configHandler = newMockConfigHandlerIntegration(env)

	env.cleanup = func() {
		// Clean up any resources
		for guildID := range env.voiceManager.connections {
			_ = env.voiceManager.LeaveChannel(guildID)
		}
	}

	return env
}

// shouldProcessMessage determines if a message should be processed based on opt-in status
func (env *integrationTestEnvironment) shouldProcessMessage(message *QueuedMessage) bool {
	isOptedIn, err := env.userService.IsOptedIn(message.UserID, message.GuildID)
	if err != nil {
		return false
	}
	return isOptedIn
}

// errorTestEnvironment provides a test environment for error scenario testing
type errorTestEnvironment struct {
	voiceManager      *mockVoiceManagerError
	ttsManager        *mockTTSManagerError
	messageQueue      MessageQueue
	channelService    *mockChannelServiceError
	permissionService *mockPermissionServiceError
	userService       UserService
	errorRecovery     *ErrorRecoveryManager

	joinHandler    *mockJoinHandlerIntegration
	controlHandler *mockControlHandlerIntegration

	cleanup func()
}

// setupErrorTestEnvironment creates a test environment for error scenarios
func setupErrorTestEnvironment(t *testing.T) *errorTestEnvironment {
	voiceManager := newMockVoiceManagerError()
	ttsManager := newMockTTSManagerError()
	messageQueue := newMockMessageQueueIntegration()
	channelService := newMockChannelServiceError()
	permissionService := newMockPermissionServiceError()
	userService := newMockUserServiceIntegration()
	configService := newMockConfigServiceIntegration()

	errorRecovery := NewErrorRecoveryManager(voiceManager, ttsManager, messageQueue, configService)

	env := &errorTestEnvironment{
		voiceManager:      voiceManager,
		ttsManager:        ttsManager,
		messageQueue:      messageQueue,
		channelService:    channelService,
		permissionService: permissionService,
		userService:       userService,
		errorRecovery:     errorRecovery,
	}

	// Create integration test environment for handlers
	integrationEnv := &integrationTestEnvironment{
		voiceManager:      newMockVoiceManagerIntegration(),
		ttsManager:        newMockTTSManagerIntegration(),
		messageQueue:      messageQueue,
		channelService:    newMockChannelServiceIntegration(),
		permissionService: newMockPermissionServiceIntegration(),
		userService:       userService,
		configService:     configService,
	}

	env.joinHandler = newMockJoinHandlerIntegration(integrationEnv)
	env.controlHandler = newMockControlHandlerIntegration(integrationEnv)

	env.cleanup = func() {
		// Clean up any resources
		for guildID := range voiceManager.connections {
			_ = voiceManager.LeaveChannel(guildID)
		}
	}

	return env
}

// shouldProcessMessage determines if a message should be processed
func (env *errorTestEnvironment) shouldProcessMessage(message *QueuedMessage) bool {
	isOptedIn, err := env.userService.IsOptedIn(message.UserID, message.GuildID)
	if err != nil {
		return false
	}
	return isOptedIn
}

// endToEndTestEnvironment provides a test environment for end-to-end testing
type endToEndTestEnvironment struct {
	voiceManager   VoiceManager
	ttsManager     TTSManager
	messageQueue   MessageQueue
	channelService ChannelService
	userService    UserService
	cleanup        func()
}

// setupEndToEndTestEnvironment creates an end-to-end test environment
func setupEndToEndTestEnvironment(t *testing.T) *endToEndTestEnvironment {
	env := &endToEndTestEnvironment{
		voiceManager:   newMockVoiceManagerIntegration(),
		ttsManager:     newMockTTSManagerIntegration(),
		messageQueue:   newMockMessageQueueIntegration(),
		channelService: newMockChannelServiceIntegration(),
		userService:    newMockUserServiceIntegration(),
	}

	env.cleanup = func() {
		// Clean up any resources
		if vm, ok := env.voiceManager.(*mockVoiceManagerIntegration); ok {
			_ = vm.Cleanup()
		}
	}

	return env
}

// processMessageQueueWithAudio processes the message queue and simulates audio playback
func (env *endToEndTestEnvironment) processMessageQueueWithAudio(guildID string) error {
	for {
		message, err := env.messageQueue.Dequeue(guildID)
		if err != nil {
			return err
		}
		if message == nil {
			break // Queue is empty
		}

		// Get TTS configuration
		var config TTSConfig
		if tm, ok := env.ttsManager.(*mockTTSManagerIntegration); ok {
			config = tm.getVoiceConfig(guildID)
		} else {
			config = TTSConfig{
				Voice:  DefaultVoice,
				Speed:  DefaultTTSSpeed,
				Volume: DefaultTTSVolume,
				Format: AudioFormatDCA,
			}
		}

		// Convert to speech
		audioData, err := env.ttsManager.ConvertToSpeech(message.Content, config.Voice, config)
		if err != nil {
			// Log error but continue processing
			continue
		}

		// Play audio
		err = env.voiceManager.PlayAudio(guildID, audioData)
		if err != nil {
			// Log error but continue processing
			continue
		}

		// Simulate processing time
		// time.Sleep(10 * time.Millisecond) // Commented out to speed up tests
	}

	return nil
}

// shouldProcessMessage determines if a message should be processed
func (env *endToEndTestEnvironment) shouldProcessMessage(message *QueuedMessage) bool {
	isOptedIn, err := env.userService.IsOptedIn(message.UserID, message.GuildID)
	if err != nil {
		return false
	}
	return isOptedIn
}
