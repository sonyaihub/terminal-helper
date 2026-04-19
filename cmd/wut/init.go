package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/shell"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Print shell integration snippets.",
	}
	cmd.AddCommand(NewInitZshCmd(), NewInitBashCmd(), NewInitFishCmd())
	return cmd
}

func NewInitZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Print the zsh integration snippet to stdout.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(shell.ZshSnippet())
		},
	}
}

func NewInitBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Print the bash integration snippet to stdout.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(shell.BashSnippet())
		},
	}
}

func NewInitFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Print the fish integration snippet to stdout.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(shell.FishSnippet())
		},
	}
}
