package main

import (
	"log"
	"os"

	"darrot/internal/bot"
	"darrot/internal/config"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
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

		// Load environment variables from .env file if it exists
		if err := godotenv.Load(); err != nil {
			// Don't fail if .env file doesn't exist, just log it
			logger.Printf("No .env file found or error loading it: %v", err)
		}

		// Load configuration from environment variables
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
