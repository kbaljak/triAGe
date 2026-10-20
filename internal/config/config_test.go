package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempConfigHome points os.UserConfigDir() at a fresh temp directory for
// the duration of a test, so Load/Save never touch the real config file.
func withTempConfigHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // os.UserConfigDir() honors this on Linux
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	withTempConfigHome(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.AgentPaths == nil {
		t.Error("expected a non-nil, empty AgentPaths map")
	}
	if len(c.AgentPaths) != 0 || len(c.Profiles) != 0 {
		t.Error("expected an empty config")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempConfigHome(t)

	c := Config{AgentPaths: map[string]string{"claude": "/custom/claude"}}
	if err := c.AddProfile("claude", "personal", "/home/user/personal-agents/.claude"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AgentPaths["claude"] != "/custom/claude" {
		t.Errorf("AgentPaths[claude] = %q, want /custom/claude", loaded.AgentPaths["claude"])
	}
	if len(loaded.Profiles["claude"]) != 1 || loaded.Profiles["claude"][0].Name != "personal" {
		t.Errorf("Profiles[claude] = %+v, want one profile named personal", loaded.Profiles["claude"])
	}

	// File actually landed where expected, not somewhere stray.
	dir, _ := os.UserConfigDir()
	if _, err := os.Stat(filepath.Join(dir, "triage", "config.json")); err != nil {
		t.Errorf("config file not found where expected: %v", err)
	}
}

func TestAddProfileValidation(t *testing.T) {
	c := Config{}
	if err := c.AddProfile("claude", "personal", "/some/path"); err != nil {
		t.Fatalf("first AddProfile: %v", err)
	}

	cases := []struct {
		name string
		want string // substring expected in the error
	}{
		{"", "empty"},
		{"   ", "empty"},
		{"default", "reserved"},
		{"DEFAULT", "reserved"},
		{"personal", "already exists"},
		{"Personal", "already exists"}, // case-insensitive collision
	}
	for _, c2 := range cases {
		t.Run(c2.name, func(t *testing.T) {
			err := c.AddProfile("claude", c2.name, "/other/path")
			if err == nil {
				t.Fatalf("AddProfile(%q) succeeded, want an error", c2.name)
			}
		})
	}

	// A distinct agent isn't affected by "claude"'s profiles.
	if err := c.AddProfile("codex", "personal", "/some/other/path"); err != nil {
		t.Errorf("AddProfile for a different agent should succeed: %v", err)
	}
}

func TestDeleteProfile(t *testing.T) {
	c := Config{}
	_ = c.AddProfile("claude", "work", "/work")
	_ = c.AddProfile("claude", "personal", "/personal")

	c.DeleteProfile("claude", "Work") // case-insensitive
	if len(c.Profiles["claude"]) != 1 || c.Profiles["claude"][0].Name != "personal" {
		t.Errorf("after delete: %+v, want only \"personal\" left", c.Profiles["claude"])
	}

	// Deleting something that isn't there is a no-op, not a crash.
	c.DeleteProfile("claude", "nonexistent")
	c.DeleteProfile("nonexistent-agent", "personal")
	if len(c.Profiles["claude"]) != 1 {
		t.Errorf("no-op deletes should not change anything: %+v", c.Profiles["claude"])
	}
}

func TestSetProfilePath(t *testing.T) {
	c := Config{}
	_ = c.AddProfile("claude", "work", "/old/path")

	c.SetProfilePath("claude", "Work", "/new/path") // case-insensitive
	if c.Profiles["claude"][0].Path != "/new/path" {
		t.Errorf("path = %q, want /new/path", c.Profiles["claude"][0].Path)
	}

	// A no-op for a profile that doesn't exist.
	c.SetProfilePath("claude", "nonexistent", "/wherever")
	if len(c.Profiles["claude"]) != 1 {
		t.Errorf("unexpected mutation: %+v", c.Profiles["claude"])
	}
}
