package scenarios

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	acceptance "acceptance"
)

// NewConcurrentMessageProcessingScenario creates a new concurrent message processing test scenario
func NewConcurrentMessageProcessingScenario() acceptance.TestScenario {
	return &ConcurrentMessageProcessingScenario{
		guildID:        "test-guild-123",
		textChannelID:  "test-text-channel-456",
		voiceChannelID: "test-voice-channel-789",
		userIDs:        []string{"user-1", "user-2", "user-3", "user-4", "user-5"},
		testMessages: []string{
			"Hello from user one",
			"This is user two speaking",
			"User three checking in",
			"Fourth user message here",
			"Fifth user saying hello",
		},
		numUsers: 5,
	}
}

// ConcurrentMessageProcessingScenario tests simultaneous user message processing
type ConcurrentMessageProcessingScenario struct {
	guildID        string
	textChannelID  string
	voiceChannelID string
	userIDs        []string
	testMessages   []string
	numUsers       int
}

func (s *ConcurrentMessageProcessingScenario) Name() string {
	return "ConcurrentMessageProcessing"
}

func (s *ConcurrentMessageProcessingScenario) Description() string {
	return "Tests that the bot can handle multiple users sending messages simultaneously without conflicts"
}

func (s *ConcurrentMessageProcessingScenario) Requirements() []string {
	return []string{"3.1"}
}

func (s *ConcurrentMessageProcessingScenario) Timeout() time.Duration {
	return 60 * time.Second
}

func (s *ConcurrentMessageProcessingScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up concurrent message processing test with %d users", s.numUsers)

	// Create test guild and channels
	guild := env.MockServer.CreateGuild(s.guildID)
	textChannel := guild.CreateTextChannel("general")
	voiceChannel := guild.CreateVoiceChannel("General Voice")

	s.textChannelID = textChannel.GetID()
	s.voiceChannelID = voiceChannel.GetID()

	// Create multiple test users
	users := make([]acceptance.MockUser, s.numUsers)
	for i := 0; i < s.numUsers; i++ {
		user := env.MockServer.SimulateUser(s.userIDs[i])
		guild.AddUser(user)
		users[i] = user
	}

	// Store test data
	env.SetTestData("guild", guild)
	env.SetTestData("textChannel", textChannel)
	env.SetTestData("voiceChannel", voiceChannel)
	env.SetTestData("users", users)

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

	// Join voice channel
	if err := users[0].SendSlashCommand("darrot-join", map[string]interface{}{
		"channel": s.voiceChannelID,
	}); err != nil {
		return fmt.Errorf("failed to send darrot-join command: %w", err)
	}

	// Wait for join to complete
	time.Sleep(3 * time.Second)

	env.LogInfo("Bot setup complete, ready for concurrent testing")
	return nil
}

func (s *ConcurrentMessageProcessingScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing concurrent message processing test")

	// Get test objects
	usersData, _ := env.GetTestData("users")
	users := usersData.([]acceptance.MockUser)

	// Send messages concurrently from all users
	var wg sync.WaitGroup
	errorChan := make(chan error, s.numUsers)

	for i := 0; i < s.numUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()

			user := users[userIndex]
			message := s.testMessages[userIndex]

			env.LogInfo("User %s sending message: %s", user.GetID(), message)

			// Send message
			if err := user.SendMessage(s.textChannelID, message); err != nil {
				errorChan <- fmt.Errorf("user %s failed to send message: %w", user.GetID(), err)
				return
			}

			// Add small random delay to simulate realistic timing
			time.Sleep(time.Duration(userIndex*100) * time.Millisecond)
		}(i)
	}

	// Wait for all messages to be sent
	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	env.LogInfo("All %d users sent messages concurrently", s.numUsers)

	// Wait for bot to process all messages
	time.Sleep(15 * time.Second)

	return nil
}

func (s *ConcurrentMessageProcessingScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating concurrent message processing results")

	// Check bot logs for message processing
	logs := env.BotInstance.GetLogs()
	messagesProcessed := 0
	ttsActivity := 0

	for _, log := range logs {
		logLower := strings.ToLower(log.Message)

		// Count message processing
		if strings.Contains(logLower, "message") {
			messagesProcessed++
		}

		// Count TTS activity
		if strings.Contains(logLower, "tts") || strings.Contains(logLower, "audio") {
			ttsActivity++
		}
	}

	env.LogInfo("Found %d message processing entries and %d TTS activity entries in logs",
		messagesProcessed, ttsActivity)

	// Stop audio capture and analyze
	audioSession, err := env.AudioCapture.StopCapture(s.voiceChannelID)
	if err != nil {
		env.LogError("Failed to stop audio capture: %v", err)
	} else if audioSession != nil {
		env.LogInfo("Audio capture completed: %d packets, duration: %v",
			audioSession.PacketCount, audioSession.Duration)
	}

	// Validate that we have evidence of concurrent processing
	if messagesProcessed == 0 && ttsActivity == 0 {
		return fmt.Errorf("no evidence of message or TTS processing found in bot logs")
	}

	// Check that bot handled the load without errors
	errorCount := 0
	for _, log := range logs {
		if log.Level == "ERROR" || log.Level == "FATAL" {
			errorCount++
			env.LogError("Bot error during concurrent test: %s", log.Message)
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("bot encountered %d errors during concurrent message processing", errorCount)
	}

	env.LogInfo("Concurrent message processing test passed")
	return nil
}

func (s *ConcurrentMessageProcessingScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up concurrent message processing test")

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
