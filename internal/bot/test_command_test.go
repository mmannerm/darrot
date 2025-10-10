package bot

import (
	"log"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewTestCommandHandler(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)

	handler := NewTestCommandHandler(logger)

	if handler == nil {
		t.Fatal("NewTestCommandHandler returned nil")
	}

	if handler.logger != logger {
		t.Error("Logger not properly set in handler")
	}
}

func TestTestCommandHandler_Definition(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	handler := NewTestCommandHandler(logger)

	definition := handler.Definition()

	if definition == nil {
		t.Fatal("Definition returned nil")
	}

	if definition.Name != "test" {
		t.Errorf("Expected command name 'test', got '%s'", definition.Name)
	}

	if definition.Description != "Test command that plays airhorn DCA file" {
		t.Errorf("Expected description 'Test command that plays airhorn DCA file', got '%s'", definition.Description)
	}

	if definition.Type != discordgo.ChatApplicationCommand {
		t.Errorf("Expected command type ChatApplicationCommand, got %v", definition.Type)
	}
}

func TestTestCommandHandler_Handle_NilSession(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	handler := NewTestCommandHandler(logger)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "test-interaction-id",
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{},
		},
	}

	// Add mock member and user
	interaction.Member = &discordgo.Member{
		User: &discordgo.User{
			Username: "testuser",
		},
	}

	// Test with nil session - this should panic as expected
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil session
			t.Log("Handler correctly panics with nil session - this is expected behavior")
		} else {
			t.Error("Expected panic when session is nil")
		}
	}()

	handler.Handle(nil, interaction)
}

func TestTestCommandHandler_ImplementsInterface(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	handler := NewTestCommandHandler(logger)

	// Verify that TestCommandHandler implements CommandHandler interface
	var _ CommandHandler = handler

	// Test that both required methods exist
	definition := handler.Definition()
	if definition == nil {
		t.Error("Definition method should return a valid command definition")
	}

	// Test Handle method signature (we can't easily test the implementation without mocking)
	// But we can verify it exists and has the right signature by calling it with nil
	// This will panic/error but proves the method exists with correct signature
	defer func() {
		if r := recover(); r != nil {
			// Expected panic/error due to nil session
			t.Log("Handle method exists with correct signature")
		}
	}()

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:   "test-interaction-id",
			Type: discordgo.InteractionApplicationCommand,
		},
	}
	interaction.Member = &discordgo.Member{
		User: &discordgo.User{Username: "testuser"},
	}

	handler.Handle(nil, interaction)
}

func TestTestCommandHandler_Definition_Properties(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	handler := NewTestCommandHandler(logger)

	definition := handler.Definition()

	// Test all required properties for the /test command
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Name", definition.Name, "test"},
		{"Description", definition.Description, "Test command that plays airhorn DCA file"},
		{"Type", definition.Type, discordgo.ChatApplicationCommand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Definition.%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// Integration test that verifies the handler can be registered with CommandRouter
func TestTestCommandHandler_Integration_WithRouter(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)

	// Create router and handler
	router := NewCommandRouter(logger)
	handler := NewTestCommandHandler(logger)

	// Register handler
	err := router.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("Failed to register test command handler: %v", err)
	}

	// Verify handler is registered
	if router.GetHandlerCount() != 1 {
		t.Errorf("Expected 1 registered handler, got %d", router.GetHandlerCount())
	}

	// Verify command definition is available
	commands := router.GetRegisteredCommands()
	if len(commands) != 1 {
		t.Fatalf("Expected 1 registered command, got %d", len(commands))
	}

	if commands[0].Name != "test" {
		t.Errorf("Expected registered command name 'test', got '%s'", commands[0].Name)
	}
}
