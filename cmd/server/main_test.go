package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailClosedConfiguration(t *testing.T) {
	tests := []struct{ name, token, path, duration, want string }{
		{"missing token", "", "", "300", "at least 32"},
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
