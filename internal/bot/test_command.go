package bot

import (
	"log"
	"os"

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

// Handle processes the /test command interaction and plays the airhorn DCA file
func (h *TestCommandHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get username and guild info
	var username string
	var guildID string

	if i.Member != nil && i.Member.User != nil {
		username = i.Member.User.Username
	} else if i.User != nil {
		username = i.User.Username
	} else {
		username = "unknown"
	}

	if i.GuildID != "" {
		guildID = i.GuildID
	}

	h.logger.Printf("Processing /test command from user: %s in guild: %s", username, guildID)

	// First respond to the interaction
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Testing airhorn DCA file...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	err := s.InteractionRespond(i.Interaction, response)
	if err != nil {
		h.logger.Printf("Error responding to /test command: %v", err)
		return err
	}

	// Try to play the airhorn DCA file if we're in a guild
	if guildID != "" {
		go h.playAirhornDCA(s, guildID)
	}

	h.logger.Printf("Successfully responded to /test command")
	return nil
}

// playAirhornDCA plays the test airhorn DCA file
func (h *TestCommandHandler) playAirhornDCA(s *discordgo.Session, guildID string) {
	// Read the DCA file
	dcaData, err := os.ReadFile("test_airhorn.dca")
	if err != nil {
		h.logger.Printf("Failed to read airhorn DCA file: %v", err)
		return
	}

	h.logger.Printf("Loaded airhorn DCA file: %d bytes", len(dcaData))

	// Find a voice connection for this guild
	for _, vs := range s.VoiceConnections {
		if vs.GuildID == guildID {
			h.logger.Printf("Found voice connection for guild %s, playing airhorn", guildID)

			// Set speaking state
			if err := vs.Speaking(true); err != nil {
				h.logger.Printf("Failed to set speaking state: %v", err)
				return
			}
			defer func() {
				if err := vs.Speaking(false); err != nil {
					h.logger.Printf("Failed to unset speaking state: %v", err)
				}
			}()

			// Parse DCA frames and send them
			frames, err := h.parseDCAFrames(dcaData)
			if err != nil {
				h.logger.Printf("Failed to parse DCA frames: %v", err)
				return
			}

			h.logger.Printf("Parsed %d DCA frames, sending to Discord", len(frames))

			// Send each frame
			for _, frame := range frames {
				vs.OpusSend <- frame
			}

			h.logger.Printf("Finished playing airhorn")
			return
		}
	}

	h.logger.Printf("No voice connection found for guild %s", guildID)
}

// parseDCAFrames parses DCA format data into individual Opus frames
func (h *TestCommandHandler) parseDCAFrames(dcaData []byte) ([][]byte, error) {
	var frames [][]byte
	offset := 0

	for offset < len(dcaData) {
		// Need at least 2 bytes for frame length header
		if offset+2 > len(dcaData) {
			break
		}

		// Read frame length (2 bytes, little-endian)
		frameLen := int(dcaData[offset]) | int(dcaData[offset+1])<<8
		offset += 2

		// Validate frame length
		if frameLen <= 0 || frameLen > 4000 {
			h.logger.Printf("Invalid DCA frame length %d at offset %d", frameLen, offset-2)
			break
		}

		// Check if we have enough data for the frame
		if offset+frameLen > len(dcaData) {
			h.logger.Printf("Incomplete DCA frame: expected %d bytes, only %d available", frameLen, len(dcaData)-offset)
			break
		}

		// Extract the Opus frame data
		frame := make([]byte, frameLen)
		copy(frame, dcaData[offset:offset+frameLen])
		frames = append(frames, frame)

		offset += frameLen
	}

	h.logger.Printf("Successfully parsed %d DCA frames from %d bytes", len(frames), len(dcaData))
	return frames, nil
}

// Definition returns the Discord slash command definition for the /test command
func (h *TestCommandHandler) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "test",
		Description: "Test command that plays airhorn DCA file",
		Type:        discordgo.ChatApplicationCommand,
	}
}
