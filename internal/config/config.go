// Package config reads and writes triAGe's on-disk settings: per-agent
// config directory overrides and named profiles, for agents installed
// somewhere other than their usual default path, or run under more than
// one account (e.g. a work and a personal Claude account).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile is one named, extra config directory for an agent, beyond its
// default. "Default" is reserved (it's the implicit profile backed by
// AgentPaths) and can't be used here.
type Profile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Config is the on-disk settings file.
type Config struct {
	// AgentPaths maps a provider ID (e.g. "claude", "codex") to a config
	// directory that replaces its built-in default. This is the "Default"
	// profile's override.
	AgentPaths map[string]string `json:"agent_paths"`

	// Profiles maps a provider ID to its extra named profiles.
	Profiles map[string][]Profile `json:"profiles,omitempty"`
}

func filePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "triage", "config.json"), nil
}

// Load reads the config file. A missing file isn't an error — it just
// means no overrides have been set yet.
func Load() (Config, error) {
	c := Config{AgentPaths: map[string]string{}}

	p, err := filePath()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{AgentPaths: map[string]string{}}, err
	}
	if c.AgentPaths == nil {
		c.AgentPaths = map[string]string{}
	}
	return c, nil
}

// Save writes the config file, creating its directory if needed.
func (c Config) Save() error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// ValidateProfileName reports whether name is usable as a new profile name
// for the given agent: non-empty, not "default" (reserved for the built-in
// profile), and not already used (case-insensitive) by that agent.
func (c Config) ValidateProfileName(agentID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("profile name can't be empty")
	}
	if strings.EqualFold(name, "default") {
		return fmt.Errorf(`"default" is reserved for the built-in profile`)
	}
	for _, p := range c.Profiles[agentID] {
		if strings.EqualFold(p.Name, name) {
			return fmt.Errorf("a profile named %q already exists for this agent", name)
		}
	}
	return nil
}

// AddProfile appends a new named profile for an agent; see
// ValidateProfileName for why it might refuse the name.
func (c *Config) AddProfile(agentID, name, path string) error {
	name = strings.TrimSpace(name)
	if err := c.ValidateProfileName(agentID, name); err != nil {
		return err
	}
	if c.Profiles == nil {
		c.Profiles = map[string][]Profile{}
	}
	c.Profiles[agentID] = append(c.Profiles[agentID], Profile{Name: name, Path: path})
	return nil
}

// DeleteProfile removes a named profile for an agent, by name
// (case-insensitive). A no-op if it doesn't exist.
func (c *Config) DeleteProfile(agentID, name string) {
	profiles := c.Profiles[agentID]
	for i, p := range profiles {
		if strings.EqualFold(p.Name, name) {
			c.Profiles[agentID] = append(profiles[:i], profiles[i+1:]...)
			return
		}
	}
}

// SetProfilePath updates an existing named profile's path in place. A no-op
// if the profile doesn't exist.
func (c *Config) SetProfilePath(agentID, name, path string) {
	for i, p := range c.Profiles[agentID] {
		if strings.EqualFold(p.Name, name) {
			c.Profiles[agentID][i].Path = path
			return
		}
	}
}
