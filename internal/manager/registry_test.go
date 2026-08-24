package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegistryReconcilesAndPersistsMetadata(t *testing.T) {
	m, dir := newTestMgr(t)
	makeFakeVenv(t, dir, "beta")
	makeFakeVenv(t, dir, "alpha")

	entries, err := m.RegistryEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name != "alpha" || entries[1].Name != "beta" {
		t.Fatalf("unexpected registry entries: %+v", entries)
	}
	project := filepath.Join(t.TempDir(), "project")
	entry, err := m.SetRegistryMetadata("alpha", project, []string{"Data", "data", " ai "})
	if err != nil {
		t.Fatal(err)
	}
	if entry.ProjectPath != project || strings.Join(entry.Tags, ",") != "ai,data" {
		t.Fatalf("unexpected metadata: %+v", entry)
	}

	reloaded := New(dir)
	entry, err = reloaded.RegistryEntry("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ProjectPath != project || len(entry.Tags) != 2 {
		t.Fatalf("metadata was not persisted: %+v", entry)
	}

	if err := os.RemoveAll(filepath.Join(dir, "beta")); err != nil {
		t.Fatal(err)
	}
	entries, err = reloaded.RegistryEntries()
	if err != nil || len(entries) != 1 {
		t.Fatalf("stale registry entry was not removed: %+v %v", entries, err)
	}
}

func TestRegistryTracksUseRenameAndRemove(t *testing.T) {
	m, dir := newTestMgr(t)
	venv := makeFakeVenv(t, dir, "old")
	bin := filepath.Join(venv, "bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "python"), []byte("#!/bin/sh\nexit 0\n"), 0o755)
	oldTime := time.Now().UTC().Add(-48 * time.Hour)
	if err := m.touchRegistry("old", oldTime); err != nil {
		t.Fatal(err)
	}
	if _, err := m.EnsureVenv("old"); err != nil {
		t.Fatal(err)
	}
	entry, _ := m.RegistryEntry("old")
	if !entry.LastUsedAt.After(oldTime) {
		t.Fatalf("last-used time was not updated: %+v", entry)
	}
	if err := m.Rename("old", "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RegistryEntry("old"); err == nil {
		t.Fatal("old registry name still exists")
	}
	if _, err := m.RegistryEntry("new"); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("new"); err != nil {
		t.Fatal(err)
	}
	entries, err := m.RegistryEntries()
	if err != nil || len(entries) != 0 {
		t.Fatalf("registry entry survived removal: %+v %v", entries, err)
	}
}

func TestFindStalePrefersRegistryLastUsed(t *testing.T) {
	m, dir := newTestMgr(t)
	venv := makeFakeVenv(t, dir, "active")
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	os.Chtimes(venv, oldTime, oldTime)
	if err := m.touchRegistry("active", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	stale, err := m.FindStale(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("recently used venv marked stale: %+v", stale)
	}
}

func TestRegistryRejectsInvalidFilesAndNames(t *testing.T) {
	m, dir := newTestMgr(t)
	os.MkdirAll(filepath.Dir(m.registryPath()), 0o755)
	os.WriteFile(m.registryPath(), []byte(`{"version":99,"environments":{}}`), 0o644)
	if _, err := m.RegistryEntries(); err == nil {
		t.Fatal("unsupported registry version should fail")
	}
	os.WriteFile(m.registryPath(), []byte(`not-json`), 0o644)
	if _, err := m.RegistryEntries(); err == nil {
		t.Fatal("malformed registry should fail")
	}
	os.Remove(m.registryPath())
	if _, err := m.RegistryEntry("../bad"); err == nil {
		t.Fatal("invalid registry name should fail")
	}
	makeFakeVenv(t, dir, "valid")
	if _, err := m.SetRegistryMetadata("missing", "", nil); err == nil {
		t.Fatal("missing venv metadata update should fail")
	}
}
