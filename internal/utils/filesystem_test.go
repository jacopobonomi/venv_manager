package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealFileSystem(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileSystem()

	sub := filepath.Join(dir, "a", "b")
	if err := fs.CreateDir(sub); err != nil {
		t.Fatalf("CreateDir: %v", err)
	}
	if !fs.Exists(sub) {
		t.Fatal("Exists should be true after CreateDir")
	}
	if !fs.IsDir(sub) {
		t.Fatal("IsDir should be true")
	}

	f := filepath.Join(sub, "file.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fs.Exists(f) {
		t.Fatal("Exists should be true for file")
	}
	if fs.IsDir(f) {
		t.Fatal("IsDir should be false for file")
	}

	entries, err := fs.ReadDir(sub)
	if err != nil || len(entries) != 1 {
		t.Fatalf("ReadDir: err=%v entries=%v", err, entries)
	}

	size, err := fs.GetDirSize(sub)
	if err != nil || size == 0 {
		t.Fatalf("GetDirSize: err=%v size=%d", err, size)
	}

	if err := fs.RemoveAll(filepath.Join(dir, "a")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if fs.Exists(sub) {
		t.Fatal("Exists should be false after RemoveAll")
	}
}

func TestExistsNonExistent(t *testing.T) {
	fs := NewFileSystem()
	if fs.Exists("/tmp/this-does-not-exist-venv-manager-test-xyz") {
		t.Fatal("expected false")
	}
}

func TestReadDirMissing(t *testing.T) {
	fs := NewFileSystem()
	_, err := fs.ReadDir("/tmp/this-does-not-exist-venv-manager-test-xyz")
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
}
