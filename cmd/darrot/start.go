package main

import (
	"log"
	"os"

	"darrot/internal/bot"
	"darrot/internal/config"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Discord TTS bot",
	Long: `Start the Discord TTS bot with the current configuration.

The bot will connect to Discord, join voice channels, and begin processing
text messages for text-to-speech conversion. All current configuration
options are available as command-line flags.

The bot will run until interrupted with Ctrl+C or a termination signal.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set up basic logging for application lifecycle
		logger := log.New(os.Stdout, "[DARROT] ", log.LstdFlags|log.Lshortfile)

		logger.Printf("Starting darrot Discord TTS bot v%s...", version)

		// Bind CLI flags to Viper
		if err := bindStartFlags(cmd); err != nil {
			logger.Fatalf("Failed to bind flags: %v", err)
			return err
		}

		// Load environment variables from .env file if it exists
		if err := godotenv.Load(); err != nil {
			// Don't fail if .env file doesn't exist, just log it
			logger.Printf("No .env file found or error loading it: %v", err)
		}

		// Load configuration from environment variables and CLI flags
		cfg, err := config.Load()
		if err != nil {
			logger.Fatalf("Failed to load configuration: %v", err)
			return err
		}

		logger.Println("Configuration loaded successfully")

		// Initialize bot instance
		botInstance, err := bot.New(cfg)
		if err != nil {
			logger.Fatalf("Failed to initialize bot: %v", err)
			return err
		}

		logger.Println("Bot instance created successfully")

		// Start the bot
		if err := botInstance.Start(); err != nil {
			logger.Fatalf("Failed to start bot: %v", err)
			return err
		}

		// Wait for shutdown signal (Ctrl+C or SIGTERM)
		botInstance.WaitForShutdown()

		// Graceful shutdown
		logger.Println("Shutting down bot...")
		if err := botInstance.Stop(); err != nil {
			logger.Printf("Error during shutdown: %v", err)
			return err
		}

		logger.Println("Bot shutdown complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Discord configuration flags
	startCmd.Flags().String("discord-token", "", "Discord bot token (required)")

	// TTS configuration flags
	startCmd.Flags().String("google-cloud-credentials-path", "", "Path to Google Cloud credentials JSON file")
	startCmd.Flags().String("tts-default-voice", "en-US-Standard-A", "Default TTS voice")
	startCmd.Flags().Float32("tts-default-speed", 1.0, "Default TTS speed (0.25-4.0)")
	startCmd.Flags().Float32("tts-default-volume", 1.0, "Default TTS volume (0.0-2.0)")
	startCmd.Flags().Int("tts-max-queue-size", 10, "Maximum TTS queue size (1-100)")
	startCmd.Flags().Int("tts-max-message-length", 500, "Maximum message length for TTS (1-2000)")
}

// bindStartFlags binds CLI flags to Viper configuration keys
func bindStartFlags(cmd *cobra.Command) error {
	// Bind Discord configuration
	if err := viper.BindPFlag("discord_token", cmd.Flags().Lookup("discord-token")); err != nil {
		return err
	}

	// Bind TTS configuration
	if err := viper.BindPFlag("google_cloud_credentials_path", cmd.Flags().Lookup("google-cloud-credentials-path")); err != nil {
		return err
	}
	if err := viper.BindPFlag("tts_default_voice", cmd.Flags().Lookup("tts-default-voice")); err != nil {
		return err
	}
	if err := viper.BindPFlag("tts_default_speed", cmd.Flags().Lookup("tts-default-speed")); err != nil {
		return err
	}
	if err := viper.BindPFlag("tts_default_volume", cmd.Flags().Lookup("tts-default-volume")); err != nil {
		return err
	}
	if err := viper.BindPFlag("tts_max_queue_size", cmd.Flags().Lookup("tts-max-queue-size")); err != nil {
		return err
	}
	if err := viper.BindPFlag("tts_max_message_length", cmd.Flags().Lookup("tts-max-message-length")); err != nil {
		return err
	}

	return nil
}
