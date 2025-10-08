package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// CommandHandler defines the interface that all command handlers must implement
type CommandHandler interface {
	// Handle processes the Discord interaction and returns an error if processing fails
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
	// Definition returns the Discord slash command definition for registration
	Definition() *discordgo.ApplicationCommand
}

// CommandRouter manages command handler registration and routing
type CommandRouter struct {
	handlers map[string]CommandHandler
	logger   *log.Logger
}

// NewCommandRouter creates a new CommandRouter instance
func NewCommandRouter(logger *log.Logger) *CommandRouter {
	return &CommandRouter{
		handlers: make(map[string]CommandHandler),
		logger:   logger,
	}
}

// RegisterHandler registers a command handler with the router
func (r *CommandRouter) RegisterHandler(handler CommandHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	definition := handler.Definition()
	if definition == nil {
		return fmt.Errorf("handler definition cannot be nil")
	}

	if definition.Name == "" {
		return fmt.Errorf("handler definition name cannot be empty")
	}

	// Check if handler is already registered
	if _, exists := r.handlers[definition.Name]; exists {
		return fmt.Errorf("handler for command '%s' is already registered", definition.Name)
	}

	r.handlers[definition.Name] = handler
	r.logger.Printf("Registered command handler: %s", definition.Name)
	return nil
}

// RouteCommand routes an interaction to the appropriate command handler
func (r *CommandRouter) RouteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.ApplicationCommandData().Name == "" {
		return fmt.Errorf("interaction command name is empty")
	}

	commandName := i.ApplicationCommandData().Name
	handler, exists := r.handlers[commandName]
	if !exists {
		return fmt.Errorf("no handler registered for command: %s", commandName)
	}

	r.logger.Printf("Routing command '%s' to handler", commandName)
	return handler.Handle(s, i)
}

// GetRegisteredCommands returns all registered command definitions
func (r *CommandRouter) GetRegisteredCommands() []*discordgo.ApplicationCommand {
	commands := make([]*discordgo.ApplicationCommand, 0, len(r.handlers))
	for _, handler := range r.handlers {
		commands = append(commands, handler.Definition())
	}
	return commands
}

// GetHandlerCount returns the number of registered handlers
func (r *CommandRouter) GetHandlerCount() int {
	return len(r.handlers)
}
