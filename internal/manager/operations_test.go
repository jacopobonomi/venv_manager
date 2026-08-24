package manager

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

func writeExecutable(t *testing.T, path, script string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func fakeOperationalVenv(t *testing.T) (*Manager, string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	m, dir := newTestMgr(t)
	venv := makeFakeVenv(t, dir, "v")
	logPath := filepath.Join(t.TempDir(), "commands.log")
	pipScript := `#!/bin/sh
echo "$*" >> "$FAKE_COMMAND_LOG"
if [ "$1" = "list" ] && [ "$2" = "--format=json" ]; then
  printf '[{"name":"Demo","version":"1.2"}]\n'
  exit 0
fi
if [ "$1" = "list" ] && [ "$2" = "--outdated" ]; then
  printf '[{"name":"Demo"}]\n'
  exit 0
fi
if [ "$1" = "freeze" ]; then
  printf 'Demo==1.2\n'
  exit 0
fi
if [ "$1" = "show" ]; then
  printf 'Name: Demo\nRequires:\n'
  exit 0
fi
exit 0
`
	writeExecutable(t, utils.PipPath(venv), pipScript)
	writeExecutable(t, utils.PythonPath(venv), "#!/bin/sh\nprintf '3.12.1\\n'\n")
	writeExecutable(t, utils.VenvExe(venv, "hello"), "#!/bin/sh\nexit 0\n")
	t.Setenv("FAKE_COMMAND_LOG", logPath)
	return m, venv, logPath
}

func TestOperationalManagerMethods(t *testing.T) {
	m, venv, logPath := fakeOperationalVenv(t)

	packages, err := m.ListPackages("v")
	if err != nil || len(packages) != 1 || packages[0] != "Demo==1.2" {
		t.Fatalf("ListPackages: %v %v", packages, err)
	}
	requirements := filepath.Join(t.TempDir(), "requirements.txt")
	os.WriteFile(requirements, []byte("Demo==1.2\n"), 0o644)
	if err := m.Install("v", requirements); err != nil {
		t.Fatal(err)
	}
	if err := m.Upgrade("v"); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(venv, "lib", "__pycache__")
	os.MkdirAll(cacheDir, 0o755)
	if err := m.Clean("v"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("pycache was not removed: %v", err)
	}
	if err := m.Run("v", []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := m.Export("v")
	if err != nil || manifest.PythonVersion != "3.12.1" {
		t.Fatalf("Export: %+v %v", manifest, err)
	}
	description, err := m.Describe("v")
	if err != nil || description.PackageCount != 1 || description.PythonVersion != "3.12.1" {
		t.Fatalf("Describe: %+v %v", description, err)
	}
	why, err := m.Why("v", "demo")
	if err != nil || !why.Direct {
		t.Fatalf("Why: %+v %v", why, err)
	}
	snapshot, err := m.CreateSnapshot("v", "test")
	if err != nil || snapshot.PackageCount != 1 {
		t.Fatalf("CreateSnapshot: %+v %v", snapshot, err)
	}
	if _, err := m.Rollback("v", snapshot.ID); err != nil {
		t.Fatal(err)
	}
	diff, err := m.DiffSnapshots("v", snapshot.ID, "")
	if err != nil || diff.To != "current" || len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Changed) != 0 {
		t.Fatalf("DiffSnapshots current: %+v %v", diff, err)
	}
	if err := m.DeleteSnapshot("v", snapshot.ID); err != nil {
		t.Fatal(err)
	}
	logData, _ := os.ReadFile(logPath)
	for _, command := range []string{"install -r", "list --outdated", "cache purge", "freeze", "show demo"} {
		if !strings.Contains(string(logData), command) {
			t.Errorf("missing command %q in log:\n%s", command, logData)
		}
	}
}

func TestRenameUsesVenvInterpreter(t *testing.T) {
	m, venv, _ := fakeOperationalVenv(t)
	if err := m.Rename("v", "renamed"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(venv), "renamed", "pyvenv.cfg")); err != nil {
		t.Fatal(err)
	}
}

func TestCreateExecAndImportWithFakePython(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	binDir := t.TempDir()
	python := filepath.Join(binDir, "python3")
	writeExecutable(t, python, `#!/bin/sh
for arg in "$@"; do target="$arg"; done
mkdir -p "$target/bin"
printf 'home = test\n' > "$target/pyvenv.cfg"
exit 0
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	m, dir := newTestMgr(t)
	if err := m.Create("created", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RegistryEntry("created"); err != nil {
		t.Fatalf("created venv was not registered: %v", err)
	}
	if err := m.Exec(ExecOptions{}, []string{"true"}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "eph-") {
			t.Fatalf("ephemeral venv was not removed: %s", entry.Name())
		}
	}
	if err := m.Import(&Manifest{Name: "imported"}); err != nil {
		t.Fatal(err)
	}
}
