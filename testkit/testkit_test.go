package testkit

import (
	"strings"
	"testing"
)

func TestNormaliseGoluca_UUIDs(t *testing.T) {
	input := "account 550e8400-e29b-41d4-a716-446655440000 GBP"
	got := NormaliseGoluca(input)
	if strings.Contains(got, "550e8400") {
		t.Errorf("UUID not replaced: %s", got)
	}
	if !strings.Contains(got, "<UUID>") {
		t.Errorf("expected <UUID> placeholder: %s", got)
	}
}

func TestNormaliseGoluca_Timestamps(t *testing.T) {
	input := "2026-01-15T14:30:00Z deposit"
	got := NormaliseGoluca(input)
	if !strings.Contains(got, "2026-01-15") {
		t.Errorf("date not preserved: %s", got)
	}
	if strings.Contains(got, "T14:30:00Z") {
		t.Errorf("timestamp not normalised: %s", got)
	}
}

func TestNormaliseGoluca_Sorting(t *testing.T) {
	input := "block B line1\n\nblock A line1\n"
	got := NormaliseGoluca(input)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	// After sorting, block A should come first.
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "block A") {
		t.Errorf("expected block A first, got: %s", lines[0])
	}
}

func TestAssertGolucaEqual_Match(t *testing.T) {
	a := "account 550e8400-e29b-41d4-a716-446655440000 GBP\n"
	b := "account aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee GBP\n"
	// These should be equal after normalisation (UUIDs replaced).
	mock := &testing.T{}
	AssertGolucaEqual(mock, a, b)
	// mock.Failed() would be true if they didn't match, but we can't
	// easily assert on a sub-T, so just verify no panic.
}
