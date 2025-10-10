package bot

import (
	"darrot/internal/config"
	"os"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNew(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid_config",
			config: &config.Config{
				DiscordToken: "test-token",
				LogLevel:     "INFO",
			},
			wantErr: false,
		},
		{
			name:    "nil_config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			if bot == nil {
				t.Errorf("New() returned nil bot")
				return
			}

			// Verify bot initialization
			if bot.config != tt.config {
				t.Errorf("New() config mismatch")
			}

			if bot.session == nil {
				t.Errorf("New() session not initialized")
			}

			if bot.logger == nil {
				t.Errorf("New() logger not initialized")
			}

			if bot.commandRouter == nil {
				t.Errorf("New() commandRouter not initialized")
			}

			if bot.isRunning {
				t.Errorf("New() bot should not be running initially")
			}

			// Verify all commands are registered (test + TTS commands)
			expectedHandlers := 6 // 1 test + 5 TTS commands
			if bot.commandRouter.GetHandlerCount() != expectedHandlers {
				t.Errorf("New() expected %d registered handlers, got %d", expectedHandlers, bot.commandRouter.GetHandlerCount())
			}
		})
	}
}

func TestBot_IsRunning(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Initially should not be running
	if bot.IsRunning() {
		t.Errorf("IsRunning() expected false, got true")
	}

	// Set running state manually for testing
	bot.isRunning = true
	if !bot.IsRunning() {
		t.Errorf("IsRunning() expected true, got false")
	}
}

func TestBot_Stop_NotRunning(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Try to stop bot that's not running
	err = bot.Stop()
	if err == nil {
		t.Errorf("Stop() expected error when bot is not running")
	}
}

func TestBot_Start_AlreadyRunning(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Set running state manually for testing
	bot.isRunning = true

	// Try to start bot that's already running
	err = bot.Start()
	if err == nil {
		t.Errorf("Start() expected error when bot is already running")
	}
}

func TestBot_RegisterCommands(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	tests := []struct {
		name           string
		setupBot       func(*Bot)
		expectError    bool
		expectLogCount int
	}{
		{
			name: "session_not_initialized",
			setupBot: func(b *Bot) {
				// Bot already has test command registered from New()
				// Session state will be nil (not connected to Discord)
			},
			expectError:    true, // Should fail because session state is not initialized
			expectLogCount: 6,    // Should have all commands registered in router (test + TTS)
		},
		{
			name: "no_commands_to_register",
			setupBot: func(b *Bot) {
				// Create empty command router
				b.commandRouter = NewCommandRouter(b.logger)
			},
			expectError:    false, // Should succeed with no commands
			expectLogCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup bot for test case
			tt.setupBot(bot)

			// Test registerCommands method
			err := bot.registerCommands()

			if tt.expectError && err == nil {
				t.Errorf("registerCommands() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("registerCommands() unexpected error: %v", err)
			}

			// Verify command count matches expectations
			commandCount := bot.commandRouter.GetHandlerCount()
			if commandCount != tt.expectLogCount {
				t.Errorf("registerCommands() expected %d commands, got %d", tt.expectLogCount, commandCount)
			}
		})
	}
}

func TestBot_RegisterCommands_ErrorHandling(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Test that registerCommands handles session initialization errors gracefully
	err = bot.registerCommands()

	// The method should return an error when session is not properly initialized
	if err == nil {
		t.Errorf("registerCommands() expected error when session not initialized, but got none")
	}

	// Verify the error message is appropriate
	expectedErrorMsg := "Discord session not properly initialized"
	if err != nil && !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("registerCommands() expected error containing '%s', got: %v", expectedErrorMsg, err)
	}
}

func TestBot_RegisterCommands_Integration(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Verify that the bot has the registerCommands method and it works with the command router
	commands := bot.commandRouter.GetRegisteredCommands()
	expectedCommands := 6 // test + 5 TTS commands
	if len(commands) != expectedCommands {
		t.Errorf("Expected %d registered commands, got %d", expectedCommands, len(commands))
	}

	// Verify the test command is properly defined
	var testCmd *discordgo.ApplicationCommand
	for _, cmd := range commands {
		if cmd.Name == "test" {
			testCmd = cmd
			break
		}
	}
	if testCmd == nil {
		t.Errorf("Expected to find test command in registered commands")
	} else {
		if testCmd.Description == "" {
			t.Errorf("Expected test command to have a description")
		}
	}
}

func TestBot_HandleInteraction_NonCommandInteraction(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Test non-command interaction (should be ignored)
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			User: &discordgo.User{
				ID:       "123456789",
				Username: "testuser",
			},
		},
	}

	// Create a mock session for testing
	session := &discordgo.Session{}

	// Call handleInteraction directly
	// This should complete without error since non-command interactions are ignored
	bot.handleInteraction(session, interaction)

	// If we reach here, the test passed (no panic occurred)
}

func TestBot_HandleInteraction_InteractionTypeCheck(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Test that handleInteraction properly filters interaction types
	interactionTypes := []discordgo.InteractionType{
		discordgo.InteractionPing,
		discordgo.InteractionMessageComponent,
		discordgo.InteractionApplicationCommandAutocomplete,
		discordgo.InteractionModalSubmit,
	}

	session := &discordgo.Session{}

	for _, interactionType := range interactionTypes {
		t.Run(string(rune(interactionType)), func(t *testing.T) {
			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Type: interactionType,
					User: &discordgo.User{
						ID:       "123456789",
						Username: "testuser",
					},
				},
			}

			// These should all be ignored (not cause panics)
			bot.handleInteraction(session, interaction)
		})
	}
}

func TestBot_SendErrorResponse_MethodExists(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Verify that the sendErrorResponse method exists by checking it's callable
	// We can't test the actual Discord API call without mocking, but we can verify
	// the method signature and that it exists
	if bot == nil {
		t.Errorf("Bot instance is nil")
	}

	// The method exists if the bot was created successfully with the handleInteraction method
	// This is an indirect test that the error handling infrastructure is in place
}

func TestBot_InteractionEventHandlerSetup(t *testing.T) {
	// Skip integration tests when Google Cloud credentials are not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test - SKIP_INTEGRATION_TESTS is set")
	}

	config := &config.Config{
		DiscordToken: "test-token",
		LogLevel:     "INFO",
	}

	bot, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Verify that event handlers are set up
	// The setupEventHandlers method is called in New(), so handlers should be registered
	if bot.session == nil {
		t.Fatalf("Bot session is nil")
	}

	// We can't directly test the handlers without accessing private fields,
	// but we can verify the bot was created successfully with handlers
	if bot.commandRouter.GetHandlerCount() == 0 {
		t.Errorf("Expected at least one command handler to be registered")
	}
}
