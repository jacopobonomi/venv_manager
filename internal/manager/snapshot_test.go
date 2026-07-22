package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSanitizeLabel(t *testing.T) {
	cases := map[string]string{
		"hello world": "hello-world",
		"pre-upgrade": "pre-upgrade",
		"bad/chars!":  "bad-chars-",
		"":            "",
	}
	for in, want := range cases {
		if got := sanitizeLabel(in); got != want {
			t.Errorf("sanitizeLabel(%q)=%q want %q", in, got, want)
		}
	}
	long := strings.Repeat("a", 100)
	if got := sanitizeLabel(long); len(got) != 40 {
		t.Errorf("expected truncated to 40, got %d", len(got))
	}
}

func TestCountLines(t *testing.T) {
	if got := countLines([]byte("a\nb\nc\n")); got != 3 {
		t.Errorf("countLines=%d want 3", got)
	}
	if got := countLines([]byte("")); got != 0 {
		t.Errorf("countLines empty=%d want 0", got)
	}
	if got := countLines([]byte("\n\n\n")); got != 0 {
		t.Errorf("countLines blank=%d want 0", got)
	}
}

func TestListSnapshotsNoDir(t *testing.T) {
	m, dir := newTestMgr(t)
	os.MkdirAll(filepath.Join(dir, "v"), 0o755)
	snaps, err := m.ListSnapshots("v")
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots, got %v", snaps)
	}
}

func TestListSnapshotsSortsNewestFirst(t *testing.T) {
	m, dir := newTestMgr(t)
	v := filepath.Join(dir, "v")
	os.MkdirAll(filepath.Join(v, ".venv-manager", "snapshots"), 0o755)
	older := filepath.Join(v, ".venv-manager", "snapshots", "20260101-100000.txt")
	newer := filepath.Join(v, ".venv-manager", "snapshots", "20260201-100000.txt")
	os.WriteFile(older, []byte("pkg==1\n"), 0o644)
	os.WriteFile(newer, []byte("pkg==2\n"), 0o644)
	// Explicit mtimes: some CI filesystems collapse back-to-back writes to
	// identical nanosecond mtimes, which would make the sort order depend
	// on filename tie-break rather than time.
	past := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	future := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, future, future); err != nil {
		t.Fatal(err)
	}

	snaps, err := m.ListSnapshots("v")
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2, got %v", snaps)
	}
	if snaps[0].ID != "20260201-100000" {
		t.Fatalf("expected newest first: %v", snaps)
	}
}
