package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"darrot/internal/bot"
	"darrot/internal/config"

	"github.com/joho/godotenv"
)

// Version information - set during build with -ldflags
var (
	version = "dev"     // Set via -ldflags "-X main.version=x.y.z"
	commit  = "unknown" // Set via -ldflags "-X main.commit=abc123"
	date    = "unknown" // Set via -ldflags "-X main.date=2024-01-01"
)

func main() {
	// Handle version flag
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Printf("darrot Discord TTS Bot\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit:  %s\n", commit)
		fmt.Printf("Date:    %s\n", date)
		return
	}

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
