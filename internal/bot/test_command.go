package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// TestCommandHandler implements the CommandHandler interface for the /test command
type TestCommandHandler struct {
	logger *log.Logger
}

// NewTestCommandHandler creates a new TestCommandHandler instance
func NewTestCommandHandler(logger *log.Logger) *TestCommandHandler {
	return &TestCommandHandler{
		logger: logger,
	}
}

// Handle processes the /test command interaction and responds with "Hello World"
func (h *TestCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get username from either Member (guild) or User (DM) context
	var username string
	if i.Member != nil && i.Member.User != nil {
		username = i.Member.User.Username
	} else if i.User != nil {
		username = i.User.Username
	} else {
		username = "unknown"
	}

	h.logger.Printf("Processing /test command from user: %s", username)

	// Create ephemeral response with "Hello World" message
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Hello World",
			Flags:   discordgo.MessageFlagsEphemeral, // Makes response visible only to command user
		},
	}

	err := s.InteractionRespond(i.Interaction, response)
	if err != nil {
		h.logger.Printf("Error responding to /test command: %v", err)
		return err
	}

	h.logger.Printf("Successfully responded to /test command")
	return nil
}

// Definition returns the Discord slash command definition for the /test command
func (h *TestCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "test",
		Description: "Test command that responds with Hello World",
		Type:        discordgo.ChatApplicationCommand,
	}
}
