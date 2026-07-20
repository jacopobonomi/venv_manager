//go:build integration
// +build integration

// Integration tests exercise the full flow against a real Python interpreter
// and PyPI. Run with:
//
//     go test -tags=integration ./internal/manager/...
//
// Skipped in the default unit test run.

package manager

import (
	"os/exec"
	"strings"
	"testing"
)

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err := exec.LookPath("python"); err != nil {
			t.Skip("python not installed")
		}
	}
}

func TestIntegration_CreateListRemove(t *testing.T) {
	requirePython(t)
	m, _ := newTestMgr(t)
	if err := m.Create("itv", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	names, err := m.List()
	if err != nil || len(names) != 1 || names[0] != "itv" {
		t.Fatalf("List=%v err=%v", names, err)
	}
	if err := m.Remove("itv"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestIntegration_InstallAndDescribe(t *testing.T) {
	requirePython(t)
	m, _ := newTestMgr(t)
	if err := m.Create("iti", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Install a tiny wheel-only package to keep the test fast.
	pipArgs := []string{"install", "wheel==0.42.0"}
	if out, err := exec.Command(m.VenvPath("iti")+"/bin/pip", pipArgs...).CombinedOutput(); err != nil {
		t.Fatalf("pip install: %v\n%s", err, out)
	}
	desc, err := m.Describe("iti")
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if desc.PackageCount == 0 {
		t.Fatalf("expected packages, got %+v", desc)
	}
	found := false
	for _, p := range desc.Packages {
		if strings.HasPrefix(strings.ToLower(p), "wheel==") {
			found = true
		}
	}
	if !found {
		t.Fatalf("wheel not in packages: %v", desc.Packages)
	}
}

func TestIntegration_SnapshotRollback(t *testing.T) {
	requirePython(t)
	m, _ := newTestMgr(t)
	if err := m.Create("its", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	pip := m.VenvPath("its") + "/bin/pip"
	exec.Command(pip, "install", "wheel==0.42.0").Run()

	snap, err := m.CreateSnapshot("its", "clean")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if snap.PackageCount == 0 {
		t.Fatalf("expected pkgs in snapshot, got %+v", snap)
	}

	// Install another package, then rollback.
	if out, err := exec.Command(pip, "install", "six==1.16.0").CombinedOutput(); err != nil {
		t.Fatalf("install six: %v\n%s", err, out)
	}
	restored, err := m.Rollback("its", "")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if restored.ID != snap.ID {
		t.Fatalf("expected restored=%q, got %q", snap.ID, restored.ID)
	}
	pkgs, _ := m.ListPackages("its")
	for _, p := range pkgs {
		if strings.HasPrefix(strings.ToLower(p), "six==") {
			t.Fatalf("rollback did not remove 'six': %v", pkgs)
		}
	}
}
