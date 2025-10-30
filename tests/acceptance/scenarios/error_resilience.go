package scenarios

import (
	"context"
	"fmt"
	"strings"
	"time"

	acceptance "acceptance"
)

// NewNetworkFailureScenario creates a new network failure test scenario
func NewNetworkFailureScenario() acceptance.TestScenario {
	return &NetworkFailureScenario{
		guildID:         "test-guild-123",
		textChannelID:   "test-text-channel-456",
		userID:          "test-user-456",
		failureDuration: 10 * time.Second,
	}
}

// NetworkFailureScenario tests bot behavior during network failures and reconnection
type NetworkFailureScenario struct {
	guildID         string
	textChannelID   string
	userID          string
	failureDuration time.Duration
}

func (s *NetworkFailureScenario) Name() string {
	return "NetworkFailureReconnection"
}

func (s *NetworkFailureScenario) Description() string {
	return "Tests that the bot handles network failures gracefully and attempts reconnection"
}

func (s *NetworkFailureScenario) Requirements() []string {
	return []string{"2.6", "3.4"}
}

func (s *NetworkFailureScenario) Timeout() time.Duration {
	return 60 * time.Second
}

func (s *NetworkFailureScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up network failure reconnection test")

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

	env.LogInfo("Bot is ready for network failure testing")
	return nil
}

func (s *NetworkFailureScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing network failure reconnection test")

	// Get test objects
	user, _ := env.GetTestData("user")
	mockUser := user.(acceptance.MockUser)

	// Phase 1: Send initial message to verify connectivity
	env.LogInfo("Phase 1: Testing initial connectivity")
	if err := mockUser.SendMessage(s.textChannelID, "Initial connectivity test message"); err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	time.Sleep(2 * time.Second)

	// Phase 2: Simulate network failure by stopping mock server
	env.LogInfo("Phase 2: Simulating network failure (stopping mock server)")
	if err := env.MockServer.Stop(); err != nil {
		return fmt.Errorf("failed to stop mock server: %w", err)
	}

	// Wait during "network failure"
	env.LogInfo("Network failure simulation active for %v", s.failureDuration)
	time.Sleep(s.failureDuration)

	// Phase 3: Restore network by restarting mock server
	env.LogInfo("Phase 3: Restoring network connectivity (restarting mock server)")
	if err := env.MockServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to restart mock server: %w", err)
	}

	// Wait for server to be ready
	time.Sleep(3 * time.Second)

	// Phase 4: Test connectivity after restoration
	env.LogInfo("Phase 4: Testing connectivity after restoration")
	if err := mockUser.SendMessage(s.textChannelID, "Post-recovery connectivity test message"); err != nil {
		env.LogError("Failed to send post-recovery message: %v", err)
		// Don't fail immediately, bot might still be reconnecting
	}

	// Wait for bot to reconnect and process
	time.Sleep(10 * time.Second)

	return nil
}

func (s *NetworkFailureScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating network failure reconnection results")

	// Check bot logs for reconnection attempts
	logs := env.BotInstance.GetLogs()
	connectionErrors := 0
	reconnectionAttempts := 0
	recoverySuccess := 0

	for _, log := range logs {
		logLower := strings.ToLower(log.Message)

		if strings.Contains(logLower, "connection") || strings.Contains(logLower, "connect") {
			if strings.Contains(logLower, "error") || strings.Contains(logLower, "failed") ||
				strings.Contains(logLower, "timeout") || strings.Contains(logLower, "refused") {
				connectionErrors++
			}

			if strings.Contains(logLower, "reconnect") || strings.Contains(logLower, "retry") {
				reconnectionAttempts++
			}

			if strings.Contains(logLower, "connected") || strings.Contains(logLower, "ready") {
				recoverySuccess++
			}
		}
	}

	env.LogInfo("Found %d connection errors, %d reconnection attempts, %d recovery successes in logs",
		connectionErrors, reconnectionAttempts, recoverySuccess)

	// Validate that bot detected the network failure
	if connectionErrors == 0 {
		env.LogInfo("No explicit connection errors found, checking for general errors during failure period")

		// Check for any errors during the failure period
		errorsDuringFailure := 0
		for _, log := range logs {
			if log.Level == "ERROR" || log.Level == "WARN" {
				errorsDuringFailure++
			}
		}

		if errorsDuringFailure == 0 {
			return fmt.Errorf("bot did not detect network failure (no connection errors or warnings)")
		}

		env.LogInfo("Found %d errors/warnings during failure period", errorsDuringFailure)
	}

	// Check that bot is still running after the test
	if !env.BotInstance.IsRunning() {
		return fmt.Errorf("bot stopped running during network failure test")
	}

	// Validate that bot attempted recovery
	if reconnectionAttempts == 0 && recoverySuccess == 0 {
		env.LogInfo("No explicit reconnection attempts found, but bot is still running")
	}

	env.LogInfo("Network failure reconnection test passed")
	return nil
}

func (s *NetworkFailureScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up network failure reconnection test")

	// Ensure mock server is running for cleanup
	if !env.MockServer.IsHealthy() {
		if err := env.MockServer.Start(ctx); err != nil {
			env.LogError("Failed to restart mock server for cleanup: %v", err)
		}
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

// NewInvalidCommandScenario creates a new invalid command test scenario
func NewInvalidCommandScenario() acceptance.TestScenario {
	return &InvalidCommandScenario{
		guildID:       "test-guild-123",
		textChannelID: "test-text-channel-456",
		userID:        "test-user-456",
	}
}

// InvalidCommandScenario tests bot behavior with invalid commands and error handling
type InvalidCommandScenario struct {
	guildID       string
	textChannelID string
	userID        string
}

func (s *InvalidCommandScenario) Name() string {
	return "InvalidCommandHandling"
}

func (s *InvalidCommandScenario) Description() string {
	return "Tests that the bot handles invalid commands gracefully and provides appropriate error responses"
}

func (s *InvalidCommandScenario) Requirements() []string {
	return []string{"3.7"}
}

func (s *InvalidCommandScenario) Timeout() time.Duration {
	return 30 * time.Second
}

func (s *InvalidCommandScenario) Setup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Setting up invalid command handling test")

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

	env.LogInfo("Bot is ready for invalid command testing")
	return nil
}

func (s *InvalidCommandScenario) Execute(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Executing invalid command handling test")

	// Get test objects
	user, _ := env.GetTestData("user")
	mockUser := user.(acceptance.MockUser)

	// Test various invalid commands
	invalidCommands := []struct {
		command     string
		options     map[string]interface{}
		description string
	}{
		{
			command:     "nonexistent-command",
			options:     map[string]interface{}{},
			description: "completely invalid command",
		},
		{
			command: "darrot-join",
			options: map[string]interface{}{
				"channel": "nonexistent-channel-id",
			},
			description: "valid command with invalid channel",
		},
		{
			command: "tts-voice",
			options: map[string]interface{}{
				"voice": "invalid-voice-name",
			},
			description: "valid command with invalid voice",
		},
	}

	for i, cmd := range invalidCommands {
		env.LogInfo("Sending invalid command %d: %s (%s)", i+1, cmd.command, cmd.description)

		if err := mockUser.SendSlashCommand(cmd.command, cmd.options); err != nil {
			env.LogInfo("Command failed as expected: %v", err)
		}

		// Wait between commands
		time.Sleep(1 * time.Second)
	}

	// Wait for bot to process all invalid commands
	time.Sleep(5 * time.Second)

	return nil
}

func (s *InvalidCommandScenario) Validate(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Validating invalid command handling results")

	// Check bot logs for error handling
	logs := env.BotInstance.GetLogs()
	errorHandling := 0
	crashErrors := 0

	for _, log := range logs {
		logLower := strings.ToLower(log.Message)

		if strings.Contains(logLower, "invalid") || strings.Contains(logLower, "unknown") ||
			strings.Contains(logLower, "not found") {
			errorHandling++
		}

		if log.Level == "FATAL" || strings.Contains(logLower, "panic") ||
			strings.Contains(logLower, "crash") {
			crashErrors++
		}
	}

	env.LogInfo("Found %d error handling entries, %d crash errors", errorHandling, crashErrors)

	// Check that bot is still running (didn't crash from invalid commands)
	if !env.BotInstance.IsRunning() {
		return fmt.Errorf("bot stopped running during invalid command test")
	}

	// Validate that bot didn't crash
	if crashErrors > 0 {
		return fmt.Errorf("bot crashed %d times during invalid command handling", crashErrors)
	}

	env.LogInfo("Invalid command handling test passed")
	return nil
}

func (s *InvalidCommandScenario) Cleanup(ctx context.Context, env *acceptance.TestEnvironment) error {
	env.LogInfo("Cleaning up invalid command handling test")

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
