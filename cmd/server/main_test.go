package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailClosedConfiguration(t *testing.T) {
	tests := []struct{ name, token, path, duration, want string }{
		{"whitespace-only token", " \t\n", "", "300", "at least 32"},
		{"short token", "short", "", "300", "at least 32"},
		{"whitespace token", strings.Repeat("x", 32) + " internal", "", "300", "at least 32"},
		{"missing token file", strings.Repeat("x", 32), "does-not-exist", "300", "read token file"},
		{"invalid duration", strings.Repeat("x", 32), "", "not-a-number", "1..3600"},
		{"excessive duration", strings.Repeat("x", 32), "", "3601", "1..3600"},
		{"zero duration", strings.Repeat("x", 32), "", "0", "1..3600"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GSMSNIFFER_TOKEN", tt.token)
			t.Setenv("GSMSNIFFER_TOKEN_FILE", tt.path)
			t.Setenv("GSMSNIFFER_MAX_DURATION_SECONDS", tt.duration)
			if err := run(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestTokenFileOverridesEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("short\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GSMSNIFFER_TOKEN", strings.Repeat("x", 40))
	t.Setenv("GSMSNIFFER_TOKEN_FILE", path)
	if err := run(); err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("error=%v", err)
	}
}

func TestLoadTokenOptionalAndFailClosed(t *testing.T) {
	valid := strings.Repeat("v", 40)
	for _, tt := range []struct {
		name, token, fileContent, want string
		file, wantError                bool
	}{
		{name: "empty means optional"},
		{name: "valid environment", token: valid, want: valid},
		{name: "short environment", token: "short", wantError: true},
		{name: "whitespace environment", token: "\n\t ", wantError: true},
		{name: "internal whitespace", token: valid + "\u2003x", wantError: true},
		{name: "empty file", file: true, wantError: true},
		{name: "empty file overrides environment", token: valid, file: true, wantError: true},
		{name: "whitespace file", file: true, fileContent: " \n\t", wantError: true},
		{name: "invalid file", file: true, fileContent: "short", wantError: true},
		{name: "valid file with newline", file: true, fileContent: valid + "\n", want: valid},
		{name: "valid file overrides invalid environment", token: "short", file: true, fileContent: valid, want: valid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GSMSNIFFER_TOKEN", tt.token)
			t.Setenv("GSMSNIFFER_TOKEN_FILE", "")
			if tt.file {
				path := filepath.Join(t.TempDir(), "token")
				if err := os.WriteFile(path, []byte(tt.fileContent), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("GSMSNIFFER_TOKEN_FILE", path)
			}
			got, err := loadToken()
			if (err != nil) != tt.wantError || got != tt.want {
				t.Fatalf("token load result mismatch: error=%v", err)
			}
		})
	}
	t.Run("unset sources", func(t *testing.T) {
		t.Setenv("GSMSNIFFER_TOKEN", "")
		t.Setenv("GSMSNIFFER_TOKEN_FILE", "")
		if err := os.Unsetenv("GSMSNIFFER_TOKEN"); err != nil {
			t.Fatal(err)
		}
		if err := os.Unsetenv("GSMSNIFFER_TOKEN_FILE"); err != nil {
			t.Fatal(err)
		}
		if token, err := loadToken(); token != "" || err != nil {
			t.Fatalf("unset sources: error=%v", err)
		}
	})
	t.Run("unreadable configured source", func(t *testing.T) {
		t.Setenv("GSMSNIFFER_TOKEN", valid)
		t.Setenv("GSMSNIFFER_TOKEN_FILE", filepath.Join(t.TempDir(), "missing"))
		if token, err := loadToken(); token != "" || err == nil {
			t.Fatalf("missing file must fail closed: error=%v", err)
		}
	})
}
