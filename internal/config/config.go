// Package config reads and writes triAGe's on-disk settings: currently just
// per-agent config directory overrides, for agents installed somewhere
// other than their usual default path.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the on-disk settings file.
type Config struct {
	// AgentPaths maps a provider ID (e.g. "claude", "codex") to a config
	// directory that replaces its built-in default.
	AgentPaths map[string]string `json:"agent_paths"`
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
