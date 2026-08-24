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

// PackageChange describes a version or source change between two snapshots.
type PackageChange struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

// SnapshotDiff is the package-level difference between two saved states, or
// between a saved state and the current environment.
type SnapshotDiff struct {
	Venv    string          `json:"venv"`
	From    string          `json:"from"`
	To      string          `json:"to"`
	Added   []string        `json:"added"`
	Removed []string        `json:"removed"`
	Changed []PackageChange `json:"changed"`
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
	snapshotID, fpath, err := writeUniqueSnapshot(dir, base, out)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		ID:           snapshotID,
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
	// Sort newest-first; fall back to ID (which embeds the timestamp) when
	// mtimes tie — filesystems on fast CI runners can produce identical
	// nanosecond mtimes for files written back-to-back.
	sort.Slice(snaps, func(i, j int) bool {
		if !snaps[i].CreatedAt.Equal(snaps[j].CreatedAt) {
			return snaps[i].CreatedAt.After(snaps[j].CreatedAt)
		}
		return snaps[i].ID > snaps[j].ID
	})
	return snaps, nil
}

// Rollback restores a venv to a snapshot. If snapshotID is empty, it uses the
// most recent snapshot. It installs the snapshot first, then removes packages
// that are not part of it. Returns the snapshot that was restored.
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
	// Capture the current state, install the target first, then remove only
	// packages that are absent from the snapshot. A failed install therefore
	// does not proactively empty the environment before reporting the error.
	cur, err := exec.Command(pip, "freeze").Output()
	if err != nil {
		return nil, fmt.Errorf("pip freeze failed: %v", err)
	}
	if out, err := exec.Command(pip, "install", "-r", target.Path).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("install from snapshot failed; existing packages were not removed: %v\n%s", err, out)
	}

	targetData, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	extras := requirementsAbsentFrom(cur, targetData)
	if len(extras) > 0 {
		tmp, err := os.CreateTemp("", "vm-uninstall-*.txt")
		if err != nil {
			return nil, err
		}
		if _, err := tmp.WriteString(strings.Join(extras, "\n") + "\n"); err != nil {
			tmp.Close()
			return nil, err
		}
		if err := tmp.Close(); err != nil {
			return nil, err
		}
		defer os.Remove(tmp.Name())
		if out, err := exec.Command(pip, "uninstall", "-y", "-r", tmp.Name()).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("uninstall failed: %v\n%s", err, out)
		}
	}
	return target, nil
}

// DeleteSnapshot removes a snapshot file.
func (m *Manager) DeleteSnapshot(name, snapshotID string) error {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return err
	}
	if !validSnapshotID(snapshotID) {
		return fmt.Errorf("invalid snapshot id %q", snapshotID)
	}
	p := filepath.Join(snapshotsDir(venvPath), snapshotID+".txt")
	if !m.fs.Exists(p) {
		return fmt.Errorf("snapshot %q not found", snapshotID)
	}
	return os.Remove(p)
}

// DiffSnapshots compares fromID with toID. An empty toID compares the saved
// snapshot with the environment's current pip-freeze state.
func (m *Manager) DiffSnapshots(name, fromID, toID string) (*SnapshotDiff, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	if fromID == "" {
		return nil, fmt.Errorf("from snapshot id is required")
	}
	fromData, err := m.snapshotData(name, fromID)
	if err != nil {
		return nil, err
	}
	toLabel := toID
	var toData []byte
	if toID == "" {
		toLabel = "current"
		toData, err = exec.Command(utils.PipPath(venvPath), "freeze").Output()
		if err != nil {
			return nil, fmt.Errorf("pip freeze failed: %v", err)
		}
	} else {
		toData, err = m.snapshotData(name, toID)
		if err != nil {
			return nil, err
		}
	}

	fromPackages := requirementSpecs(fromData)
	toPackages := requirementSpecs(toData)
	diff := &SnapshotDiff{
		Venv:    name,
		From:    fromID,
		To:      toLabel,
		Added:   []string{},
		Removed: []string{},
		Changed: []PackageChange{},
	}
	for packageName, toSpec := range toPackages {
		fromSpec, existed := fromPackages[packageName]
		switch {
		case !existed:
			diff.Added = append(diff.Added, toSpec)
		case fromSpec != toSpec:
			diff.Changed = append(diff.Changed, PackageChange{Name: packageName, From: fromSpec, To: toSpec})
		}
	}
	for packageName, fromSpec := range fromPackages {
		if _, exists := toPackages[packageName]; !exists {
			diff.Removed = append(diff.Removed, fromSpec)
		}
	}
	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Slice(diff.Changed, func(i, j int) bool { return diff.Changed[i].Name < diff.Changed[j].Name })
	return diff, nil
}

func (m *Manager) snapshotData(name, snapshotID string) ([]byte, error) {
	if !validSnapshotID(snapshotID) {
		return nil, fmt.Errorf("invalid snapshot id %q", snapshotID)
	}
	snapshots, err := m.ListSnapshots(name)
	if err != nil {
		return nil, err
	}
	for _, snapshot := range snapshots {
		if snapshot.ID == snapshotID {
			return os.ReadFile(snapshot.Path)
		}
	}
	return nil, fmt.Errorf("snapshot %q not found for venv %q", snapshotID, name)
}

func requirementSpecs(data []byte) map[string]string {
	specs := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if name := requirementName(line); name != "" {
			specs[name] = line
		}
	}
	return specs
}

func writeUniqueSnapshot(dir, base string, data []byte) (string, string, error) {
	for suffix := 0; ; suffix++ {
		id := base
		if suffix > 0 {
			id = fmt.Sprintf("%s-%d", base, suffix)
		}
		path := filepath.Join(dir, id+".txt")
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", "", err
		}
		if _, err := file.Write(data); err != nil {
			file.Close()
			os.Remove(path)
			return "", "", err
		}
		if err := file.Close(); err != nil {
			os.Remove(path)
			return "", "", err
		}
		return id, path, nil
	}
}

func validSnapshotID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func requirementsAbsentFrom(current, target []byte) []string {
	targetNames := requirementNames(target)
	var extras []string
	for _, line := range strings.Split(strings.TrimSpace(string(current)), "\n") {
		name := requirementName(line)
		if name != "" && !targetNames[name] {
			extras = append(extras, line)
		}
	}
	return extras
}

func requirementNames(data []byte) map[string]bool {
	names := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if name := requirementName(line); name != "" {
			names[name] = true
		}
	}
	return names
}

func requirementName(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "--") {
		return ""
	}
	if idx := strings.LastIndex(line, "#egg="); idx >= 0 {
		return normalizePkgName(line[idx+5:])
	}
	line = strings.TrimPrefix(line, "-e ")
	if idx := strings.Index(line, " @ "); idx >= 0 {
		line = line[:idx]
	}
	if idx := strings.IndexAny(line, "=<>!~["); idx >= 0 {
		line = line[:idx]
	}
	return normalizePkgName(strings.TrimSpace(line))
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
