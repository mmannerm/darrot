package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// setupErrorHandling configures comprehensive error handling for the CLI
func setupErrorHandling(rootCmd *cobra.Command) {
	// Set custom usage template with better formatting
	rootCmd.SetUsageTemplate(getCustomUsageTemplate())

	// Set custom help template with better formatting
	rootCmd.SetHelpTemplate(getCustomHelpTemplate())

	// Configure command suggestion settings
	rootCmd.SuggestionsMinimumDistance = 1
	rootCmd.SuggestFor = []string{"start", "version", "config", "help", "completion"}

	// Set custom error handling function
	rootCmd.SetFlagErrorFunc(handleFlagError)

	// Configure all subcommands with error handling
	configureSubcommandErrorHandling(rootCmd)
}

// handleFlagError provides helpful error messages for invalid flags
func handleFlagError(cmd *cobra.Command, err error) error {
	errorMsg := err.Error()

	// Extract flag name from error message
	var flagName string
	if strings.Contains(errorMsg, "unknown flag:") {
		parts := strings.Split(errorMsg, "unknown flag: ")
		if len(parts) > 1 {
			flagName = strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(errorMsg, "flag needs an argument:") {
		parts := strings.Split(errorMsg, "flag needs an argument: ")
		if len(parts) > 1 {
			flagName = strings.TrimSpace(parts[1])
		}
	}

	fmt.Fprintf(os.Stderr, "Error: %s\n\n", errorMsg)

	// Provide suggestions for unknown flags
	if flagName != "" && strings.Contains(errorMsg, "unknown flag:") {
		suggestions := suggestFlags(cmd, flagName)
		if len(suggestions) > 0 {
			fmt.Fprintf(os.Stderr, "Did you mean one of these?\n")
			for _, suggestion := range suggestions {
				fmt.Fprintf(os.Stderr, "  %s\n", suggestion)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	// Show available flags
	fmt.Fprintf(os.Stderr, "Available flags for '%s':\n", cmd.Name())
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden {
			usage := flag.Usage
			if flag.DefValue != "" && flag.DefValue != "false" {
				usage += fmt.Sprintf(" (default: %s)", flag.DefValue)
			}
			fmt.Fprintf(os.Stderr, "  --%s: %s\n", flag.Name, usage)
		}
	})

	fmt.Fprintf(os.Stderr, "\nUse '%s --help' for more information.\n", cmd.CommandPath())
	return fmt.Errorf("invalid flag usage")
}

// handleUnknownCommand provides helpful error messages for unknown commands
func handleUnknownCommand(cmd *cobra.Command, unknownCmd string) error {
	fmt.Fprintf(os.Stderr, "Error: unknown command '%s' for '%s'\n\n", unknownCmd, cmd.Name())

	// Get command suggestions
	suggestions := suggestCommands(cmd, unknownCmd)
	if len(suggestions) > 0 {
		fmt.Fprintf(os.Stderr, "Did you mean one of these?\n")
		for _, suggestion := range suggestions {
			fmt.Fprintf(os.Stderr, "  %s\n", suggestion)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Show available commands
	fmt.Fprintf(os.Stderr, "Available commands:\n")
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden {
			fmt.Fprintf(os.Stderr, "  %-12s %s\n", subCmd.Name(), subCmd.Short)
		}
	}

	fmt.Fprintf(os.Stderr, "\nUse '%s --help' for more information about available commands.\n", cmd.Name())
	return fmt.Errorf("unknown command")
}

// suggestFlags suggests similar flag names using Levenshtein distance
func suggestFlags(cmd *cobra.Command, unknownFlag string) []string {
	var suggestions []string

	// Remove leading dashes from unknown flag
	cleanUnknown := strings.TrimLeft(unknownFlag, "-")

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden {
			distance := levenshteinDistance(cleanUnknown, flag.Name)
			// Suggest if distance is small relative to flag length
			if distance <= 2 || (len(flag.Name) > 4 && distance <= len(flag.Name)/2) {
				suggestions = append(suggestions, "--"+flag.Name)
			}
		}
	})

	// Also check shorthand flags
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden && flag.Shorthand != "" {
			if len(cleanUnknown) == 1 && cleanUnknown == flag.Shorthand {
				suggestions = append(suggestions, "-"+flag.Shorthand+" (--"+flag.Name+")")
			}
		}
	})

	// Sort suggestions by similarity (shorter distance first)
	sort.Slice(suggestions, func(i, j int) bool {
		return len(suggestions[i]) < len(suggestions[j])
	})

	// Limit to top 3 suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// suggestCommands suggests similar command names using Levenshtein distance
func suggestCommands(cmd *cobra.Command, unknownCmd string) []string {
	var suggestions []string

	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden {
			distance := levenshteinDistance(unknownCmd, subCmd.Name())
			// Suggest if distance is small relative to command length
			if distance <= 2 || (len(subCmd.Name()) > 4 && distance <= len(subCmd.Name())/2) {
				suggestions = append(suggestions, subCmd.Name())
			}
		}
	}

	// Also check aliases
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden {
			for _, alias := range subCmd.Aliases {
				distance := levenshteinDistance(unknownCmd, alias)
				if distance <= 2 {
					suggestions = append(suggestions, fmt.Sprintf("%s (alias for %s)", alias, subCmd.Name()))
				}
			}
		}
	}

	// Sort suggestions by similarity
	sort.Slice(suggestions, func(i, j int) bool {
		return len(suggestions[i]) < len(suggestions[j])
	})

	// Limit to top 3 suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// configureSubcommandErrorHandling configures error handling for all subcommands
func configureSubcommandErrorHandling(rootCmd *cobra.Command) {
	// Configure error handling for all commands recursively
	configureCommandErrorHandling(rootCmd)
	for _, cmd := range rootCmd.Commands() {
		configureSubcommandErrorHandling(cmd)
	}
}

// configureCommandErrorHandling configures error handling for a specific command
func configureCommandErrorHandling(cmd *cobra.Command) {
	// Set custom flag error function
	cmd.SetFlagErrorFunc(handleFlagError)

	// Wrap existing RunE function to provide better error context
	if cmd.RunE != nil {
		originalRunE := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			err := originalRunE(cmd, args)
			if err != nil {
				return enhanceError(cmd, err)
			}
			return nil
		}
	}

	// Wrap existing Run function to provide better error context
	if cmd.Run != nil {
		originalRun := cmd.Run
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			originalRun(cmd, args)
			return nil
		}
		cmd.Run = nil
	}
}

// enhanceError provides enhanced error messages with context and suggestions
func enhanceError(cmd *cobra.Command, err error) error {
	errorMsg := err.Error()

	// Configuration-related errors
	if strings.Contains(errorMsg, "configuration") || strings.Contains(errorMsg, "config") {
		fmt.Fprintf(os.Stderr, "\nConfiguration Error Help:\n")
		fmt.Fprintf(os.Stderr, "========================\n")
		fmt.Fprintf(os.Stderr, "• Use 'darrot config validate' to check your configuration\n")
		fmt.Fprintf(os.Stderr, "• Use 'darrot config show' to see current configuration values\n")
		fmt.Fprintf(os.Stderr, "• Use 'darrot config create' to generate a sample configuration file\n")
		fmt.Fprintf(os.Stderr, "\nConfiguration precedence (highest to lowest):\n")
		fmt.Fprintf(os.Stderr, "  1. CLI flags (--flag-name)\n")
		fmt.Fprintf(os.Stderr, "  2. Environment variables (DRT_*)\n")
		fmt.Fprintf(os.Stderr, "  3. Configuration file\n")
		fmt.Fprintf(os.Stderr, "  4. Default values\n")
	}

	// Discord-related errors
	if strings.Contains(errorMsg, "discord") || strings.Contains(errorMsg, "token") {
		fmt.Fprintf(os.Stderr, "\nDiscord Configuration Help:\n")
		fmt.Fprintf(os.Stderr, "==========================\n")
		fmt.Fprintf(os.Stderr, "• Set Discord token: DRT_DISCORD_TOKEN=your-bot-token\n")
		fmt.Fprintf(os.Stderr, "• Or use CLI flag: --discord-token your-bot-token\n")
		fmt.Fprintf(os.Stderr, "• Get a bot token at: https://discord.com/developers/applications\n")
	}

	// TTS-related errors
	if strings.Contains(errorMsg, "tts") || strings.Contains(errorMsg, "google") || strings.Contains(errorMsg, "cloud") {
		fmt.Fprintf(os.Stderr, "\nTTS Configuration Help:\n")
		fmt.Fprintf(os.Stderr, "======================\n")
		fmt.Fprintf(os.Stderr, "• Set Google Cloud credentials: DRT_TTS_GOOGLE_CLOUD_CREDENTIALS_PATH=/path/to/credentials.json\n")
		fmt.Fprintf(os.Stderr, "• Or use CLI flag: --google-cloud-credentials-path /path/to/credentials.json\n")
		fmt.Fprintf(os.Stderr, "• Get credentials at: https://console.cloud.google.com/apis/credentials\n")
		fmt.Fprintf(os.Stderr, "• Enable Text-to-Speech API in your Google Cloud project\n")
	}

	// File-related errors
	if strings.Contains(errorMsg, "file") || strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "directory") {
		fmt.Fprintf(os.Stderr, "\nFile System Help:\n")
		fmt.Fprintf(os.Stderr, "================\n")
		fmt.Fprintf(os.Stderr, "• Check that file paths exist and are accessible\n")
		fmt.Fprintf(os.Stderr, "• Ensure the application has read/write permissions\n")
		fmt.Fprintf(os.Stderr, "• Use absolute paths to avoid ambiguity\n")
	}

	// Network-related errors
	if strings.Contains(errorMsg, "network") || strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "timeout") {
		fmt.Fprintf(os.Stderr, "\nNetwork Connectivity Help:\n")
		fmt.Fprintf(os.Stderr, "=========================\n")
		fmt.Fprintf(os.Stderr, "• Check your internet connection\n")
		fmt.Fprintf(os.Stderr, "• Verify firewall settings allow outbound connections\n")
		fmt.Fprintf(os.Stderr, "• Check if Discord or Google Cloud services are accessible\n")
	}

	fmt.Fprintf(os.Stderr, "\nFor more help:\n")
	fmt.Fprintf(os.Stderr, "• Use '%s --help' for command usage\n", cmd.CommandPath())
	fmt.Fprintf(os.Stderr, "• Use 'darrot config validate' to check configuration\n")
	fmt.Fprintf(os.Stderr, "• Check the documentation for troubleshooting guides\n")

	return err
}

// getCustomUsageTemplate returns a custom usage template with better formatting
func getCustomUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}

For more help and documentation, visit: https://github.com/your-org/darrot
`
}

// getCustomHelpTemplate returns a custom help template with better formatting
func getCustomHelpTemplate() string {
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}
