package detect

import (
	"strings"
	"unicode"
)

// Classification is the result of running Classify on an input line.
type Classification int

const (
	// PassThrough means the shell should produce its usual "command not found"
	// for the line; we do not believe it was natural language.
	PassThrough Classification = iota
	// Route means the line looks like natural language and should be handed to
	// the configured harness.
	Route
)

// ForcedMode captures a per-prompt mode override parsed from the line's
// leading prefix (see spec §13 1a).
type ForcedMode string

const (
	ForceNone        ForcedMode = ""
	ForceInteractive ForcedMode = "interactive"
	ForceHeadless    ForcedMode = "headless"
)

// Result is the richer output of Parse — it reports the classification, any
// forced mode implied by a prefix, and the line stripped of its prefix so
// callers have a clean prompt to forward to the harness.
type Result struct {
	Class  Classification
	Forced ForcedMode
	Line   string
}

// Options tweaks Parse / Classify with user-configured token sets.
type Options struct {
	// Passthrough first tokens — never route even if NL-shaped. Matched
	// case-sensitively against the first whitespace-delimited token.
	Passthrough []string
	// ExtraStopwords extends the built-in stopword set (spec §6). Case-insensitive.
	ExtraStopwords []string
	// ExtraInterrogatives extends the built-in interrogative set. Case-insensitive.
	ExtraInterrogatives []string
}

// Parse runs prefix handling, user-configured passthrough, then Classify.
// Prefix precedence, checked longest-first:
//
//	??    → Route, ForceHeadless, strip "??"
//	?!    → Route, ForceInteractive, strip "?!"
//	?     → Route, strip "?"
//	\     → Route, strip "\"
//	!     → PassThrough, no stripping
func Parse(line string, opts ...Options) Result {
	line = strings.TrimSpace(line)
	if line == "" {
		return Result{Class: PassThrough, Line: line}
	}
	switch {
	case strings.HasPrefix(line, "??"):
		return Result{Class: Route, Forced: ForceHeadless, Line: strings.TrimSpace(line[2:])}
	case strings.HasPrefix(line, "?!"):
		return Result{Class: Route, Forced: ForceInteractive, Line: strings.TrimSpace(line[2:])}
	case line[0] == '?':
		return Result{Class: Route, Line: strings.TrimSpace(line[1:])}
	case line[0] == '\\':
		return Result{Class: Route, Line: strings.TrimSpace(line[1:])}
	case line[0] == '!':
		return Result{Class: PassThrough, Line: line}
	}

	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	// User-configured passthrough first-tokens (spec §7 behavior.passthrough).
	if len(o.Passthrough) > 0 {
		first := firstToken(line)
		for _, tok := range o.Passthrough {
			if tok == first {
				return Result{Class: PassThrough, Line: line}
			}
		}
	}

	return Result{Class: Classify(line, o), Line: line}
}

func firstToken(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' {
			return line[:i]
		}
	}
	return line
}

// stopwords is the built-in fixed set from spec §6. Users extend this via
// Options.ExtraStopwords, not by modifying the map.
var stopwords = map[string]struct{}{
	"the": {}, "a": {}, "is": {}, "how": {}, "what": {}, "why": {},
	"can": {}, "i": {}, "my": {}, "to": {}, "do": {}, "does": {},
	"should": {},
}

// interrogatives is the built-in fixed set. Includes apostrophe-dropped
// contractions (`whats`, `hows`…) because users often type them without the
// apostrophe and none of them collide with real binaries on macOS/Linux.
var interrogatives = map[string]struct{}{
	"how": {}, "what": {}, "why": {}, "explain": {}, "write": {},
	"make": {}, "fix": {}, "help": {},
	"whats": {}, "hows": {}, "whos": {}, "wheres": {}, "whens": {},
}

// Classify returns Route when line looks like natural language the user meant
// for their AI harness, or PassThrough otherwise. See spec §6.
func Classify(line string, opts ...Options) Classification {
	line = strings.TrimSpace(line)
	if line == "" {
		return PassThrough
	}

	// Explicit escape hatches take precedence over everything else.
	switch line[0] {
	case '!':
		return PassThrough
	case '?', '\\':
		return Route
	}

	tokens := strings.Fields(line)

	// Hard gates.
	if len(tokens) < 3 {
		return PassThrough
	}
	if strings.ContainsAny(tokens[0], "/.-~$") {
		return PassThrough
	}
	if containsShellMetachar(line) {
		return PassThrough
	}

	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	// Soft signals — need at least two.
	signals := 0
	lower := strings.ToLower(line)
	lowerTokens := tokenizeWords(lower)
	if hasWord(lowerTokens, stopwords, o.ExtraStopwords) {
		signals++
	}
	if strings.ContainsRune(line, '?') {
		signals++
	}
	if strings.ContainsAny(line, "',") {
		signals++
	}
	if len(tokens) >= 6 {
		signals++
	}
	first := strings.ToLower(tokens[0])
	if _, ok := interrogatives[first]; ok {
		signals++
	} else if matchesExtra(first, o.ExtraInterrogatives) {
		signals++
	}

	if signals >= 2 {
		return Route
	}
	return PassThrough
}

// containsShellMetachar flags the unquoted shell metacharacters listed in
// spec §6. We do a straight-quote scan — good enough for v1.
func containsShellMetachar(line string) bool {
	inSingle := false
	inDouble := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch c {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			continue
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			continue
		}
		if inSingle || inDouble {
			continue
		}
		switch c {
		case '|', '>', '<', '&', ';', '`':
			return true
		case '$':
			if i+1 < len(line) && line[i+1] == '(' {
				return true
			}
		}
	}
	return false
}

// tokenizeWords splits on non-letter/digit runs so that "i" in "do i" and
// "difference" are both recognizable as whole words.
func tokenizeWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// hasWord reports whether any token matches the built-in set or the
// user-supplied extras (case-insensitive).
func hasWord(tokens []string, builtin map[string]struct{}, extras []string) bool {
	for _, t := range tokens {
		if _, ok := builtin[t]; ok {
			return true
		}
		if matchesExtra(t, extras) {
			return true
		}
	}
	return false
}

func matchesExtra(token string, extras []string) bool {
	for _, e := range extras {
		if strings.EqualFold(token, e) {
			return true
		}
	}
	return false
}

// Explanation carries the full reasoning behind a classification decision.
type Explanation struct {
	Class          Classification
	Forced         ForcedMode
	Line           string
	FirstToken     string
	TokenCount     int
	HardGate       string
	PrefixOverride string
	Signals        []Signal
}

// Signal is one soft heuristic that fired during classification.
type Signal struct {
	Name   string
	Token  string
	Source string
}

// Explain replays the classifier's decision for line and records every signal
// that contributed. It is intended for human-facing output only — callers that
// need performance should use Classify or Parse instead.
func Explain(line string, opts ...Options) Explanation {
	line = strings.TrimSpace(line)
	exp := Explanation{Class: PassThrough, Line: line}

	if line == "" {
		exp.HardGate = "empty line"
		return exp
	}

	// Prefix handling — runs before everything else.
	switch {
	case strings.HasPrefix(line, "??"):
		exp.PrefixOverride = "??"
		exp.Forced = ForceHeadless
		exp.Class = Route
		exp.Line = strings.TrimSpace(line[2:])
		return exp
	case strings.HasPrefix(line, "?!"):
		exp.PrefixOverride = "?!"
		exp.Forced = ForceInteractive
		exp.Class = Route
		exp.Line = strings.TrimSpace(line[2:])
		return exp
	case line[0] == '?':
		exp.PrefixOverride = "?"
		exp.Class = Route
		exp.Line = strings.TrimSpace(line[1:])
		return exp
	case line[0] == '\\':
		exp.PrefixOverride = "\\"
		exp.Class = Route
		exp.Line = strings.TrimSpace(line[1:])
		return exp
	case line[0] == '!':
		exp.PrefixOverride = "!"
		exp.Class = PassThrough
		return exp
	}

	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	tokens := strings.Fields(line)
	exp.TokenCount = len(tokens)
	if len(tokens) > 0 {
		exp.FirstToken = tokens[0]
	}

	if len(tokens) < 3 {
		exp.HardGate = "fewer than 3 tokens"
		return exp
	}
	if strings.ContainsAny(tokens[0], "/.-~$") {
		exp.HardGate = "first token contains path character"
		return exp
	}
	if containsShellMetachar(line) {
		exp.HardGate = "line contains shell metacharacter"
		return exp
	}

	lower := strings.ToLower(line)
	lowerTokens := tokenizeWords(lower)

	// Stopword signal — report the first matching token.
	for _, t := range lowerTokens {
		if _, ok := stopwords[t]; ok {
			exp.Signals = append(exp.Signals, Signal{Name: "stopword hit", Token: t, Source: "built-in"})
			break
		}
		if matchesExtra(t, o.ExtraStopwords) {
			exp.Signals = append(exp.Signals, Signal{Name: "stopword hit", Token: t, Source: "user-added"})
			break
		}
	}

	if strings.ContainsRune(line, '?') {
		exp.Signals = append(exp.Signals, Signal{Name: "contains question mark", Token: "?"})
	}
	if strings.ContainsAny(line, "',") {
		exp.Signals = append(exp.Signals, Signal{Name: "contains apostrophe or comma"})
	}
	if len(tokens) >= 6 {
		exp.Signals = append(exp.Signals, Signal{Name: "6 or more tokens"})
	}

	first := strings.ToLower(tokens[0])
	if _, ok := interrogatives[first]; ok {
		exp.Signals = append(exp.Signals, Signal{Name: "first token is interrogative", Token: first, Source: "built-in"})
	} else if matchesExtra(first, o.ExtraInterrogatives) {
		exp.Signals = append(exp.Signals, Signal{Name: "first token is interrogative", Token: first, Source: "user-added"})
	}

	if len(exp.Signals) >= 2 {
		exp.Class = Route
	}
	return exp
}
