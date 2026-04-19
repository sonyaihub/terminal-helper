package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/ui"
)

// Placement names map to the two user-extensible classifier sets.
// `first-word` extends detection.extra_interrogatives (only fires as the first
// token on a line); `anywhere` extends detection.extra_stopwords (soft signal
// anywhere on the line). See internal/detect/heuristic.go for the semantics.
const (
	placementFirstWord = "first-word"
	placementAnywhere  = "anywhere"
)

func NewKeywordsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keywords",
		Short: "Add, list, or remove keywords that make the classifier route input to the harness.",
	}
	cmd.AddCommand(newKeywordsList(), newKeywordsAdd(), newKeywordsRemove())
	return cmd
}

func newKeywordsList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show user-added keywords, grouped by placement.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			fmt.Println("First-word triggers (detection.extra_interrogatives):")
			printKeywordList(cfg.Detection.ExtraInterrogatives)
			fmt.Println("Anywhere signals (detection.extra_stopwords):")
			printKeywordList(cfg.Detection.ExtraStopwords)
			return nil
		},
	}
}

func printKeywordList(items []string) {
	if len(items) == 0 {
		fmt.Println("  (none)")
		return
	}
	sorted := append([]string{}, items...)
	sort.Strings(sorted)
	for _, k := range sorted {
		fmt.Printf("  - %s\n", k)
	}
}

func newKeywordsAdd() *cobra.Command {
	var firstWord, anywhere bool
	cmd := &cobra.Command{
		Use:   "add <word>",
		Short: "Add a keyword. Without --first-word/--anywhere, prompts for placement.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if firstWord && anywhere {
				return fmt.Errorf("pick one of --first-word or --anywhere, not both")
			}
			word, err := normalizeKeyword(args[0])
			if err != nil {
				return err
			}

			placement, err := resolvePlacement(firstWord, anywhere)
			if err != nil {
				return err
			}

			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}
			added := false
			switch placement {
			case placementFirstWord:
				cfg.Detection.ExtraInterrogatives, added = appendUnique(cfg.Detection.ExtraInterrogatives, word)
			case placementAnywhere:
				cfg.Detection.ExtraStopwords, added = appendUnique(cfg.Detection.ExtraStopwords, word)
			}
			if !added {
				fmt.Printf("• %q already in the %s set — no change\n", word, placement)
				return nil
			}
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ added %q as %s keyword\n", word, placement)
			return nil
		},
	}
	cmd.Flags().BoolVar(&firstWord, "first-word", false, "fires only when the keyword is the first token on the line")
	cmd.Flags().BoolVar(&anywhere, "anywhere", false, "soft signal anywhere on the line")
	return cmd
}

func newKeywordsRemove() *cobra.Command {
	var firstWord, anywhere bool
	cmd := &cobra.Command{
		Use:   "remove <word>",
		Short: "Remove a keyword. If the word is in both sets, pass --first-word or --anywhere.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if firstWord && anywhere {
				return fmt.Errorf("pick one of --first-word or --anywhere, not both")
			}
			word, err := normalizeKeyword(args[0])
			if err != nil {
				return err
			}

			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}

			inFirstWord := containsFold(cfg.Detection.ExtraInterrogatives, word)
			inAnywhere := containsFold(cfg.Detection.ExtraStopwords, word)

			switch {
			case !inFirstWord && !inAnywhere:
				return fmt.Errorf("%q is not in either keyword set (run `wut keywords list`)", word)
			case inFirstWord && inAnywhere && !firstWord && !anywhere:
				return fmt.Errorf("%q is in both sets — rerun with --first-word or --anywhere", word)
			}

			target := placementAnywhere
			switch {
			case firstWord:
				target = placementFirstWord
			case anywhere:
				target = placementAnywhere
			case inFirstWord:
				target = placementFirstWord
			}

			switch target {
			case placementFirstWord:
				if !inFirstWord {
					return fmt.Errorf("%q is not in the first-word set", word)
				}
				cfg.Detection.ExtraInterrogatives = removeFold(cfg.Detection.ExtraInterrogatives, word)
			case placementAnywhere:
				if !inAnywhere {
					return fmt.Errorf("%q is not in the anywhere set", word)
				}
				cfg.Detection.ExtraStopwords = removeFold(cfg.Detection.ExtraStopwords, word)
			}
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ removed %q from %s\n", word, target)
			return nil
		},
	}
	cmd.Flags().BoolVar(&firstWord, "first-word", false, "disambiguates removal when the word is in both sets")
	cmd.Flags().BoolVar(&anywhere, "anywhere", false, "disambiguates removal when the word is in both sets")
	return cmd
}

// resolvePlacement returns the chosen placement, either from flags or by
// launching the interactive picker. Exactly one of firstWord/anywhere may be
// set — the caller validates that invariant.
func resolvePlacement(firstWord, anywhere bool) (string, error) {
	switch {
	case firstWord:
		return placementFirstWord, nil
	case anywhere:
		return placementAnywhere, nil
	}
	return pickPlacement()
}

func pickPlacement() (string, error) {
	opts := []ui.Option{
		{Label: "first-word trigger", Hint: "e.g. 'debug my code …' — fires only when it's the first token"},
		{Label: "anywhere signal", Hint: "soft signal anywhere on the line (boosts routing)"},
	}
	idx, err := ui.Select("? Where should this keyword trigger routing?", opts, 0)
	if err != nil {
		if errors.Is(err, ui.ErrNotTTY) {
			return "", fmt.Errorf("no TTY for interactive picker — rerun with --first-word or --anywhere")
		}
		return "", err
	}
	switch idx {
	case 0:
		return placementFirstWord, nil
	case 1:
		return placementAnywhere, nil
	}
	return "", fmt.Errorf("unexpected picker index %d", idx)
}

// normalizeKeyword lowercases the word and rejects whitespace. Both built-in
// sets are lowercase and the classifier already compares case-insensitively
// (see detect.matchesExtra), so we store lowercase to keep the config file
// from growing noise like "Debug" alongside "debug".
func normalizeKeyword(raw string) (string, error) {
	w := strings.TrimSpace(raw)
	if w == "" {
		return "", fmt.Errorf("keyword cannot be empty")
	}
	if strings.ContainsAny(w, " \t\n") {
		return "", fmt.Errorf("keyword %q contains whitespace; add one word at a time", w)
	}
	return strings.ToLower(w), nil
}

func appendUnique(list []string, word string) ([]string, bool) {
	if containsFold(list, word) {
		return list, false
	}
	return append(list, word), true
}

func containsFold(list []string, word string) bool {
	for _, item := range list {
		if strings.EqualFold(item, word) {
			return true
		}
	}
	return false
}

func removeFold(list []string, word string) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		if !strings.EqualFold(item, word) {
			out = append(out, item)
		}
	}
	return out
}
