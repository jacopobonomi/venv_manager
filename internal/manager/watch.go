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
	// (write-to-tmp + rename) still fire events. If path doesn't exist yet,
	// treat it as a file whose directory should exist — watch the parent.
	dir := path
	fi, statErr := statOrErr(path)
	switch {
	case statErr == nil && fi.IsDir():
		// watch the directory tree itself
	case statErr == nil:
		dir = filepath.Dir(path)
	default:
		dir = filepath.Dir(path)
		if dir == "" || dir == "." {
			dir, _ = filepath.Abs(".")
		}
	}
	// fsnotify is not recursive: register every subdirectory explicitly, and
	// keep registering new ones as they appear (Create events below).
	if err := addWatchTree(watcher, dir); err != nil {
		return fmt.Errorf("cannot watch %q: %v", dir, err)
	}

	logf("watching %s (venv: %s) — syncing imports on change. Ctrl+C to stop.", dir, opts.Venv)
	// Skip initial sync if the target file doesn't exist yet; the first
	// file-create event will trigger a sync.
	if statErr == nil {
		if err := m.syncImports(path, opts.Venv, logf); err != nil {
			logf("initial sync error: %v", err)
		}
	}

	var timer *time.Timer
	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := osStat(ev.Name); err == nil && fi.IsDir() {
					if !skipWatchDir(filepath.Base(ev.Name)) {
						if err := addWatchTree(watcher, ev.Name); err != nil {
							logf("cannot watch new dir %q: %v", ev.Name, err)
						}
					}
					continue
				}
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

// skipWatchDir mirrors the directories Scan already ignores.
func skipWatchDir(name string) bool {
	switch name {
	case ".venv", "venv", ".git", "__pycache__", "node_modules", "site-packages":
		return true
	}
	return false
}

// addWatchTree registers dir and all its non-ignored subdirectories.
func addWatchTree(w *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(p string, fi fileInfo, err error) error {
		if err != nil {
			// The root must be watchable; children that vanish mid-walk are fine.
			if p == dir {
				return err
			}
			return nil
		}
		if !fi.IsDir() {
			return nil
		}
		if p != dir && skipWatchDir(fi.Name()) {
			return filepath.SkipDir
		}
		return w.Add(p)
	})
}
