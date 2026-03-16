package testkit

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var uuidRe = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
var timestampRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`)

// NormaliseGoluca strips UUIDs, normalises timestamps to dates, and sorts lines
// within transaction blocks for deterministic comparison.
func NormaliseGoluca(raw string) string {
	// Replace UUIDs with placeholder.
	s := uuidRe.ReplaceAllString(raw, "<UUID>")
	// Normalise full timestamps to date-only.
	s = timestampRe.ReplaceAllStringFunc(s, func(ts string) string {
		if len(ts) >= 10 {
			return ts[:10]
		}
		return ts
	})

	// Sort transaction blocks (blocks separated by blank lines) for determinism.
	blocks := splitBlocks(s)
	sort.Strings(blocks)
	return strings.Join(blocks, "\n\n") + "\n"
}

// splitBlocks splits text into blocks separated by blank lines.
func splitBlocks(s string) []string {
	var blocks []string
	var current []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			}
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}

// AssertGolucaEqual compares two .goluca strings and reports a diff on mismatch.
func AssertGolucaEqual(t *testing.T, got, want string) {
	t.Helper()
	gotNorm := NormaliseGoluca(got)
	wantNorm := NormaliseGoluca(want)
	if gotNorm == wantNorm {
		return
	}

	gotLines := strings.Split(gotNorm, "\n")
	wantLines := strings.Split(wantNorm, "\n")

	t.Errorf("goluca mismatch:\n--- got (%d lines) ---\n%s\n--- want (%d lines) ---\n%s",
		len(gotLines), gotNorm, len(wantLines), wantNorm)
}

// Golden compares got against a golden file in testdata/, updating it if the
// GOLDEN_UPDATE env var is set.
func Golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".goluca")

	if os.Getenv("GOLDEN_UPDATE") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s not found (set GOLDEN_UPDATE=1 to create): %v", path, err)
	}
	AssertGolucaEqual(t, got, string(want))
}
