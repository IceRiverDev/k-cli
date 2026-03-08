package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for k-cli and print them to stdout.

Bash:
  # Load for the current session:
  source <(k-cli completion bash)

  # Persist across sessions (Linux):
  k-cli completion bash > /etc/bash_completion.d/k-cli

  # Persist across sessions (macOS with Homebrew bash-completion@2):
  k-cli completion bash > $(brew --prefix)/etc/bash_completion.d/k-cli

Zsh:
  # Load for the current session:
  source <(k-cli completion zsh)

  # Persist across sessions (add to ~/.zshrc):
  echo 'source <(k-cli completion zsh)' >> ~/.zshrc

  # Or install to a fpath directory:
  k-cli completion zsh > "${fpath[1]}/_k-cli"

  # With oh-my-zsh:
  k-cli completion zsh > ~/.oh-my-zsh/completions/_k-cli

Fish:
  k-cli completion fish > ~/.config/fish/completions/k-cli.fish

PowerShell:
  # Load for the current session:
  k-cli completion powershell | Out-String | Invoke-Expression

  # Persist across sessions (add to your PowerShell profile):
  k-cli completion powershell >> $PROFILE`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	// Override PersistentPreRunE so completion works without a kubeconfig.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %q", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
