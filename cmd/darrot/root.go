package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	logLevel string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "darrot",
	Short: "Discord TTS Bot",
	Long: `darrot is a Discord Parrot Text-to-Speech (TTS) AI application that listens to Discord chat channels and converts text messages to speech.

The bot joins Discord voice channels, creates voice-text channel pairings, and provides real-time TTS processing with user privacy controls and administrative settings.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set up comprehensive error handling
	setupErrorHandling(rootCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.darrot.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "log level (DEBUG, INFO, WARN, ERROR)")

	// Set up custom completion functions
	setupRootCompletions()

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}

// setupRootCompletions configures custom completion functions for root command flags
func setupRootCompletions() {
	// Custom completion for log-level flag
	_ = rootCmd.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Custom completion for config file paths
	_ = rootCmd.RegisterFlagCompletionFunc("config", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml", "json", "toml"}, cobra.ShellCompDirectiveFilterFileExt
	})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".darrot" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath("/etc/darrot/")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".darrot")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
