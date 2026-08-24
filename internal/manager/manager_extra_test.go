package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateNameExtra(t *testing.T) {
	// additional cases not covered by the existing TestValidateName
	ok := []string{"Env.3", "my.env", "env_2.1"}
	for _, n := range ok {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) unexpected error: %v", n, err)
		}
	}
	bad := []string{"/abs", "has space", "has/slash", "has\\backslash"}
	for _, n := range bad {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) expected error, got nil", n)
		}
	}
}

func TestNewWithOptions_defaults(t *testing.T) {
	m := NewWithOptions(Options{})
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".venvs")
	if m.GetBaseDir() != want {
		t.Fatalf("GetBaseDir=%q want %q", m.GetBaseDir(), want)
	}
}

func TestSetFileSystem(t *testing.T) {
	m, _ := newTestMgr(t)
	fs := m.fs
	m.SetFileSystem(fs) // should not panic
}

func TestUsingUv(t *testing.T) {
	m := NewWithOptions(Options{UseUv: false})
	if m.UsingUv() {
		t.Fatal("expected UsingUv=false when UseUv option is false")
	}
}

func TestGetBaseDir(t *testing.T) {
	m, dir := newTestMgr(t)
	if m.GetBaseDir() != dir {
		t.Fatalf("GetBaseDir=%q want %q", m.GetBaseDir(), dir)
	}
}

func TestEnsureVenv(t *testing.T) {
	m, dir := newTestMgr(t)
	// missing
	if _, err := m.EnsureVenv("nope"); err == nil {
		t.Fatal("expected error for missing venv")
	}
	// present
	makeFakeVenv(t, dir, "v")
	if _, err := m.EnsureVenv("v"); err != nil {
		t.Fatalf("EnsureVenv: %v", err)
	}
}

func TestCreateInvalidName(t *testing.T) {
	m, _ := newTestMgr(t)
	if err := m.Create("../evil", ""); err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestCreateAlreadyExists(t *testing.T) {
	m, dir := newTestMgr(t)
	os.MkdirAll(filepath.Join(dir, "v"), 0o755)
	if err := m.Create("v", ""); err == nil {
		t.Fatal("expected error for existing venv")
	}
}

func TestRemoveInvalidName(t *testing.T) {
	m, _ := newTestMgr(t)
	if err := m.Remove("../evil"); err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestGetActivationCommand(t *testing.T) {
	m, dir := newTestMgr(t)
	makeFakeVenv(t, dir, "v")

	cases := map[string]string{
		"bash":       "activate",
		"zsh":        "activate",
		"fish":       "activate.fish",
		"csh":        "activate.csh",
		"cmd":        "activate.bat",
		"powershell": "Activate.ps1",
	}
	for shell, suffix := range cases {
		cmd, err := m.GetActivationCommand("v", shell)
		if err != nil {
			t.Errorf("GetActivationCommand(%q): %v", shell, err)
			continue
		}
		if !strings.Contains(cmd, suffix) {
			t.Errorf("shell %q: expected %q in %q", shell, suffix, cmd)
		}
	}
	// missing venv
	if _, err := m.GetActivationCommand("nope", "bash"); err == nil {
		t.Fatal("expected error for missing venv")
	}
}

func TestGetActivationCommandQuotesPaths(t *testing.T) {
	base := filepath.Join(t.TempDir(), "venvs with 'quotes'")
	m := New(base)
	makeFakeVenv(t, base, "v")

	posix, err := m.GetActivationCommand("v", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(posix, `'"'"'`) || !strings.HasPrefix(posix, "source '") {
		t.Fatalf("unsafe POSIX activation command: %q", posix)
	}
	pwsh, err := m.GetActivationCommand("v", "powershell")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pwsh, "''") || !strings.HasPrefix(pwsh, ". '") {
		t.Fatalf("unsafe PowerShell activation command: %q", pwsh)
	}
}

func TestEnsureVenvRejectsOrdinaryDirectory(t *testing.T) {
	m, dir := newTestMgr(t)
	os.MkdirAll(filepath.Join(dir, "ordinary"), 0o755)
	if _, err := m.EnsureVenv("ordinary"); err == nil {
		t.Fatal("ordinary directory must not be accepted as a venv")
	}
}

func TestGetSizeGlobal(t *testing.T) {
	m, dir := newTestMgr(t)
	for _, n := range []string{"a", "b"} {
		makeFakeVenv(t, dir, n)
		os.WriteFile(filepath.Join(dir, n, "f"), []byte("data"), 0o644)
	}
	m.SetGlobal(true)
	sizes, err := m.GetSize("")
	if err != nil {
		t.Fatalf("GetSize global: %v", err)
	}
	if len(sizes) != 2 {
		t.Fatalf("expected 2, got %v", sizes)
	}
}

func TestGetSizeSingle(t *testing.T) {
	m, dir := newTestMgr(t)
	makeFakeVenv(t, dir, "v")
	os.WriteFile(filepath.Join(dir, "v", "f"), []byte("hello"), 0o644)
	sizes, err := m.GetSize("v")
	if err != nil {
		t.Fatalf("GetSize single: %v", err)
	}
	if sizes["v"] == 0 {
		t.Fatal("expected non-zero size")
	}
}

func TestGetSizeMissing(t *testing.T) {
	m, _ := newTestMgr(t)
	if _, err := m.GetSize("nope"); err == nil {
		t.Fatal("expected error for missing venv")
	}
}

func TestDoctorReturnsReport(t *testing.T) {
	m, _ := newTestMgr(t)
	r := m.Doctor()
	if r == nil {
		t.Fatal("Doctor returned nil")
	}
	if r.BaseDir != m.GetBaseDir() {
		t.Fatalf("BaseDir mismatch: %q", r.BaseDir)
	}
}

func TestExportMissingVenv(t *testing.T) {
	m, _ := newTestMgr(t)
	if _, err := m.Export("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteSnapshotMissing(t *testing.T) {
	m, dir := newTestMgr(t)
	makeFakeVenv(t, dir, "v")
	if err := m.DeleteSnapshot("v", "nope"); err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestScanSummary(t *testing.T) {
	m, _ := newTestMgr(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.py"), []byte("import requests\n"), 0o644)
	rep, err := m.Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	s := rep.Summary()
	if s == "" {
		t.Fatal("expected non-empty summary")
	}
}
