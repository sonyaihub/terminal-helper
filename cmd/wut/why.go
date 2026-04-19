package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/detect"
)

func NewWhyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "why <line>",
		Short: "Explain how the classifier would handle a line (without invoking the harness).",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			line := strings.Join(args, " ")
			exp := detect.Explain(line, detect.Options{
				Passthrough:         cfg.Behavior.Passthrough,
				ExtraStopwords:      cfg.Detection.ExtraStopwords,
				ExtraInterrogatives: cfg.Detection.ExtraInterrogatives,
			})
			printExplanation(line, exp)
			return nil
		},
	}
}

func printExplanation(original string, exp detect.Explanation) {
	fmt.Printf("%q\n", original)

	if exp.Class == detect.Route {
		fmt.Println("  → would ROUTE to harness")
	} else {
		fmt.Println("  → would PASSTHROUGH")
	}

	if exp.PrefixOverride != "" {
		fmt.Printf("  prefix override: %q (forces route, strips prefix)\n", exp.PrefixOverride)
		if exp.Line != "" {
			fmt.Printf("  stripped line: %q\n", exp.Line)
		}
		return
	}

	if exp.HardGate != "" {
		fmt.Printf("  hard gate tripped: %s\n", exp.HardGate)
		return
	}

	if exp.FirstToken != "" {
		fmt.Printf("  first token: %s\n", exp.FirstToken)
	}
	fmt.Printf("  tokens: %d\n", exp.TokenCount)

	total := len(exp.Signals)
	fmt.Printf("  signals (%d of 2 required):\n", total)
	for _, s := range exp.Signals {
		line := fmt.Sprintf("    ✔ %s", s.Name)
		if s.Token != "" {
			line += fmt.Sprintf(": %q", s.Token)
		}
		if s.Source != "" {
			line += fmt.Sprintf(" (%s)", s.Source)
		}
		fmt.Println(line)
	}
}
