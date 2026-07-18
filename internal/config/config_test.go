package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	t.Setenv("VENV_MANAGER_CONFIG", filepath.Join(t.TempDir(), "nope.json"))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PruneAfterDays != 90 {
		t.Fatalf("expected default 90, got %d", cfg.PruneAfterDays)
	}
}

func TestSaveAndLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg", "config.json")
	t.Setenv("VENV_MANAGER_CONFIG", p)
	c := &Config{BaseDir: "/tmp/vs", DefaultPython: "3.12", UseUv: true, PruneAfterDays: 30}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected file, got %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseDir != "/tmp/vs" || got.DefaultPython != "3.12" || !got.UseUv || got.PruneAfterDays != 30 {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestPathXDG(t *testing.T) {
	t.Setenv("VENV_MANAGER_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/opt/xdg")
	if got := Path(); got != "/opt/xdg/venv-manager/config.json" {
		t.Fatalf("Path=%q", got)
	}
}
