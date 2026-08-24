package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathDefault(t *testing.T) {
	// Without any env override the path should land under home/.config
	t.Setenv("VENV_MANAGER_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	p := Path()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "venv-manager", "config.json")
	if p != expected {
		t.Fatalf("Path()=%q want %q", p, expected)
	}
}

func TestLoadMalformed(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(f, []byte("{not json"), 0o644)
	t.Setenv("VENV_MANAGER_CONFIG", f)
	_, err := Load()
	if err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestSaveCreatesParentDirs(t *testing.T) {
	deep := filepath.Join(t.TempDir(), "deep", "nested", "config.json")
	t.Setenv("VENV_MANAGER_CONFIG", deep)
	c := &Config{BaseDir: "/tmp/vs", PruneAfterDays: 7}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(deep); err != nil {
		t.Fatalf("expected file at %s: %v", deep, err)
	}
}

func TestLoadDefaultPruneAfterDays(t *testing.T) {
	// A config with PruneAfterDays=0 should fall back to 90.
	f := filepath.Join(t.TempDir(), "zero.json")
	os.WriteFile(f, []byte(`{"prune_after_days":0}`), 0o644)
	t.Setenv("VENV_MANAGER_CONFIG", f)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PruneAfterDays != 90 {
		t.Fatalf("expected 90, got %d", cfg.PruneAfterDays)
	}
}

func TestLoadRejectsNegativePruneAfterDays(t *testing.T) {
	f := filepath.Join(t.TempDir(), "negative.json")
	os.WriteFile(f, []byte(`{"prune_after_days":-1}`), 0o644)
	t.Setenv("VENV_MANAGER_CONFIG", f)
	if _, err := Load(); err == nil {
		t.Fatal("expected negative prune_after_days to be rejected")
	}
}
