package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for darrot.

The completion script for each shell will be output to stdout. You can source it or
write it to a file and source it from your shell's configuration file.

Bash:
  # Load completion for current session
  source <(darrot completion bash)
  
  # Install completion permanently (Linux)
  darrot completion bash > /etc/bash_completion.d/darrot
  
  # Install completion permanently (macOS with Homebrew)
  darrot completion bash > $(brew --prefix)/etc/bash_completion.d/darrot

Zsh:
  # Load completion for current session
  source <(darrot completion zsh)
  
  # Install completion permanently
  darrot completion zsh > "${fpath[1]}/_darrot"
  
  # You may need to start a new shell for this setup to take effect

Fish:
  # Load completion for current session
  darrot completion fish | source
  
  # Install completion permanently
  darrot completion fish > ~/.config/fish/completions/darrot.fish

`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := args[0]

		switch shell {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell: %s", shell)
		}
	},
}

// completionBashCmd represents the completion bash command
var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long: `Generate bash completion script for darrot.

To load completion for the current session:
  source <(darrot completion bash)

To install completion permanently on Linux:
  darrot completion bash > /etc/bash_completion.d/darrot

To install completion permanently on macOS (with Homebrew):
  darrot completion bash > $(brew --prefix)/etc/bash_completion.d/darrot

You may need to start a new shell for this setup to take effect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Root().GenBashCompletion(os.Stdout)
	},
}

// completionZshCmd represents the completion zsh command
var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long: `Generate zsh completion script for darrot.

To load completion for the current session:
  source <(darrot completion zsh)

To install completion permanently:
  darrot completion zsh > "${fpath[1]}/_darrot"

You may need to start a new shell for this setup to take effect.

Note: If you're using Oh My Zsh, you can place the completion file in:
  ~/.oh-my-zsh/completions/_darrot`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Root().GenZshCompletion(os.Stdout)
	},
}

// completionFishCmd represents the completion fish command
var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Long: `Generate fish completion script for darrot.

To load completion for the current session:
  darrot completion fish | source

To install completion permanently:
  darrot completion fish > ~/.config/fish/completions/darrot.fish

You may need to start a new shell for this setup to take effect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// Add individual shell completion commands as subcommands
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
}
