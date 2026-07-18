package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestMgr(t *testing.T) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	return New(dir), dir
}

func TestListEmpty(t *testing.T) {
	m, _ := newTestMgr(t)
	got, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestListSorted(t *testing.T) {
	m, dir := newTestMgr(t)
	for _, n := range []string{"zeta", "alpha", "mike"} {
		if err := os.MkdirAll(filepath.Join(dir, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "mike", "zeta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRemoveMissing(t *testing.T) {
	m, _ := newTestMgr(t)
	err := m.Remove("nope")
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected does-not-exist error, got %v", err)
	}
}

func TestRemoveOK(t *testing.T) {
	m, dir := newTestMgr(t)
	p := filepath.Join(dir, "v")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("v"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("expected removed, got %v", err)
	}
}

func TestVenvPath(t *testing.T) {
	m, dir := newTestMgr(t)
	if got, want := m.VenvPath("foo"), filepath.Join(dir, "foo"); got != want {
		t.Fatalf("VenvPath=%q want %q", got, want)
	}
}

func TestResolveTargetsNoNameNoGlobal(t *testing.T) {
	m, _ := newTestMgr(t)
	if _, err := m.resolveTargets(""); err == nil {
		t.Fatal("expected error when no name and not global")
	}
}

func TestResolveTargetsGlobal(t *testing.T) {
	m, dir := newTestMgr(t)
	for _, n := range []string{"a", "b"} {
		os.MkdirAll(filepath.Join(dir, n), 0o755)
	}
	m.SetGlobal(true)
	got, err := m.resolveTargets("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 targets, got %v", got)
	}
}

func TestGetSizeRequiresNameOrGlobal(t *testing.T) {
	m, _ := newTestMgr(t)
	if _, err := m.GetSize(""); err == nil {
		t.Fatal("expected error without name and without global")
	}
}

func TestFindStale(t *testing.T) {
	m, dir := newTestMgr(t)
	old := filepath.Join(dir, "old")
	fresh := filepath.Join(dir, "fresh")
	os.MkdirAll(old, 0o755)
	os.MkdirAll(fresh, 0o755)
	past := time.Now().AddDate(0, 0, -100)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	stale, err := m.FindStale(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].Name != "old" {
		t.Fatalf("expected [old], got %v", stale)
	}
}
