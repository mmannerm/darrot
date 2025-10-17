package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Version information - set during build with -ldflags
var (
	version = "dev"     // Set via -ldflags "-X main.version=x.y.z"
	commit  = "unknown" // Set via -ldflags "-X main.commit=abc123"
	date    = "unknown" // Set via -ldflags "-X main.date=2024-01-01"
)

// VersionInfo represents the version information structure
type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display version information for the darrot Discord TTS bot.

Shows the current version, git commit hash, and build date.
Use --format json for machine-readable output.`,
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")

		versionInfo := VersionInfo{
			Version: version,
			Commit:  commit,
			Date:    date,
		}

		switch format {
		case "json":
			jsonOutput, err := json.MarshalIndent(versionInfo, "", "  ")
			if err != nil {
				fmt.Printf("Error formatting JSON: %v\n", err)
				return
			}
			fmt.Println(string(jsonOutput))
		default:
			fmt.Printf("darrot Discord TTS Bot\n")
			fmt.Printf("Version: %s\n", versionInfo.Version)
			fmt.Printf("Commit:  %s\n", versionInfo.Commit)
			fmt.Printf("Date:    %s\n", versionInfo.Date)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Add format flag for JSON output
	versionCmd.Flags().StringP("format", "f", "text", "output format (text, json)")
}
