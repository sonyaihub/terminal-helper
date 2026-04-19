package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const completionLongHelp = `Generate a shell completion script.

Install (zsh, oh-my-zsh-compatible):
  wut completion zsh > "${fpath[1]}/_wut"
  autoload -U compinit && compinit

Install (zsh, no framework):
  mkdir -p ~/.zsh/completions
  wut completion zsh > ~/.zsh/completions/_wut
  echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
  echo 'autoload -U compinit && compinit' >> ~/.zshrc

Install (bash, with bash-completion):
  wut completion bash > $(brew --prefix)/etc/bash_completion.d/wut

Install (fish):
  wut completion fish > ~/.config/fish/completions/wut.fish`

func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion <zsh|bash|fish>",
		Short:                 "Generate a shell completion script.",
		Long:                  completionLongHelp,
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"zsh", "bash", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			}
			return fmt.Errorf("unsupported shell %q — supported: zsh, bash, fish", args[0])
		},
	}
	return cmd
}
