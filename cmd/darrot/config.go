package main

import (
	"fmt"
	"os"

	"darrot/internal/config"

	"github.com/spf13/cobra"
)

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long: `Configuration management utilities for the darrot Discord TTS bot.

Provides subcommands to validate, inspect, and create configuration files
without starting the bot. Useful for troubleshooting configuration issues
and verifying settings before deployment.`,
}

// configValidateCmd represents the config validate subcommand
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration without starting the bot",
	Long: `Validate all configuration sources and report any errors.

This command loads configuration from all sources (CLI flags, environment
variables, config files, and defaults) and validates that all required
values are present and valid. It does not start the bot.

Exit codes:
  0 - Configuration is valid
  1 - Configuration validation failed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a new config manager
		cm := config.NewConfigManager()

		// Bind CLI flags to the config manager's Viper instance
		if err := bindFlagsToConfigManager(cm, cmd); err != nil {
			return fmt.Errorf("failed to bind flags: %w", err)
		}

		// If a config file was specified via global flag, set it on the config manager
		// Note: The root command's initConfig has already processed the global --config flag
		// and set up the global viper instance, but we need to use our ConfigManager's instance
		if cfgFile != "" {
			cm.SetConfigFile(cfgFile)
		}

		// Attempt to load and validate configuration
		cfg, err := cm.LoadConfig()
		if err != nil {
			// Print detailed error message
			fmt.Fprintf(os.Stderr, "Configuration validation failed:\n")
			fmt.Fprintf(os.Stderr, "  %v\n\n", err)

			// Provide helpful suggestions based on error type
			printValidationSuggestions(err)

			return fmt.Errorf("configuration validation failed")
		}

		// Configuration is valid - print success message
		fmt.Println("✓ Configuration validation successful")
		fmt.Printf("  Discord token: %s\n", maskSensitiveValue(cfg.DiscordToken))
		fmt.Printf("  Log level: %s\n", cfg.LogLevel)
		fmt.Printf("  TTS voice: %s\n", cfg.TTS.DefaultVoice)
		fmt.Printf("  TTS speed: %.2f\n", cfg.TTS.DefaultSpeed)
		fmt.Printf("  TTS volume: %.2f\n", cfg.TTS.DefaultVolume)
		fmt.Printf("  Max queue size: %d\n", cfg.TTS.MaxQueueSize)
		fmt.Printf("  Max message length: %d\n", cfg.TTS.MaxMessageLength)

		if cfg.TTS.GoogleCloudCredentialsPath != "" {
			fmt.Printf("  Google Cloud credentials: %s\n", maskSensitiveValue(cfg.TTS.GoogleCloudCredentialsPath))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)

	// Add the same flags as the start command for validation
	addConfigFlags(configValidateCmd)
}

// addConfigFlags adds configuration flags to a command
func addConfigFlags(cmd *cobra.Command) {
	// Discord configuration flags
	cmd.Flags().String("discord-token", "", "Discord bot token (required)")

	// TTS configuration flags
	cmd.Flags().String("google-cloud-credentials-path", "", "Path to Google Cloud credentials JSON file")
	cmd.Flags().String("tts-default-voice", "en-US-Standard-A", "Default TTS voice")
	cmd.Flags().Float32("tts-default-speed", 1.0, "Default TTS speed (0.25-4.0)")
	cmd.Flags().Float32("tts-default-volume", 1.0, "Default TTS volume (0.0-2.0)")
	cmd.Flags().Int("tts-max-queue-size", 10, "Maximum TTS queue size (1-100)")
	cmd.Flags().Int("tts-max-message-length", 500, "Maximum message length for TTS (1-2000)")
}

// bindFlagsToConfigManager binds CLI flags to the ConfigManager's Viper instance
func bindFlagsToConfigManager(cm *config.ConfigManager, cmd *cobra.Command) error {
	v := cm.GetViper()

	// Bind Discord configuration
	if err := v.BindPFlag("discord_token", cmd.Flags().Lookup("discord-token")); err != nil {
		return err
	}

	// Bind TTS configuration
	if err := v.BindPFlag("tts.google_cloud_credentials_path", cmd.Flags().Lookup("google-cloud-credentials-path")); err != nil {
		return err
	}
	if err := v.BindPFlag("tts.default_voice", cmd.Flags().Lookup("tts-default-voice")); err != nil {
		return err
	}
	if err := v.BindPFlag("tts.default_speed", cmd.Flags().Lookup("tts-default-speed")); err != nil {
		return err
	}
	if err := v.BindPFlag("tts.default_volume", cmd.Flags().Lookup("tts-default-volume")); err != nil {
		return err
	}
	if err := v.BindPFlag("tts.max_queue_size", cmd.Flags().Lookup("tts-max-queue-size")); err != nil {
		return err
	}
	if err := v.BindPFlag("tts.max_message_length", cmd.Flags().Lookup("tts-max-message-length")); err != nil {
		return err
	}

	return nil
}

// maskSensitiveValue masks sensitive configuration values for display
func maskSensitiveValue(value string) string {
	if value == "" {
		return "<not set>"
	}

	if len(value) <= 8 {
		return "***"
	}

	// Show first 4 and last 4 characters with asterisks in between
	return value[:4] + "***" + value[len(value)-4:]
}

// printValidationSuggestions provides helpful suggestions based on validation errors
func printValidationSuggestions(err error) {
	errorMsg := err.Error()

	fmt.Fprintf(os.Stderr, "Suggestions:\n")

	// Discord token suggestions
	if contains(errorMsg, "discord_token") {
		fmt.Fprintf(os.Stderr, "  • Set Discord token via environment variable: DRT_DISCORD_TOKEN=your-token\n")
		fmt.Fprintf(os.Stderr, "  • Set Discord token via config file: discord_token: your-token\n")
		fmt.Fprintf(os.Stderr, "  • Set Discord token via CLI flag: --discord-token your-token\n")
	}

	// Log level suggestions
	if contains(errorMsg, "log_level") {
		fmt.Fprintf(os.Stderr, "  • Valid log levels: DEBUG, INFO, WARN, ERROR, FATAL\n")
		fmt.Fprintf(os.Stderr, "  • Set via environment variable: DRT_LOG_LEVEL=INFO\n")
		fmt.Fprintf(os.Stderr, "  • Set via config file: log_level: INFO\n")
		fmt.Fprintf(os.Stderr, "  • Set via CLI flag: --log-level INFO\n")
	}

	// TTS speed suggestions
	if contains(errorMsg, "default_speed") {
		fmt.Fprintf(os.Stderr, "  • TTS speed must be between 0.25 and 4.0\n")
		fmt.Fprintf(os.Stderr, "  • Set via environment variable: DRT_TTS_DEFAULT_SPEED=1.0\n")
		fmt.Fprintf(os.Stderr, "  • Set via config file: tts.default_speed: 1.0\n")
		fmt.Fprintf(os.Stderr, "  • Set via CLI flag: --tts-default-speed 1.0\n")
	}

	// TTS volume suggestions
	if contains(errorMsg, "default_volume") {
		fmt.Fprintf(os.Stderr, "  • TTS volume must be between 0.0 and 2.0\n")
		fmt.Fprintf(os.Stderr, "  • Set via environment variable: DRT_TTS_DEFAULT_VOLUME=1.0\n")
		fmt.Fprintf(os.Stderr, "  • Set via config file: tts.default_volume: 1.0\n")
		fmt.Fprintf(os.Stderr, "  • Set via CLI flag: --tts-default-volume 1.0\n")
	}

	// Queue size suggestions
	if contains(errorMsg, "max_queue_size") {
		fmt.Fprintf(os.Stderr, "  • Max queue size must be between 1 and 100\n")
		fmt.Fprintf(os.Stderr, "  • Set via environment variable: DRT_TTS_MAX_QUEUE_SIZE=10\n")
		fmt.Fprintf(os.Stderr, "  • Set via config file: tts.max_queue_size: 10\n")
		fmt.Fprintf(os.Stderr, "  • Set via CLI flag: --tts-max-queue-size 10\n")
	}

	// Message length suggestions
	if contains(errorMsg, "max_message_length") {
		fmt.Fprintf(os.Stderr, "  • Max message length must be between 1 and 2000\n")
		fmt.Fprintf(os.Stderr, "  • Set via environment variable: DRT_TTS_MAX_MESSAGE_LENGTH=500\n")
		fmt.Fprintf(os.Stderr, "  • Set via config file: tts.max_message_length: 500\n")
		fmt.Fprintf(os.Stderr, "  • Set via CLI flag: --tts-max-message-length 500\n")
	}

	fmt.Fprintf(os.Stderr, "\nConfiguration precedence (highest to lowest):\n")
	fmt.Fprintf(os.Stderr, "  1. CLI flags (--flag-name)\n")
	fmt.Fprintf(os.Stderr, "  2. Environment variables (DRT_*)\n")
	fmt.Fprintf(os.Stderr, "  3. Configuration file\n")
	fmt.Fprintf(os.Stderr, "  4. Default values\n")
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

// containsSubstring is a simple substring check helper
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
