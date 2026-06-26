package version

import (
	"strings"
	"testing"
)

func TestString_DevBuildIncludesCommit(t *testing.T) {
	got := String()
	t.Logf("resolved version: %q", got)

	if got == "" {
		t.Fatal("String() returned empty")
	}

	if Version == "dev" {
		// Unstamped build: should be plain "dev" only when VCS info is
		// unavailable; otherwise "dev-<commit>" (optionally "-dirty").
		if got != "dev" && !strings.HasPrefix(got, "dev-") {
			t.Fatalf("dev build version = %q, want \"dev\" or \"dev-<commit>\"", got)
		}
	} else if got != Version {
		t.Fatalf("stamped build version = %q, want %q verbatim", got, Version)
	}
}
