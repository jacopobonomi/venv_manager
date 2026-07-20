package manager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// Snapshot is a captured pip-freeze state of a venv.
type Snapshot struct {
	ID           string    `json:"id"`
	Label        string    `json:"label,omitempty"`
	Venv         string    `json:"venv"`
	CreatedAt    time.Time `json:"created_at"`
	PackageCount int       `json:"package_count"`
	Path         string    `json:"path"`
}

func snapshotsDir(venvPath string) string {
	return filepath.Join(venvPath, ".venv-manager", "snapshots")
}

// CreateSnapshot captures the current pip freeze output and stores it under the venv.
func (m *Manager) CreateSnapshot(name, label string) (*Snapshot, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	out, err := exec.Command(utils.PipPath(venvPath), "freeze").Output()
	if err != nil {
		return nil, fmt.Errorf("pip freeze failed: %v", err)
	}
	dir := snapshotsDir(venvPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := now.Format("20060102-150405")
	base := id
	if label != "" {
		base = id + "_" + sanitizeLabel(label)
	}
	fpath := filepath.Join(dir, base+".txt")
	if err := os.WriteFile(fpath, out, 0o644); err != nil {
		return nil, err
	}
	return &Snapshot{
		ID:           base,
		Label:        label,
		Venv:         name,
		CreatedAt:    now,
		PackageCount: countLines(out),
		Path:         fpath,
	}, nil
}

// ListSnapshots returns all snapshots for a venv, newest first.
func (m *Manager) ListSnapshots(name string) ([]Snapshot, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	dir := snapshotsDir(venvPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var snaps []Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".txt")
		label := ""
		if idx := strings.Index(id, "_"); idx >= 0 {
			label = id[idx+1:]
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		snaps = append(snaps, Snapshot{
			ID:           id,
			Label:        label,
			Venv:         name,
			CreatedAt:    info.ModTime().UTC(),
			PackageCount: countLines(data),
			Path:         filepath.Join(dir, e.Name()),
		})
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].CreatedAt.After(snaps[j].CreatedAt) })
	return snaps, nil
}

// Rollback restores a venv to a snapshot. If snapshotID is empty, uses the
// most recent snapshot. Uninstalls current packages, then reinstalls from
// snapshot file. Returns the snapshot that was restored.
func (m *Manager) Rollback(name, snapshotID string) (*Snapshot, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	snaps, err := m.ListSnapshots(name)
	if err != nil {
		return nil, err
	}
	if len(snaps) == 0 {
		return nil, fmt.Errorf("no snapshots for venv %q", name)
	}
	var target *Snapshot
	if snapshotID == "" {
		target = &snaps[0]
	} else {
		for i := range snaps {
			if snaps[i].ID == snapshotID {
				target = &snaps[i]
				break
			}
		}
		if target == nil {
			return nil, fmt.Errorf("snapshot %q not found for venv %q", snapshotID, name)
		}
	}

	pip := utils.PipPath(venvPath)
	// Freeze current, uninstall everything, then install snapshot.
	cur, err := exec.Command(pip, "freeze").Output()
	if err != nil {
		return nil, fmt.Errorf("pip freeze failed: %v", err)
	}
	if len(strings.TrimSpace(string(cur))) > 0 {
		tmp, err := os.CreateTemp("", "vm-uninstall-*.txt")
		if err != nil {
			return nil, err
		}
		tmp.Write(cur)
		tmp.Close()
		defer os.Remove(tmp.Name())
		if out, err := exec.Command(pip, "uninstall", "-y", "-r", tmp.Name()).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("uninstall failed: %v\n%s", err, out)
		}
	}
	if out, err := exec.Command(pip, "install", "-r", target.Path).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("install from snapshot failed: %v\n%s", err, out)
	}
	return target, nil
}

// DeleteSnapshot removes a snapshot file.
func (m *Manager) DeleteSnapshot(name, snapshotID string) error {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return err
	}
	p := filepath.Join(snapshotsDir(venvPath), snapshotID+".txt")
	if !m.fs.Exists(p) {
		return fmt.Errorf("snapshot %q not found", snapshotID)
	}
	return os.Remove(p)
}

func sanitizeLabel(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func countLines(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	n := 0
	for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if strings.TrimSpace(ln) != "" {
			n++
		}
	}
	return n
}
