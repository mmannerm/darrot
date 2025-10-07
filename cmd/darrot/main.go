package main

import (
	"log"
	"os"

	"darrot/internal/bot"
	"darrot/internal/config"

	"github.com/joho/godotenv"
)

func main() {
	// Set up basic logging for application lifecycle
	logger := log.New(os.Stdout, "[DARROT] ", log.LstdFlags|log.Lshortfile)

	logger.Println("Starting darrot Discord TTS bot...")

	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env file doesn't exist, just log it
		logger.Printf("No .env file found or error loading it: %v", err)
	}

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Println("Configuration loaded successfully")

	// Initialize bot instance
	botInstance, err := bot.New(cfg)
	if err != nil {
		logger.Fatalf("Failed to initialize bot: %v", err)
	}

	logger.Println("Bot instance created successfully")

	// Start the bot
	if err := botInstance.Start(); err != nil {
		logger.Fatalf("Failed to start bot: %v", err)
	}

	// Wait for shutdown signal (Ctrl+C or SIGTERM)
	botInstance.WaitForShutdown()

	// Graceful shutdown
	logger.Println("Shutting down bot...")
	if err := botInstance.Stop(); err != nil {
		logger.Printf("Error during shutdown: %v", err)
		os.Exit(1)
	}

	logger.Println("Bot shutdown complete")
}
