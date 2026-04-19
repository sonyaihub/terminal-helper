package detect

import (
	"strings"
	"testing"
)

func TestClassifyContractionsRoute(t *testing.T) {
	// "whats going on" → 3 tokens, "whats" is an interrogative (apostrophe-dropped)
	// and len>=6 is still false → only 1 signal, still PassThrough.
	// "whats going on here now bro" → still 1 signal (interrogative) + len>=6 = 2.
	if got := Classify("whats going on here now bro"); got != Route {
		t.Fatalf("expected Route for contraction-interrogative long line, got %v", got)
	}
	// "hows the weather today" → stopword "the" + interrogative "hows" = 2.
	if got := Classify("hows the weather today"); got != Route {
		t.Fatalf("expected Route for 'hows the weather today', got %v", got)
	}
}

func TestClassifyExtraSets(t *testing.T) {
	// Without config — "yo bruh sup" has 0 signals → PassThrough.
	if got := Classify("yo bruh sup"); got != PassThrough {
		t.Fatalf("unconfigured slang should passthrough, got %v", got)
	}
	// With extras registered, "yo" is an interrogative and "bruh" is a stopword → 2 signals.
	opts := Options{
		ExtraStopwords:      []string{"bruh"},
		ExtraInterrogatives: []string{"yo"},
	}
	if got := Classify("yo bruh sup", opts); got != Route {
		t.Fatalf("configured slang should route, got %v", got)
	}
}

func TestParsePassthroughTokens(t *testing.T) {
	opts := Options{Passthrough: []string{"howto", "make"}}
	// Would normally route (interrogative "make" + stopwords), but config
	// says pass through.
	if got := Parse("make all the things for me please", opts); got.Class != PassThrough {
		t.Fatalf("expected passthrough for user-configured token, got %+v", got)
	}
	// Different first token still classifies normally.
	if got := Parse("how do I rebase onto main", opts); got.Class != Route {
		t.Fatalf("unrelated token should route, got %+v", got)
	}
}

func TestParsePrefixes(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		wantClass  Classification
		wantForced ForcedMode
		wantLine   string
	}{
		{"double-q forces headless", "??how do I rebase onto main", Route, ForceHeadless, "how do I rebase onto main"},
		{"q-bang forces interactive", "?!fix this regex for me", Route, ForceInteractive, "fix this regex for me"},
		{"single q routes no force", "? short", Route, ForceNone, "short"},
		{"backslash routes no force", "\\ foo bar", Route, ForceNone, "foo bar"},
		{"bang passes through", "!how do I", PassThrough, ForceNone, "!how do I"},
		{"plain NL falls through to classifier", "how do I rebase onto main", Route, ForceNone, "how do I rebase onto main"},
		{"plain typo falls through to classifier", "gti status", PassThrough, ForceNone, "gti status"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.line)
			if got.Class != tc.wantClass || got.Forced != tc.wantForced || got.Line != tc.wantLine {
				t.Fatalf("Parse(%q) = %+v; want {Class:%v Forced:%q Line:%q}", tc.line, got, tc.wantClass, tc.wantForced, tc.wantLine)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		line string
		want Classification
	}{
		// Real commands — must pass through.
		{"ls flags", "ls -la", PassThrough},
		{"git status", "git status", PassThrough},
		{"cd tilde", "cd ~/tmp", PassThrough},
		{"relative script", "./script.sh", PassThrough},
		{"python -V", "python3 -V", PassThrough},

		// Typos — must pass through.
		{"single typo", "gti", PassThrough},
		{"single sl", "sl", PassThrough},
		{"two-word typo", "pythno script.py", PassThrough},
		{"git typo", "gti statsu", PassThrough},

		// Natural language — must route.
		{"rebase q", "how do I rebase onto main", Route},
		{"reset vs revert", "what is the difference between git reset and git revert", Route},
		{"regex explain", "explain what this regex does in plain english", Route},

		// Escape hatch.
		{"qmark prefix", "? one", Route},
		{"backslash prefix", "\\ foo bar", Route},

		// Passthrough prefix beats heuristic.
		{"bang prefix", "!how do I rebase onto main", PassThrough},

		// Shell metachars anywhere → pass through.
		{"pipe in prose", "how do i grep | sort", PassThrough},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.line)
			if got != tc.want {
				t.Fatalf("Classify(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestExplainHardGates(t *testing.T) {
	cases := []struct {
		line     string
		wantGate string
	}{
		{"", "empty line"},
		{"ls", "fewer than 3 tokens"},
		{"./script.sh foo bar", "first token contains path character"},
	}
	for _, tc := range cases {
		got := Explain(tc.line)
		if got.HardGate == "" {
			t.Errorf("Explain(%q): expected hard gate %q, got none", tc.line, tc.wantGate)
			continue
		}
		if !strings.Contains(got.HardGate, tc.wantGate) {
			t.Errorf("Explain(%q).HardGate = %q, want to contain %q", tc.line, got.HardGate, tc.wantGate)
		}
		if len(got.Signals) != 0 {
			t.Errorf("Explain(%q): expected no signals when hard-gated, got %v", tc.line, got.Signals)
		}
	}
}

func TestExplainRouteSignals(t *testing.T) {
	got := Explain("how do I rebase")
	if got.Class != Route {
		t.Fatalf("Explain(\"how do I rebase\").Class = %v, want Route", got.Class)
	}
	if len(got.Signals) < 2 {
		t.Fatalf("expected at least 2 signals, got %d: %v", len(got.Signals), got.Signals)
	}
	var hasInterrogative bool
	for _, s := range got.Signals {
		if strings.Contains(s.Name, "interrogative") {
			hasInterrogative = true
		}
	}
	if !hasInterrogative {
		t.Errorf("expected an interrogative signal in %v", got.Signals)
	}
}

func TestExplainPrefixOverride(t *testing.T) {
	got := Explain("? anything works")
	if got.PrefixOverride != "?" {
		t.Errorf("PrefixOverride = %q, want %q", got.PrefixOverride, "?")
	}
	if got.Class != Route {
		t.Errorf("Class = %v, want Route", got.Class)
	}
	if len(got.Signals) != 0 {
		t.Errorf("expected no soft signals with prefix override, got %v", got.Signals)
	}
}

func TestExplainUserAddedKeyword(t *testing.T) {
	got := Explain("deploy the thing", Options{ExtraInterrogatives: []string{"deploy"}})
	if got.Class != Route {
		t.Fatalf("Class = %v, want Route", got.Class)
	}
	var found bool
	for _, s := range got.Signals {
		if strings.Contains(s.Name, "interrogative") && s.Token == "deploy" && s.Source == "user-added" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected user-added interrogative signal for 'deploy', signals: %v", got.Signals)
	}
}
