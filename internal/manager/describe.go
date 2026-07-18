package manager

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// Description is a full JSON-serializable snapshot of a venv — the single
// endpoint an AI (or script) should call to get everything it needs to reason
// about an environment.
type Description struct {
	Name          string            `json:"name"`
	Path          string            `json:"path"`
	PythonVersion string            `json:"python_version,omitempty"`
	PythonPath    string            `json:"python_path"`
	PipPath       string            `json:"pip_path"`
	Packages      []string          `json:"packages"`
	PackageCount  int               `json:"package_count"`
	SizeBytes     int64             `json:"size_bytes"`
	SizeHuman     string            `json:"size_human"`
	ModifiedAt    time.Time         `json:"modified_at"`
	FreezeHash    string            `json:"freeze_hash"`
	Activation    map[string]string `json:"activation"`
}

// Describe returns a full Description of a venv.
func (m *Manager) Describe(name string) (*Description, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	pkgs, err := m.ListPackages(name)
	if err != nil {
		return nil, err
	}
	sizes, _ := m.GetSize(name)

	verOut, _ := exec.Command(utils.PythonPath(venvPath), "-c", "import sys;print('%d.%d.%d'%sys.version_info[:3])").Output()
	ver := strings.TrimSpace(string(verOut))

	info, _ := os.Stat(venvPath)
	var mtime time.Time
	if info != nil {
		mtime = info.ModTime()
	}

	h := sha256.Sum256([]byte(strings.Join(pkgs, "\n")))

	activation := map[string]string{}
	for _, sh := range []string{"bash", "zsh", "fish", "pwsh", "cmd"} {
		if cmdStr, err := m.GetActivationCommand(name, sh); err == nil {
			activation[sh] = cmdStr
		}
	}

	return &Description{
		Name:          name,
		Path:          venvPath,
		PythonVersion: ver,
		PythonPath:    utils.PythonPath(venvPath),
		PipPath:       utils.PipPath(venvPath),
		Packages:      pkgs,
		PackageCount:  len(pkgs),
		SizeBytes:     sizes[name],
		SizeHuman:     utils.FormatSize(sizes[name]),
		ModifiedAt:    mtime,
		FreezeHash:    hex.EncodeToString(h[:]),
		Activation:    activation,
	}, nil
}
