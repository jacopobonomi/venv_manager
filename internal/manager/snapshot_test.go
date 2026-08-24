package manager

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestWriteUniqueSnapshotDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	id1, path1, err := writeUniqueSnapshot(dir, "same", []byte("first"))
	if err != nil {
		t.Fatal(err)
	}
	id2, path2, err := writeUniqueSnapshot(dir, "same", []byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	if id1 == id2 || path1 == path2 {
		t.Fatalf("snapshot collision was not resolved: %q %q", id1, id2)
	}
	first, _ := os.ReadFile(path1)
	if string(first) != "first" {
		t.Fatalf("first snapshot was overwritten: %q", first)
	}
}

func TestSnapshotIDAndRequirementHelpers(t *testing.T) {
	for _, id := range []string{"20260824-120000", "id_label-1"} {
		if !validSnapshotID(id) {
			t.Errorf("validSnapshotID(%q)=false", id)
		}
	}
	for _, id := range []string{"", ".", "..", "../escape", "bad id"} {
		if validSnapshotID(id) {
			t.Errorf("validSnapshotID(%q)=true", id)
		}
	}

	current := []byte("Keep_Pkg==1\nextra==2\n-e git+https://example.invalid/repo#egg=Editable\n")
	target := []byte("keep-pkg==1\neditable @ https://example.invalid/archive.whl\n")
	extras := requirementsAbsentFrom(current, target)
	if strings.Join(extras, ",") != "extra==2" {
		t.Fatalf("unexpected extras: %v", extras)
	}
}

func TestRollbackInstallFailureDoesNotUninstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test executable")
	}
	m, dir := newTestMgr(t)
	venv := makeFakeVenv(t, dir, "v")
	bin := filepath.Join(venv, "bin")
	os.MkdirAll(bin, 0o755)
	logPath := filepath.Join(t.TempDir(), "pip.log")
	pip := filepath.Join(bin, "pip")
	script := "#!/bin/sh\n" +
		"echo \"$1\" >> '" + logPath + "'\n" +
		"if [ \"$1\" = freeze ]; then echo oldpkg==1; exit 0; fi\n" +
		"if [ \"$1\" = install ]; then exit 1; fi\n" +
		"exit 0\n"
	if err := os.WriteFile(pip, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	snapshotDir := snapshotsDir(venv)
	os.MkdirAll(snapshotDir, 0o755)
	os.WriteFile(filepath.Join(snapshotDir, "snap.txt"), []byte("target==1\n"), 0o644)

	if _, err := m.Rollback("v", "snap"); err == nil {
		t.Fatal("expected rollback install failure")
	}
	logData, _ := os.ReadFile(logPath)
	if strings.Contains(string(logData), "uninstall") {
		t.Fatalf("rollback uninstalled packages after failed install: %s", logData)
	}
}

func TestDeleteSnapshotRejectsTraversal(t *testing.T) {
	m, dir := newTestMgr(t)
	makeFakeVenv(t, dir, "v")
	if err := m.DeleteSnapshot("v", "../escape"); err == nil {
		t.Fatal("expected invalid snapshot ID error")
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
	makeFakeVenv(t, dir, "v")
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
	v := makeFakeVenv(t, dir, "v")
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

func TestDiffSnapshots(t *testing.T) {
	m, dir := newTestMgr(t)
	venv := makeFakeVenv(t, dir, "v")
	dir = snapshotsDir(venv)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "before.txt"), []byte("keep==1\nremove==1\nchange==1\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "after.txt"), []byte("keep==1\nadd==2\nchange==2\n"), 0o644)

	diff, err := m.DiffSnapshots("v", "before", "after")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(diff.Added, ",") != "add==2" || strings.Join(diff.Removed, ",") != "remove==1" {
		t.Fatalf("unexpected add/remove diff: %+v", diff)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Name != "change" || diff.Changed[0].From != "change==1" || diff.Changed[0].To != "change==2" {
		t.Fatalf("unexpected changed diff: %+v", diff.Changed)
	}
}

func TestDiffSnapshotsValidation(t *testing.T) {
	m, dir := newTestMgr(t)
	makeFakeVenv(t, dir, "v")
	if _, err := m.DiffSnapshots("v", "", ""); err == nil {
		t.Fatal("missing from snapshot should fail")
	}
	if _, err := m.DiffSnapshots("v", "../bad", ""); err == nil {
		t.Fatal("invalid snapshot id should fail")
	}
	if _, err := m.DiffSnapshots("v", "missing", ""); err == nil {
		t.Fatal("missing snapshot should fail")
	}
}
