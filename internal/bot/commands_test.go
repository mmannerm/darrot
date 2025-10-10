package bot

import (
	"log"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// MockCommandHandler implements CommandHandler for testing
type MockCommandHandler struct {
	name        string
	description string
	handleFunc  func(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

func (m *MockCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if m.handleFunc != nil {
		return m.handleFunc(s, i)
	}
	return nil
}

func (m *MockCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        m.name,
		Description: m.description,
		Type:        discordgo.ChatApplicationCommand,
	}
}

func TestNewCommandRouter(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	if router == nil {
		t.Fatal("NewCommandRouter returned nil")
	}

	if router.handlers == nil {
		t.Error("handlers map not initialized")
	}

	if router.logger != logger {
		t.Error("logger not set correctly")
	}

	if router.GetHandlerCount() != 0 {
		t.Errorf("expected 0 handlers, got %d", router.GetHandlerCount())
	}
}

func TestCommandRouter_RegisterHandler(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	tests := []struct {
		name        string
		handler     CommandHandler
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid handler",
			handler:     &MockCommandHandler{name: "test", description: "Test command"},
			expectError: false,
		},
		{
			name:        "nil handler",
			handler:     nil,
			expectError: true,
			errorMsg:    "handler cannot be nil",
		},
		{
			name:        "handler with empty name",
			handler:     &MockCommandHandler{name: "", description: "Test command"},
			expectError: true,
			errorMsg:    "handler definition name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.RegisterHandler(tt.handler)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCommandRouter_RegisterHandler_Duplicate(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	handler1 := &MockCommandHandler{name: "test", description: "First test command"}
	handler2 := &MockCommandHandler{name: "test", description: "Second test command"}

	// Register first handler
	err := router.RegisterHandler(handler1)
	if err != nil {
		t.Fatalf("failed to register first handler: %v", err)
	}

	// Try to register duplicate handler
	err = router.RegisterHandler(handler2)
	if err == nil {
		t.Error("expected error when registering duplicate handler")
	}

	expectedError := "handler for command 'test' is already registered"
	if err.Error() != expectedError {
		t.Errorf("expected error '%s', got '%s'", expectedError, err.Error())
	}

	// Verify only one handler is registered
	if router.GetHandlerCount() != 1 {
		t.Errorf("expected 1 handler, got %d", router.GetHandlerCount())
	}
}

func TestCommandRouter_RouteCommand(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	// Create mock handler
	handlerCalled := false
	handler := &MockCommandHandler{
		name:        "test",
		description: "Test command",
		handleFunc: func(s *discordgo.Session, i *discordgo.InteractionCreate) error {
			handlerCalled = true
			return nil
		},
	}

	// Register handler
	err := router.RegisterHandler(handler)
	if err != nil {
		t.Fatalf("failed to register handler: %v", err)
	}

	// Create mock interaction
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				ID:   "123",
				Name: "test",
			},
		},
	}

	// Route command
	err = router.RouteCommand(nil, interaction)
	if err != nil {
		t.Errorf("unexpected error routing command: %v", err)
	}

	if !handlerCalled {
		t.Error("handler was not called")
	}
}

func TestCommandRouter_RouteCommand_UnknownCommand(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	// Create interaction for unknown command
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				ID:   "123",
				Name: "unknown",
			},
		},
	}

	// Try to route unknown command
	err := router.RouteCommand(nil, interaction)
	if err == nil {
		t.Error("expected error for unknown command")
	}

	expectedError := "no handler registered for command: unknown"
	if err.Error() != expectedError {
		t.Errorf("expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestCommandRouter_RouteCommand_EmptyCommandName(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	// Create interaction with empty command name
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				ID:   "123",
				Name: "",
			},
		},
	}

	// Try to route command with empty name
	err := router.RouteCommand(nil, interaction)
	if err == nil {
		t.Error("expected error for empty command name")
	}

	expectedError := "interaction command name is empty"
	if err.Error() != expectedError {
		t.Errorf("expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestCommandRouter_GetRegisteredCommands(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	// Initially should have no commands
	commands := router.GetRegisteredCommands()
	if len(commands) != 0 {
		t.Errorf("expected 0 commands, got %d", len(commands))
	}

	// Register handlers
	handler1 := &MockCommandHandler{name: "test1", description: "First test command"}
	handler2 := &MockCommandHandler{name: "test2", description: "Second test command"}

	router.RegisterHandler(handler1)
	router.RegisterHandler(handler2)

	// Get registered commands
	commands = router.GetRegisteredCommands()
	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}

	// Verify command names are present
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name] = true
	}

	if !commandNames["test1"] {
		t.Error("test1 command not found in registered commands")
	}

	if !commandNames["test2"] {
		t.Error("test2 command not found in registered commands")
	}
}

func TestCommandRouter_GetHandlerCount(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	router := NewCommandRouter(logger)

	// Initially should have 0 handlers
	if router.GetHandlerCount() != 0 {
		t.Errorf("expected 0 handlers, got %d", router.GetHandlerCount())
	}

	// Register one handler
	handler1 := &MockCommandHandler{name: "test1", description: "First test command"}
	router.RegisterHandler(handler1)

	if router.GetHandlerCount() != 1 {
		t.Errorf("expected 1 handler, got %d", router.GetHandlerCount())
	}

	// Register another handler
	handler2 := &MockCommandHandler{name: "test2", description: "Second test command"}
	router.RegisterHandler(handler2)

	if router.GetHandlerCount() != 2 {
		t.Errorf("expected 2 handlers, got %d", router.GetHandlerCount())
	}
}
