package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"darrot/internal/config"

	"github.com/bwmarrin/discordgo"
)

// Bot represents the Discord bot instance with session management and command routing
type Bot struct {
	session       *discordgo.Session
	config        *config.Config
	logger        *log.Logger
	commandRouter *CommandRouter
	isRunning     bool
}

// New creates a new Bot instance with the provided configuration
func New(cfg *config.Config) (*Bot, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[BOT] ", log.LstdFlags|log.Lshortfile)

	// Create command router
	commandRouter := NewCommandRouter(logger)

	bot := &Bot{
		session:       session,
		config:        cfg,
		logger:        logger,
		commandRouter: commandRouter,
		isRunning:     false,
	}

	// Register the test command handler
	testHandler := NewTestCommandHandler(logger)
	if err := commandRouter.RegisterHandler(testHandler); err != nil {
		return nil, fmt.Errorf("failed to register test command handler: %w", err)
	}

	// Set up event handlers
	bot.setupEventHandlers()

	return bot, nil
}

// Start connects the bot to Discord and registers slash commands
func (b *Bot) Start() error {
	if b.isRunning {
		return fmt.Errorf("bot is already running")
	}

	b.logger.Println("Starting Discord bot...")

	// Open Discord connection
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	b.logger.Println("Discord connection established")

	// Register slash commands
	if err := b.registerCommands(); err != nil {
		b.logger.Printf("Warning: Failed to register commands: %v", err)
		// Continue running even if command registration fails
	}

	b.isRunning = true
	b.logger.Println("Bot started successfully")

	return nil
}

// Stop gracefully shuts down the bot
func (b *Bot) Stop() error {
	if !b.isRunning {
		return fmt.Errorf("bot is not running")
	}

	b.logger.Println("Stopping Discord bot...")

	// Close Discord connection
	if err := b.session.Close(); err != nil {
		b.logger.Printf("Error closing Discord connection: %v", err)
		return fmt.Errorf("failed to close Discord connection: %w", err)
	}

	b.isRunning = false
	b.logger.Println("Bot stopped successfully")

	return nil
}

// registerCommands registers all slash commands with Discord
func (b *Bot) registerCommands() error {
	b.logger.Println("Registering slash commands...")

	commands := b.commandRouter.GetRegisteredCommands()
	if len(commands) == 0 {
		b.logger.Println("No commands to register")
		return nil
	}

	// Check if session state is available (required for Discord API calls)
	if b.session.State == nil || b.session.State.User == nil {
		return fmt.Errorf("Discord session not properly initialized - cannot register commands")
	}

	for _, command := range commands {
		b.logger.Printf("Registering command: %s", command.Name)

		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", command)
		if err != nil {
			return fmt.Errorf("failed to register command '%s': %w", command.Name, err)
		}

		b.logger.Printf("Successfully registered command: %s", command.Name)
	}

	b.logger.Printf("Successfully registered %d slash commands", len(commands))
	return nil
}

// setupEventHandlers configures Discord event handlers
func (b *Bot) setupEventHandlers() {
	// Handle ready event
	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		b.logger.Printf("Bot is ready! Logged in as: %s#%s", r.User.Username, r.User.Discriminator)
	})

	// Handle interaction events (slash commands)
	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		b.handleInteraction(s, i)
	})

	// Handle disconnect events
	b.session.AddHandler(func(s *discordgo.Session, d *discordgo.Disconnect) {
		b.logger.Println("Discord connection lost")
	})

	// Handle resume events (connection restored)
	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Resumed) {
		b.logger.Println("Discord connection restored")
	})
}

// handleInteraction processes incoming Discord interactions
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Only handle application command interactions
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	// Route command to appropriate handler
	if err := b.commandRouter.RouteCommand(s, i); err != nil {
		b.logger.Printf("Error handling interaction: %v", err)

		// Send error response to user
		b.sendErrorResponse(s, i, "Sorry, something went wrong processing your command.")
	}
}

// sendErrorResponse sends a user-friendly error message
func (b *Bot) sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		b.logger.Printf("Failed to send error response: %v", err)
	}
}

// IsRunning returns whether the bot is currently running
func (b *Bot) IsRunning() bool {
	return b.isRunning
}

// WaitForShutdown blocks until a shutdown signal is received
func (b *Bot) WaitForShutdown() {
	// Create channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	b.logger.Println("Bot is running. Press Ctrl+C to stop.")

	// Wait for signal
	<-stop

	b.logger.Println("Shutdown signal received")
}
