package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jacopobonomi/venv-manager/internal/manager"
)

func fakeVenv(t *testing.T, baseDir, name string) {
	t.Helper()
	path := filepath.Join(baseDir, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "pyvenv.cfg"), []byte("home = test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshDoesNotEnableGlobalMode(t *testing.T) {
	dir := t.TempDir()
	fakeVenv(t, dir, "one")
	mgr := manager.New(dir)
	model := New(mgr)
	msg := model.refreshCmd()()
	refreshed, ok := msg.(refreshedMsg)
	if !ok || refreshed.err != nil || len(refreshed.items) != 1 {
		t.Fatalf("unexpected refresh result: %#v", msg)
	}
	if _, err := mgr.GetSize(""); err == nil {
		t.Fatal("refresh unexpectedly enabled global mode")
	}
}

func TestModelUpdateAndViews(t *testing.T) {
	model := New(manager.New(t.TempDir()))
	if model.Init() == nil {
		t.Fatal("expected initial refresh command")
	}
	if got := model.View(); got != "loading…" {
		t.Fatalf("unexpected initial view: %q", got)
	}

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model = updated.(Model)
	updated, _ = model.Update(refreshedMsg{items: []list.Item{venvItem{name: "demo", size: 12}}})
	model = updated.(Model)
	updated, _ = model.Update(detailsMsg{name: "demo", packages: []string{"pkg==1"}, size: 12})
	model = updated.(Model)
	if view := model.View(); !strings.Contains(view, "demo") || !strings.Contains(view, "pkg==1") {
		t.Fatalf("details missing from view: %q", view)
	}
	updated, _ = model.Update(statusMsg{text: "failed", err: true})
	model = updated.(Model)
	if model.statusOk {
		t.Fatal("error status was not recorded")
	}
}

func TestItemAndHelpMethods(t *testing.T) {
	item := venvItem{name: "demo", size: 1024}
	if item.Title() != "demo" || item.FilterValue() != "demo" || !strings.Contains(item.Description(), "KB") {
		t.Fatalf("unexpected item rendering: %#v", item)
	}
	keys := newKeymap()
	if len(keys.ShortHelp()) == 0 || len(keys.FullHelp()) == 0 {
		t.Fatal("help bindings should not be empty")
	}
}

func TestModelCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test executable")
	}
	dir := t.TempDir()
	fakeVenv(t, dir, "demo")
	pip := filepath.Join(dir, "demo", "bin", "pip")
	if err := os.MkdirAll(filepath.Dir(pip), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pip, []byte(`#!/bin/sh
if [ "$1" = list ]; then printf '[{"name":"Pkg","version":"1"}]\n'; fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	model := New(manager.New(dir))

	details := model.detailsCmd("demo")().(detailsMsg)
	if details.err != nil || len(details.packages) != 1 {
		t.Fatalf("unexpected details: %#v", details)
	}
	missing := model.detailsCmd("missing")().(detailsMsg)
	if missing.err == nil {
		t.Fatal("missing details should fail")
	}
	cleaned := model.cleanCmd("demo")().(statusMsg)
	if cleaned.err {
		t.Fatalf("clean failed: %#v", cleaned)
	}
	removed := model.removeCmd("demo")().(statusMsg)
	if removed.err {
		t.Fatalf("remove failed: %#v", removed)
	}
	failed := model.removeCmd("demo")().(statusMsg)
	if !failed.err {
		t.Fatal("second remove should fail")
	}
}

func TestUpdateBranches(t *testing.T) {
	model := New(manager.New(t.TempDir()))
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	model = updated.(Model)
	updated, _ = model.Update(refreshedMsg{err: os.ErrPermission})
	model = updated.(Model)
	if model.statusOk {
		t.Fatal("refresh error should set error status")
	}
	updated, _ = model.Update(detailsMsg{err: os.ErrNotExist})
	model = updated.(Model)
	if model.statusOk || model.loading {
		t.Fatal("details error state was not applied")
	}

	updated, _ = model.Update(refreshedMsg{items: []list.Item{venvItem{name: "demo"}}})
	model = updated.(Model)
	for _, key := range []string{"r", "c", "d"} {
		updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		model = updated.(Model)
		if command == nil {
			t.Errorf("key %q did not produce a command", key)
		}
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if command == nil {
		t.Error("enter did not produce a details command")
	}
	_, quit := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if quit == nil {
		t.Fatal("quit key did not produce a command")
	}

	model.selected = "empty"
	model.details = nil
	if view := model.detailsView(); !strings.Contains(view, "none installed") {
		t.Fatalf("empty details not rendered: %q", view)
	}
}
