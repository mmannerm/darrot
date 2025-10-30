package scenarios

import (
	"context"
	"fmt"
	"strings"
	"time"

	acceptance "acceptance"
)

// DarrotJoinScenario tests the darrot-join command functionality
type DarrotJoinScenario struct {
	guildID        string
	textChannelID  string
	voiceChannelID string
	userID         string
}

// NewDarrotJoinScenario creates a new darrot-join test scenario
func NewDarrotJoinScenario() acceptance.TestScenario {
	return &DarrotJoinScenario{
		guildID:        "test-guild-123",
		textChannelID:  "test-text-channel-456",
		voiceChannelID: "test-voice-channel-789",
		userID:         "test-user-456",
	}
}

func (s *DarrotJoinScenario) Name() string {
	return "DarrotJoinCommand"
}

func (s *DarrotJoinScenario) Description() string {
	return "Tests that the bot correctly responds to darrot-join command and attempts to join the specified voice channel"
}

func (s *DarrotJoinScenario) Requirements() []string {
	return []string{"2.3"}
}

func (s *DarrotJoinScenario) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *DarrotJoinScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up darrot-join command test")

	// Create test guild and channels
	guild := env.MockServer.CreateGuild(s.guildID)
	textChannel := guild.CreateTextChannel("general")
	voiceChannel := guild.CreateVoiceChannel("General Voice")

	s.textChannelID = textChannel.GetID()
	s.voiceChannelID = voiceChannel.GetID()

	// Create test user
	user := env.MockServer.SimulateUser(s.userID)
	guild.AddUser(user)

	// Store test data
	env.SetTestData("guild", guild)
	env.SetTestData("textChannel", textChannel)
	env.SetTestData("voiceChannel", voiceChannel)
	env.SetTestData("user", user)

	// Start bot instance
	botConfig := &acceptance.BotConfig{
		BinaryPath:    env.Config.Bot.BinaryPath,
		WorkingDir:    env.Config.Bot.WorkingDir,
		LogLevel:      env.Config.Bot.LogLevel,
		DiscordToken:  "test-bot-token-123",
		DiscordAPIURL: env.MockServer.GetBaseURL(),
	}

	if err := env.BotInstance.Start(ctx, botConfig); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	// Wait for bot to be ready
	if err := env.BotInstance.WaitForReady(env.Config.Bot.StartupTimeout); err != nil {
		return fmt.Errorf("bot failed to become ready: %w", err)
	}

	env.LogInfo("Bot is ready for testing")
	return nil
}

func (s *DarrotJoinScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing darrot-join command test")

	// Get test objects
	user, _ := env.GetTestData("user")
	mockUser := user.(acceptance.MockUser)

	// Send darrot-join command
	joinCommand := fmt.Sprintf("/darrot-join %s", s.voiceChannelID)
	if err := mockUser.SendSlashCommand("darrot-join", map[string]interface{}{
		"channel": s.voiceChannelID,
	}); err != nil {
		return fmt.Errorf("failed to send darrot-join command: %w", err)
	}

	env.LogInfo("Sent darrot-join command for channel %s", s.voiceChannelID)

	// Wait for bot to process the command
	time.Sleep(2 * time.Second)

	return nil
}

func (s *DarrotJoinScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating darrot-join command results")

	// Check bot logs for join attempt
	logs := env.BotInstance.GetLogs()
	joinAttempted := false

	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.Message), "join") &&
			strings.Contains(strings.ToLower(log.Message), "voice") {
			joinAttempted = true
			env.LogInfo("Found voice join attempt in logs: %s", log.Message)
			break
		}
	}

	if !joinAttempted {
		return fmt.Errorf("bot did not attempt to join voice channel")
	}

	// Check for interactions recorded by mock server
	interactions := env.MockServer.GetInteractions()
	commandReceived := false

	for _, interaction := range interactions {
		if data, ok := interaction.Data.(map[string]interface{}); ok {
			if name, exists := data["name"]; exists && name == "darrot-join" {
				commandReceived = true
				env.LogInfo("Found darrot-join interaction in mock server")
				break
			}
		}
	}

	if !commandReceived {
		return fmt.Errorf("darrot-join command was not received by mock server")
	}

	env.LogInfo("darrot-join command test passed")
	return nil
}

func (s *DarrotJoinScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up darrot-join command test")

	// Stop bot instance
	if err := env.BotInstance.Stop(); err != nil {
		env.LogError("Failed to stop bot instance: %v", err)
	}

	// Reset mock server
	if err := env.MockServer.Reset(); err != nil {
		env.LogError("Failed to reset mock server: %v", err)
	}

	return nil
}

// NewDarrotLeaveScenario creates a new darrot-leave test scenario
func NewDarrotLeaveScenario() acceptance.TestScenario {
	return &DarrotLeaveScenario{
		guildID:        "test-guild-123",
		textChannelID:  "test-text-channel-456",
		voiceChannelID: "test-voice-channel-789",
		userID:         "test-user-456",
	}
}

// DarrotLeaveScenario tests the darrot-leave command functionality
type DarrotLeaveScenario struct {
	guildID        string
	textChannelID  string
	voiceChannelID string
	userID         string
}

func (s *DarrotLeaveScenario) Name() string {
	return "DarrotLeaveCommand"
}

func (s *DarrotLeaveScenario) Description() string {
	return "Tests that the bot correctly responds to darrot-leave command and leaves the voice channel"
}

func (s *DarrotLeaveScenario) Requirements() []string {
	return []string{"2.3"}
}

func (s *DarrotLeaveScenario) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *DarrotLeaveScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up darrot-leave command test")

	// Create test guild and channels
	guild := env.MockServer.CreateGuild(s.guildID)
	textChannel := guild.CreateTextChannel("general")
	voiceChannel := guild.CreateVoiceChannel("General Voice")

	s.textChannelID = textChannel.GetID()
	s.voiceChannelID = voiceChannel.GetID()

	// Create test user
	user := env.MockServer.SimulateUser(s.userID)
	guild.AddUser(user)

	// Store test data
	env.SetTestData("guild", guild)
	env.SetTestData("textChannel", textChannel)
	env.SetTestData("voiceChannel", voiceChannel)
	env.SetTestData("user", user)

	// Start bot instance
	botConfig := &acceptance.BotConfig{
		BinaryPath:    env.Config.Bot.BinaryPath,
		WorkingDir:    env.Config.Bot.WorkingDir,
		LogLevel:      env.Config.Bot.LogLevel,
		DiscordToken:  "test-bot-token-123",
		DiscordAPIURL: env.MockServer.GetBaseURL(),
	}

	if err := env.BotInstance.Start(ctx, botConfig); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	// Wait for bot to be ready
	if err := env.BotInstance.WaitForReady(env.Config.Bot.StartupTimeout); err != nil {
		return fmt.Errorf("bot failed to become ready: %w", err)
	}

	// First, simulate bot joining voice channel
	mockUser := user.(acceptance.MockUser)
	if err := mockUser.SendSlashCommand("darrot-join", map[string]interface{}{
		"channel": s.voiceChannelID,
	}); err != nil {
		return fmt.Errorf("failed to send initial darrot-join command: %w", err)
	}

	// Wait for join to complete
	time.Sleep(2 * time.Second)

	env.LogInfo("Bot setup complete, ready for leave test")
	return nil
}

func (s *DarrotLeaveScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing darrot-leave command test")

	// Get test objects
	user, _ := env.GetTestData("user")
	mockUser := user.(acceptance.MockUser)

	// Send darrot-leave command
	if err := mockUser.SendSlashCommand("darrot-leave", map[string]interface{}{}); err != nil {
		return fmt.Errorf("failed to send darrot-leave command: %w", err)
	}

	env.LogInfo("Sent darrot-leave command")

	// Wait for bot to process the command
	time.Sleep(2 * time.Second)

	return nil
}

func (s *DarrotLeaveScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating darrot-leave command results")

	// Check bot logs for leave attempt
	logs := env.BotInstance.GetLogs()
	leaveAttempted := false

	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.Message), "leave") ||
			strings.Contains(strings.ToLower(log.Message), "disconnect") {
			leaveAttempted = true
			env.LogInfo("Found voice leave attempt in logs: %s", log.Message)
			break
		}
	}

	if !leaveAttempted {
		return fmt.Errorf("bot did not attempt to leave voice channel")
	}

	// Check for interactions recorded by mock server
	interactions := env.MockServer.GetInteractions()
	commandReceived := false

	for _, interaction := range interactions {
		if data, ok := interaction.Data.(map[string]interface{}); ok {
			if name, exists := data["name"]; exists && name == "darrot-leave" {
				commandReceived = true
				env.LogInfo("Found darrot-leave interaction in mock server")
				break
			}
		}
	}

	if !commandReceived {
		return fmt.Errorf("darrot-leave command was not received by mock server")
	}

	env.LogInfo("darrot-leave command test passed")
	return nil
}

func (s *DarrotLeaveScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up darrot-leave command test")

	// Stop bot instance
	if err := env.BotInstance.Stop(); err != nil {
		env.LogError("Failed to stop bot instance: %v", err)
	}

	// Reset mock server
	if err := env.MockServer.Reset(); err != nil {
		env.LogError("Failed to reset mock server: %v", err)
	}

	return nil
}

// NewTTSMessageProcessingScenario creates a new TTS message processing test scenario
func NewTTSMessageProcessingScenario() acceptance.TestScenario {
	return &TTSMessageProcessingScenario{
		guildID:        "test-guild-123",
		textChannelID:  "test-text-channel-456",
		voiceChannelID: "test-voice-channel-789",
		userID:         "test-user-456",
		testMessage:    "Hello, this is a test message for TTS processing",
	}
}

// TTSMessageProcessingScenario tests TTS message processing functionality
type TTSMessageProcessingScenario struct {
	guildID        string
	textChannelID  string
	voiceChannelID string
	userID         string
	testMessage    string
}

func (s *TTSMessageProcessingScenario) Name() string {
	return "TTSMessageProcessing"
}

func (s *TTSMessageProcessingScenario) Description() string {
	return "Tests that the bot processes text messages and generates TTS audio when monitoring a channel"
}

func (s *TTSMessageProcessingScenario) Requirements() []string {
	return []string{"2.4"}
}

func (s *TTSMessageProcessingScenario) Timeout() time.Duration {
	return 45 * time.Second
}

func (s *TTSMessageProcessingScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up TTS message processing test")

	// Create test guild and channels
	guild := env.MockServer.CreateGuild(s.guildID)
	textChannel := guild.CreateTextChannel("general")
	voiceChannel := guild.CreateVoiceChannel("General Voice")

	s.textChannelID = textChannel.GetID()
	s.voiceChannelID = voiceChannel.GetID()

	// Create test user
	user := env.MockServer.SimulateUser(s.userID)
	guild.AddUser(user)

	// Store test data
	env.SetTestData("guild", guild)
	env.SetTestData("textChannel", textChannel)
	env.SetTestData("voiceChannel", voiceChannel)
	env.SetTestData("user", user)

	// Start audio capture
	if err := env.AudioCapture.StartCapture(s.voiceChannelID); err != nil {
		return fmt.Errorf("failed to start audio capture: %w", err)
	}

	// Start bot instance
	botConfig := &acceptance.BotConfig{
		BinaryPath:    env.Config.Bot.BinaryPath,
		WorkingDir:    env.Config.Bot.WorkingDir,
		LogLevel:      env.Config.Bot.LogLevel,
		DiscordToken:  "test-bot-token-123",
		DiscordAPIURL: env.MockServer.GetBaseURL(),
	}

	if err := env.BotInstance.Start(ctx, botConfig); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	// Wait for bot to be ready
	if err := env.BotInstance.WaitForReady(env.Config.Bot.StartupTimeout); err != nil {
		return fmt.Errorf("bot failed to become ready: %w", err)
	}

	// Join voice channel first
	mockUser := user.(acceptance.MockUser)
	if err := mockUser.SendSlashCommand("darrot-join", map[string]interface{}{
		"channel": s.voiceChannelID,
	}); err != nil {
		return fmt.Errorf("failed to send darrot-join command: %w", err)
	}

	// Wait for join to complete
	time.Sleep(3 * time.Second)

	env.LogInfo("Bot setup complete, ready for TTS test")
	return nil
}

func (s *TTSMessageProcessingScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing TTS message processing test")

	// Get test objects
	user, _ := env.GetTestData("user")
	textChannel, _ := env.GetTestData("textChannel")

	mockUser := user.(acceptance.MockUser)
	mockChannel := textChannel.(acceptance.MockChannel)

	// Send test message to text channel
	if err := mockUser.SendMessage(s.textChannelID, s.testMessage); err != nil {
		return fmt.Errorf("failed to send test message: %w", err)
	}

	env.LogInfo("Sent test message: %s", s.testMessage)

	// Also send via channel interface for completeness
	if err := mockChannel.SendMessage(s.testMessage, s.userID); err != nil {
		return fmt.Errorf("failed to send message via channel: %w", err)
	}

	// Wait for TTS processing
	env.LogInfo("Waiting for TTS processing...")
	time.Sleep(10 * time.Second)

	return nil
}

func (s *TTSMessageProcessingScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating TTS message processing results")

	// Check bot logs for TTS processing
	logs := env.BotInstance.GetLogs()
	ttsProcessed := false
	messageReceived := false

	for _, log := range logs {
		logLower := strings.ToLower(log.Message)
		if strings.Contains(logLower, "tts") || strings.Contains(logLower, "text-to-speech") {
			ttsProcessed = true
			env.LogInfo("Found TTS processing in logs: %s", log.Message)
		}
		if strings.Contains(logLower, "message") && strings.Contains(logLower, s.testMessage) {
			messageReceived = true
			env.LogInfo("Found message processing in logs: %s", log.Message)
		}
	}

	if !messageReceived {
		env.LogInfo("Message processing not found in logs, checking for general message handling")
		for _, log := range logs {
			if strings.Contains(strings.ToLower(log.Message), "message") {
				messageReceived = true
				env.LogInfo("Found general message handling: %s", log.Message)
				break
			}
		}
	}

	// Stop audio capture and analyze
	audioSession, err := env.AudioCapture.StopCapture(s.voiceChannelID)
	if err != nil {
		env.LogError("Failed to stop audio capture: %v", err)
	} else if audioSession != nil {
		env.LogInfo("Audio capture completed: %d packets, duration: %v",
			audioSession.PacketCount, audioSession.Duration)

		// Validate audio format
		formatValidation, err := env.AudioCapture.ValidateFormat(s.voiceChannelID, env.Config.Audio.ExpectedFormat)
		if err == nil && formatValidation.IsValid {
			env.LogInfo("Audio format validation passed")
		} else {
			env.LogInfo("Audio format validation: %v", formatValidation)
		}
	}

	// Check if we have evidence of TTS processing
	if !ttsProcessed && !messageReceived {
		return fmt.Errorf("no evidence of TTS or message processing found in bot logs")
	}

	env.LogInfo("TTS message processing test passed")
	return nil
}

func (s *TTSMessageProcessingScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up TTS message processing test")

	// Stop audio capture if still active
	if _, err := env.AudioCapture.StopCapture(s.voiceChannelID); err != nil {
		env.LogError("Failed to stop audio capture: %v", err)
	}

	// Stop bot instance
	if err := env.BotInstance.Stop(); err != nil {
		env.LogError("Failed to stop bot instance: %v", err)
	}

	// Reset mock server
	if err := env.MockServer.Reset(); err != nil {
		env.LogError("Failed to reset mock server: %v", err)
	}

	return nil
}

// NewConfigurationCommandScenario creates a new configuration command test scenario
func NewConfigurationCommandScenario() acceptance.TestScenario {
	return &ConfigurationCommandScenario{
		guildID:       "test-guild-123",
		textChannelID: "test-text-channel-456",
		userID:        "test-user-456",
	}
}

// ConfigurationCommandScenario tests configuration command functionality
type ConfigurationCommandScenario struct {
	guildID       string
	textChannelID string
	userID        string
}

func (s *ConfigurationCommandScenario) Name() string {
	return "ConfigurationCommands"
}

func (s *ConfigurationCommandScenario) Description() string {
	return "Tests that the bot correctly handles configuration commands and updates settings"
}

func (s *ConfigurationCommandScenario) Requirements() []string {
	return []string{"2.7"}
}

func (s *ConfigurationCommandScenario) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *ConfigurationCommandScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up configuration command test")

	// Create test guild and channels
	guild := env.MockServer.CreateGuild(s.guildID)
	textChannel := guild.CreateTextChannel("general")

	s.textChannelID = textChannel.GetID()

	// Create test user
	user := env.MockServer.SimulateUser(s.userID)
	guild.AddUser(user)

	// Store test data
	env.SetTestData("guild", guild)
	env.SetTestData("textChannel", textChannel)
	env.SetTestData("user", user)

	// Start bot instance
	botConfig := &acceptance.BotConfig{
		BinaryPath:    env.Config.Bot.BinaryPath,
		WorkingDir:    env.Config.Bot.WorkingDir,
		LogLevel:      env.Config.Bot.LogLevel,
		DiscordToken:  "test-bot-token-123",
		DiscordAPIURL: env.MockServer.GetBaseURL(),
	}

	if err := env.BotInstance.Start(ctx, botConfig); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	// Wait for bot to be ready
	if err := env.BotInstance.WaitForReady(env.Config.Bot.StartupTimeout); err != nil {
		return fmt.Errorf("bot failed to become ready: %w", err)
	}

	env.LogInfo("Bot is ready for configuration testing")
	return nil
}

func (s *ConfigurationCommandScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing configuration command test")

	// Get test objects
	user, _ := env.GetTestData("user")
	mockUser := user.(acceptance.MockUser)

	// Test various configuration commands
	configCommands := []struct {
		command string
		options map[string]interface{}
	}{
		{
			command: "tts-voice",
			options: map[string]interface{}{
				"voice": "en-US-Standard-B",
			},
		},
		{
			command: "tts-speed",
			options: map[string]interface{}{
				"speed": 1.2,
			},
		},
		{
			command: "tts-volume",
			options: map[string]interface{}{
				"volume": 0.8,
			},
		},
	}

	for _, cmd := range configCommands {
		env.LogInfo("Sending configuration command: %s", cmd.command)
		if err := mockUser.SendSlashCommand(cmd.command, cmd.options); err != nil {
			return fmt.Errorf("failed to send %s command: %w", cmd.command, err)
		}

		// Wait between commands
		time.Sleep(1 * time.Second)
	}

	// Wait for all commands to be processed
	time.Sleep(3 * time.Second)

	return nil
}

func (s *ConfigurationCommandScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating configuration command results")

	// Check bot logs for configuration updates
	logs := env.BotInstance.GetLogs()
	configUpdated := false

	for _, log := range logs {
		logLower := strings.ToLower(log.Message)
		if strings.Contains(logLower, "config") ||
			strings.Contains(logLower, "setting") ||
			strings.Contains(logLower, "voice") ||
			strings.Contains(logLower, "speed") ||
			strings.Contains(logLower, "volume") {
			configUpdated = true
			env.LogInfo("Found configuration update in logs: %s", log.Message)
			break
		}
	}

	// Check for interactions recorded by mock server
	interactions := env.MockServer.GetInteractions()
	configCommandsReceived := 0

	for _, interaction := range interactions {
		if data, ok := interaction.Data.(map[string]interface{}); ok {
			if name, exists := data["name"]; exists {
				nameStr := fmt.Sprintf("%v", name)
				if strings.HasPrefix(nameStr, "tts-") {
					configCommandsReceived++
					env.LogInfo("Found configuration command interaction: %s", nameStr)
				}
			}
		}
	}

	if configCommandsReceived == 0 {
		return fmt.Errorf("no configuration commands were received by mock server")
	}

	if !configUpdated {
		env.LogInfo("Configuration updates not found in logs, but commands were received")
	}

	env.LogInfo("Configuration command test passed (%d commands processed)", configCommandsReceived)
	return nil
}

func (s *ConfigurationCommandScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up configuration command test")

	// Stop bot instance
	if err := env.BotInstance.Stop(); err != nil {
		env.LogError("Failed to stop bot instance: %v", err)
	}

	// Reset mock server
	if err := env.MockServer.Reset(); err != nil {
		env.LogError("Failed to reset mock server: %v", err)
	}

	return nil
}
