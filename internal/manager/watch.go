package manager

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// WatchOptions configures watch mode.
type WatchOptions struct {
	// Venv name to keep in sync. Required (create it beforehand or use `create`).
	Venv string
	// Debounce delay between rapid edits.
	Debounce time.Duration
	// Log stream for status messages. If nil, discarded.
	Log io.Writer
}

// Watch monitors path (file or directory) and installs any missing imports
// into the venv whenever a .py file changes. Blocks until context cancel or
// watcher error. Ctrl+C in the CLI exits.
func (m *Manager) Watch(path string, opts WatchOptions) error {
	if opts.Venv == "" {
		return fmt.Errorf("watch requires --venv")
	}
	if _, err := m.requireVenv(opts.Venv); err != nil {
		return err
	}
	if opts.Debounce == 0 {
		opts.Debounce = 500 * time.Millisecond
	}
	logf := func(format string, a ...any) {
		if opts.Log != nil {
			fmt.Fprintf(opts.Log, format+"\n", a...)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Watch the directory containing the path so that editor-atomic renames
	// (write-to-tmp + rename) still fire events.
	dir := path
	if fi, err := statOrErr(path); err == nil && !fi.IsDir() {
		dir = filepath.Dir(path)
	}
	if err := watcher.Add(dir); err != nil {
		return err
	}

	logf("watching %s (venv: %s) — syncing imports on change. Ctrl+C to stop.", dir, opts.Venv)
	if err := m.syncImports(path, opts.Venv, logf); err != nil {
		logf("initial sync error: %v", err)
	}

	var timer *time.Timer
	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !strings.HasSuffix(ev.Name, ".py") {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(opts.Debounce, func() {
				if err := m.syncImports(path, opts.Venv, logf); err != nil {
					logf("sync error: %v", err)
				}
			})
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logf("watcher error: %v", err)
		}
	}
}

// syncImports scans and pip-installs anything missing.
func (m *Manager) syncImports(path, venv string, logf func(string, ...any)) error {
	rep, err := m.Scan(path, venv)
	if err != nil {
		return err
	}
	if len(rep.Missing) == 0 {
		logf("[%s] up to date (%d third-party imports)", time.Now().Format("15:04:05"), len(rep.ThirdParty))
		return nil
	}
	logf("[%s] installing missing: %s", time.Now().Format("15:04:05"), strings.Join(rep.Missing, ", "))
	pip := utils.PipPath(m.VenvPath(venv))
	args := append([]string{"install"}, rep.Missing...)
	out, err := exec.Command(pip, args...).CombinedOutput()
	if err != nil {
		logf("pip install failed:\n%s", string(out))
		return err
	}
	logf("[%s] ok: %d installed", time.Now().Format("15:04:05"), len(rep.Missing))
	return nil
}

func statOrErr(path string) (fileInfo, error) {
	return osStat(path)
}
