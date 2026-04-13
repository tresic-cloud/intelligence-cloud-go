package version

import (
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	t.Run("Version defaults to dev", func(t *testing.T) {
		if Version != "dev" {
			t.Errorf("expected Version = %q, got %q", "dev", Version)
		}
	})

	t.Run("Commit defaults to unknown", func(t *testing.T) {
		if Commit != "unknown" {
			t.Errorf("expected Commit = %q, got %q", "unknown", Commit)
		}
	})

	t.Run("Date defaults to unknown", func(t *testing.T) {
		if Date != "unknown" {
			t.Errorf("expected Date = %q, got %q", "unknown", Date)
		}
	})
}

func TestStringDefault(t *testing.T) {
	// Ensure String() returns the expected format with default values.
	got := String()
	want := "dev (unknown, unknown)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestStringNeverEmpty(t *testing.T) {
	got := String()
	if got == "" {
		t.Error("String() must never return an empty string")
	}
}

func TestStringWithCustomValues(t *testing.T) {
	// Save originals and restore after the test.
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	})

	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "release values",
			version: "v0.1.0",
			commit:  "abc1234",
			date:    "2026-04-13T10:00:00Z",
			want:    "v0.1.0 (abc1234, 2026-04-13T10:00:00Z)",
		},
		{
			name:    "full commit hash",
			version: "v1.2.3",
			commit:  "deadbeefcafebabe1234567890abcdef12345678",
			date:    "2025-12-31T23:59:59Z",
			want:    "v1.2.3 (deadbeefcafebabe1234567890abcdef12345678, 2025-12-31T23:59:59Z)",
		},
		{
			name:    "pre-release version",
			version: "v0.0.1-rc.1",
			commit:  "f00ba42",
			date:    "2026-01-01T00:00:00Z",
			want:    "v0.0.1-rc.1 (f00ba42, 2026-01-01T00:00:00Z)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			Date = tt.date

			got := String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStringFormat(t *testing.T) {
	// Verify the structural contract: the string contains the version,
	// commit, and date separated by expected delimiters.
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	})

	Version = "v2.0.0"
	Commit = "aaa1111"
	Date = "2026-06-15T12:00:00Z"

	got := String()

	if !strings.HasPrefix(got, "v2.0.0") {
		t.Errorf("String() should start with Version, got %q", got)
	}
	if !strings.Contains(got, "aaa1111") {
		t.Errorf("String() should contain Commit, got %q", got)
	}
	if !strings.Contains(got, "2026-06-15T12:00:00Z") {
		t.Errorf("String() should contain Date, got %q", got)
	}
	if !strings.Contains(got, "(") || !strings.Contains(got, ")") {
		t.Errorf("String() should wrap commit and date in parentheses, got %q", got)
	}
}
