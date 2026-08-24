package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jacopobonomi/venv-manager/internal/config"
	"github.com/jacopobonomi/venv-manager/internal/manager"
	"github.com/jacopobonomi/venv-manager/internal/utils"
)

func TestSingleOrGlobalArgs(t *testing.T) {
	oldGlobal := globalFlag
	t.Cleanup(func() { globalFlag = oldGlobal })

	globalFlag = false
	if err := singleOrGlobalArgs(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := singleOrGlobalArgs(nil, []string{"one"}); err != nil {
		t.Fatal(err)
	}
	if err := singleOrGlobalArgs(nil, []string{"one", "two"}); err == nil {
		t.Fatal("expected too-many-arguments error")
	}
	globalFlag = true
	if err := singleOrGlobalArgs(nil, []string{"one"}); err == nil {
		t.Fatal("expected --global/name conflict")
	}
}

func writeTestExecutable(t *testing.T, path, script string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setupCLIManager(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	dir := t.TempDir()
	venv := filepath.Join(dir, "demo")
	os.MkdirAll(filepath.Join(venv, "bin"), 0o755)
	os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("home = test\n"), 0o644)
	pip := `#!/bin/sh
if [ "$1" = list ] && [ "$2" = --format=json ]; then printf '[{"name":"Demo","version":"1.0"}]\n'; exit 0; fi
if [ "$1" = list ] && [ "$2" = --outdated ]; then printf '[]\n'; exit 0; fi
if [ "$1" = freeze ]; then printf 'Demo==1.0\n'; exit 0; fi
if [ "$1" = show ]; then printf 'Name: Demo\nRequires:\n'; exit 0; fi
exit 0
`
	writeTestExecutable(t, utils.PipPath(venv), pip)
	writeTestExecutable(t, utils.PythonPath(venv), "#!/bin/sh\nprintf '3.12.1\\n'\n")
	writeTestExecutable(t, utils.VenvExe(venv, "hello"), "#!/bin/sh\nexit 0\n")
	mgr = manager.New(dir)
	cfg = &config.Config{BaseDir: dir, PruneAfterDays: 90}
	return dir
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	fn()
	writer.Close()
	os.Stdout = old
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	return string(data)
}

func TestReadOnlyCLICommands(t *testing.T) {
	setupCLIManager(t)
	oldJSON, oldGlobal := jsonFlag, globalFlag
	t.Cleanup(func() { jsonFlag, globalFlag = oldJSON, oldGlobal })
	globalFlag = false

	out := captureStdout(t, func() {
		listCmd().Run(nil, nil)
		packagesCmd().Run(nil, []string{"demo"})
		sizeCmd().Run(nil, []string{"demo"})
		doctorCmd().Run(nil, nil)
		activateCmd().Run(nil, []string{"demo"})
		deactivateCmd().Run(nil, nil)
		exportCmd().Run(nil, []string{"demo"})
		describeCmd().Run(nil, []string{"demo"})
		whyCmd().Run(nil, []string{"demo", "demo"})
		snapshotsCmd().Run(nil, []string{"demo"})
	})
	for _, expected := range []string{"Available virtual environments", "Demo==1.0", "venv-manager doctor", "source '", `"python_version"`} {
		if !strings.Contains(out, expected) {
			t.Errorf("missing %q in output", expected)
		}
	}

	jsonFlag = true
	jsonOut := captureStdout(t, func() {
		listCmd().Run(nil, nil)
		packagesCmd().Run(nil, []string{"demo"})
		doctorCmd().Run(nil, nil)
	})
	if !strings.Contains(jsonOut, `"demo"`) || !strings.Contains(jsonOut, `"base_dir"`) {
		t.Fatalf("unexpected JSON output: %s", jsonOut)
	}
}

func TestMutatingCLICommands(t *testing.T) {
	dir := setupCLIManager(t)
	oldGlobal := globalFlag
	t.Cleanup(func() { globalFlag = oldGlobal })
	globalFlag = false
	requirements := filepath.Join(t.TempDir(), "requirements.txt")
	os.WriteFile(requirements, []byte("Demo==1.0\n"), 0o644)

	captureStdout(t, func() {
		installCmd().Run(nil, []string{"demo", requirements})
		upgradeCmd().Run(nil, []string{"demo"})
		cleanCmd().Run(nil, []string{"demo"})
		runCmd().Run(nil, []string{"demo", "hello"})
		snapshotCmd().Run(nil, []string{"demo"})
		rollbackCmd().Run(nil, []string{"demo"})
		renameCmd().Run(nil, []string{"demo", "renamed"})
		removeCmd().Run(nil, []string{"renamed"})
	})
	if _, err := os.Stat(filepath.Join(dir, "renamed")); !os.IsNotExist(err) {
		t.Fatalf("renamed venv should have been removed: %v", err)
	}
}

func TestScanPruneConfigAndCompletionCommands(t *testing.T) {
	dir := setupCLIManager(t)
	pyfile := filepath.Join(t.TempDir(), "app.py")
	os.WriteFile(pyfile, []byte("import requests\n"), 0o644)
	oldJSON := jsonFlag
	t.Cleanup(func() { jsonFlag = oldJSON })
	jsonFlag = false
	prune := pruneCmd()
	prune.Flags().Set("dry-run", "true")
	prune.Flags().Set("days", "1")
	past := time.Now().AddDate(0, 0, -10)
	os.Chtimes(filepath.Join(dir, "demo"), past, past)

	out := captureStdout(t, func() {
		scanCmd().Run(nil, []string{pyfile})
		prune.Run(nil, nil)
		configCmd().Commands()[1].Run(nil, nil)
		completionCmd().Run(completionCmd(), []string{"bash"})
	})
	if !strings.Contains(out, "requests") || !strings.Contains(out, "Stale venvs") || !strings.Contains(out, "bash completion") {
		t.Fatalf("unexpected command output: %s", out)
	}
}

func TestCreateExecAndImportCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX test executables")
	}
	binDir := t.TempDir()
	writeTestExecutable(t, filepath.Join(binDir, "python3"), `#!/bin/sh
for arg in "$@"; do target="$arg"; done
mkdir -p "$target/bin"
printf 'home = test\n' > "$target/pyvenv.cfg"
exit 0
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := t.TempDir()
	mgr = manager.New(dir)
	cfg = &config.Config{BaseDir: dir, PruneAfterDays: 90}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	os.WriteFile(manifestPath, []byte(`{"name":"imported","requirements":[]}`), 0o644)

	captureStdout(t, func() {
		createCmd().Run(nil, []string{"created"})
		execCmd().Run(nil, []string{"true"})
		importCmd().Run(nil, []string{manifestPath})
	})
	for _, name := range []string{"created", "imported"} {
		if _, err := os.Stat(filepath.Join(dir, name, "pyvenv.cfg")); err != nil {
			t.Fatalf("%s was not created: %v", name, err)
		}
	}
}

func TestRegistryAndSnapshotDiffCommands(t *testing.T) {
	dir := setupCLIManager(t)
	oldJSON := jsonFlag
	t.Cleanup(func() { jsonFlag = oldJSON })
	jsonFlag = false
	venv := filepath.Join(dir, "demo")
	snapshotDir := filepath.Join(venv, ".venv-manager", "snapshots")
	os.MkdirAll(snapshotDir, 0o755)
	os.WriteFile(filepath.Join(snapshotDir, "before.txt"), []byte("one==1\n"), 0o644)
	os.WriteFile(filepath.Join(snapshotDir, "after.txt"), []byte("one==2\ntwo==1\n"), 0o644)

	out := captureStdout(t, func() {
		registryCmd().Run(nil, nil)
		snapshotDiffCmd().Run(nil, []string{"demo", "before", "after"})
	})
	if !strings.Contains(out, "demo") || !strings.Contains(out, "+ two==1") || !strings.Contains(out, "one==1") {
		t.Fatalf("unexpected registry/diff output: %s", out)
	}

	jsonFlag = true
	jsonOut := captureStdout(t, func() {
		registryCmd().Run(nil, []string{"demo"})
		snapshotDiffCmd().Run(nil, []string{"demo", "before", "after"})
	})
	if !strings.Contains(jsonOut, `"last_used_at"`) || !strings.Contains(jsonOut, `"changed"`) {
		t.Fatalf("unexpected JSON output: %s", jsonOut)
	}
}
