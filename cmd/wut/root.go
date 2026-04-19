package main

import "github.com/spf13/cobra"

// Version is the compiled-in version string. Bumped by release tooling later.
const Version = "0.0.1"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wut",
		Short:         "Route accidental natural-language input at a shell prompt to an AI harness.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	// Cobra's default --version flag has no short form; add -v explicitly.
	cmd.Flags().BoolP("version", "v", false, "print version and exit")
	// Match the `version` subcommand's output ("0.0.1\n") rather than
	// Cobra's default "wut version 0.0.1\n".
	cmd.SetVersionTemplate("{{.Version}}\n")
	return cmd
}
