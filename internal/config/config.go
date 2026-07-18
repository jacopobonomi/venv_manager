package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user-tunable settings.
type Config struct {
	// BaseDir where venvs live. Defaults to ~/.venvs.
	BaseDir string `json:"base_dir,omitempty"`
	// DefaultPython version (e.g. "3.12"). Empty means system default.
	DefaultPython string `json:"default_python,omitempty"`
	// UseUv: prefer `uv` over `python -m venv` / `pip` when available.
	UseUv bool `json:"use_uv,omitempty"`
	// PruneAfterDays: how many days of inactivity mark a venv as stale.
	PruneAfterDays int `json:"prune_after_days,omitempty"`
}

// Path returns the config file path (respects $XDG_CONFIG_HOME).
func Path() string {
	if p := os.Getenv("VENV_MANAGER_CONFIG"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "venv-manager", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".config", "venv-manager", "config.json")
}

// Load reads the config file, returning defaults if missing.
func Load() (*Config, error) {
	cfg := &Config{PruneAfterDays: 90}
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.PruneAfterDays == 0 {
		cfg.PruneAfterDays = 90
	}
	return cfg, nil
}

// Save writes the config to disk, creating parent dirs as needed.
func (c *Config) Save() error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
