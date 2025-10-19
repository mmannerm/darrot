package main

import (
	"encoding/json"
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

// configShowCmd represents the config show subcommand
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display effective configuration with source information",
	Long: `Display the effective configuration with source information.

This command shows the current configuration values that would be used
by the bot, along with information about where each value comes from
(CLI flags, environment variables, config file, or defaults).

Sensitive values like tokens are masked for security.

Output formats:
  human-readable (default) - Formatted output with source information
  json                     - JSON format for programmatic use`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get format flag
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return fmt.Errorf("failed to get format flag: %w", err)
		}

		// Validate format
		if format != "human" && format != "json" {
			return fmt.Errorf("invalid format '%s': must be 'human' or 'json'", format)
		}

		// Create a new config manager
		cm := config.NewConfigManager()

		// Bind CLI flags to the config manager's Viper instance
		if err := bindFlagsToConfigManager(cm, cmd); err != nil {
			return fmt.Errorf("failed to bind flags: %w", err)
		}

		// If a config file was specified via global flag, set it on the config manager
		if cfgFile != "" {
			cm.SetConfigFile(cfgFile)
		}

		// Load configuration with source information
		configWithSources, err := cm.LoadConfigWithSources()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Display configuration based on format
		if format == "json" {
			return displayConfigJSON(configWithSources)
		}

		return displayConfigHuman(configWithSources)
	},
}

// configCreateCmd represents the config create subcommand
var configCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Save current effective configuration to file",
	Long: `Save the current effective configuration to a YAML file.

This command loads the current configuration from all sources (CLI flags,
environment variables, config files, and defaults) and saves it to a
properly formatted YAML configuration file.

The saved configuration file will include all non-sensitive configuration
values. Sensitive values like Discord tokens are excluded for security
and should be provided via environment variables.

Default output location: ./darrot-config.yaml
Custom output location: Use --output flag to specify a different path

Example usage:
  darrot config create                           # Save to ./darrot-config.yaml
  darrot config create --output ~/my-config.yaml # Save to custom location`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get output flag
		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return fmt.Errorf("failed to get output flag: %w", err)
		}

		// Create a new config manager
		cm := config.NewConfigManager()

		// Bind CLI flags to the config manager's Viper instance
		if err := bindFlagsToConfigManager(cm, cmd); err != nil {
			return fmt.Errorf("failed to bind flags: %w", err)
		}

		// If a config file was specified via global flag, set it on the config manager
		if cfgFile != "" {
			cm.SetConfigFile(cfgFile)
		}

		// If no output path specified, use default
		if outputPath == "" {
			outputPath = cm.GetDefaultConfigPath()
		}

		// Save configuration to file
		if err := cm.SaveConfigToFile(outputPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		// Print success message
		fmt.Printf("✓ Configuration saved to: %s\n", outputPath)
		fmt.Println()
		fmt.Println("Note: Sensitive values like Discord tokens are excluded from the config file.")
		fmt.Println("Set sensitive configuration via environment variables:")
		fmt.Println("  DRT_DISCORD_TOKEN=your-bot-token")
		fmt.Println("  DRT_TTS_GOOGLE_CLOUD_CREDENTIALS_PATH=/path/to/credentials.json")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configCreateCmd)

	// Add the same flags as the start command for validation
	addConfigFlags(configValidateCmd)
	addConfigFlags(configShowCmd)
	addConfigFlags(configCreateCmd)

	// Add format flag to show command
	configShowCmd.Flags().String("format", "human", "Output format (human, json)")

	// Add output flag to create command
	configCreateCmd.Flags().String("output", "", "Output file path (default: ./darrot-config.yaml)")

	// Set up custom completion functions for config commands
	setupConfigCompletions()
}

// setupConfigCompletions configures custom completion functions for config command flags
func setupConfigCompletions() {
	// Custom completion for config show format flag
	_ = configShowCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"human", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Custom completion for config create output flag
	_ = configCreateCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
	})

	// Add completions to all config subcommands
	for _, cmd := range []*cobra.Command{configValidateCmd, configShowCmd, configCreateCmd} {
		_ = cmd.RegisterFlagCompletionFunc("tts-default-voice", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			voices := []string{
				"en-US-Standard-A", "en-US-Standard-B", "en-US-Standard-C", "en-US-Standard-D",
				"en-US-Wavenet-A", "en-US-Wavenet-B", "en-US-Wavenet-C", "en-US-Wavenet-D",
				"en-US-Neural2-A", "en-US-Neural2-C", "en-US-Neural2-D", "en-US-Neural2-E",
			}
			return voices, cobra.ShellCompDirectiveNoFileComp
		})

		_ = cmd.RegisterFlagCompletionFunc("google-cloud-credentials-path", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"json"}, cobra.ShellCompDirectiveFilterFileExt
		})
	}
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

// displayConfigHuman displays configuration in human-readable format with source information
func displayConfigHuman(configWithSources *config.ConfigWithSources) error {
	cfg := configWithSources.Config
	sources := configWithSources.Sources

	fmt.Println("Effective Configuration:")
	fmt.Println("========================")
	fmt.Println()

	// Core configuration
	fmt.Println("Core Configuration:")
	fmt.Printf("  Discord Token: %s", maskSensitiveValue(cfg.DiscordToken))
	if source, ok := sources["discord_token"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()

	fmt.Printf("  Log Level: %s", cfg.LogLevel)
	if source, ok := sources["log_level"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()
	fmt.Println()

	// TTS configuration
	fmt.Println("TTS Configuration:")
	if cfg.TTS.GoogleCloudCredentialsPath != "" {
		fmt.Printf("  Google Cloud Credentials: %s", maskSensitiveValue(cfg.TTS.GoogleCloudCredentialsPath))
		if source, ok := sources["tts.google_cloud_credentials_path"]; ok {
			fmt.Printf(" (source: %s)", source.Source)
		}
		fmt.Println()
	}

	fmt.Printf("  Default Voice: %s", cfg.TTS.DefaultVoice)
	if source, ok := sources["tts.default_voice"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()

	fmt.Printf("  Default Speed: %.2f", cfg.TTS.DefaultSpeed)
	if source, ok := sources["tts.default_speed"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()

	fmt.Printf("  Default Volume: %.2f", cfg.TTS.DefaultVolume)
	if source, ok := sources["tts.default_volume"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()

	fmt.Printf("  Max Queue Size: %d", cfg.TTS.MaxQueueSize)
	if source, ok := sources["tts.max_queue_size"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()

	fmt.Printf("  Max Message Length: %d", cfg.TTS.MaxMessageLength)
	if source, ok := sources["tts.max_message_length"]; ok {
		fmt.Printf(" (source: %s)", source.Source)
	}
	fmt.Println()
	fmt.Println()

	// Configuration precedence information
	fmt.Println("Configuration Precedence (highest to lowest):")
	fmt.Println("  1. CLI flags (--flag-name)")
	fmt.Println("  2. Environment variables (DRT_*)")
	fmt.Println("  3. Configuration file")
	fmt.Println("  4. Default values")

	return nil
}

// displayConfigJSON displays configuration in JSON format
func displayConfigJSON(configWithSources *config.ConfigWithSources) error {
	cfg := configWithSources.Config
	sources := configWithSources.Sources

	// Create a structure for JSON output that includes masked sensitive values
	output := map[string]interface{}{
		"config": map[string]interface{}{
			"discord_token": maskSensitiveValue(cfg.DiscordToken),
			"log_level":     cfg.LogLevel,
			"tts": map[string]interface{}{
				"google_cloud_credentials_path": maskSensitiveValue(cfg.TTS.GoogleCloudCredentialsPath),
				"default_voice":                 cfg.TTS.DefaultVoice,
				"default_speed":                 cfg.TTS.DefaultSpeed,
				"default_volume":                cfg.TTS.DefaultVolume,
				"max_queue_size":                cfg.TTS.MaxQueueSize,
				"max_message_length":            cfg.TTS.MaxMessageLength,
			},
		},
		"sources": sources,
		"precedence": []string{
			"CLI flags (--flag-name)",
			"Environment variables (DRT_*)",
			"Configuration file",
			"Default values",
		},
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}
