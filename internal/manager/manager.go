package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// Manager encapsulates venv operations against a base directory.
type Manager struct {
	baseDir       string
	defaultPython string
	useUv         bool
	fs            utils.FileSystem
	global        bool
}

// Options configures Manager construction.
type Options struct {
	BaseDir       string
	DefaultPython string
	UseUv         bool
}

// New constructs a Manager. Empty BaseDir defaults to ~/.venvs.
func New(baseDir string) *Manager {
	return NewWithOptions(Options{BaseDir: baseDir})
}

// NewWithOptions is the full constructor.
func NewWithOptions(opts Options) *Manager {
	if opts.BaseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		opts.BaseDir = filepath.Join(homeDir, ".venvs")
	}
	return &Manager{
		baseDir:       opts.BaseDir,
		defaultPython: opts.DefaultPython,
		useUv:         opts.UseUv && uvAvailable(),
		fs:            utils.NewFileSystem(),
	}
}

// SetFileSystem swaps the filesystem implementation (used in tests).
func (m *Manager) SetFileSystem(fs utils.FileSystem) { m.fs = fs }

func (m *Manager) SetGlobal(global bool) { m.global = global }
func (m *Manager) GetBaseDir() string    { return m.baseDir }
func (m *Manager) UsingUv() bool         { return m.useUv }

// VenvPath returns the absolute path to a named venv.
func (m *Manager) VenvPath(name string) string { return filepath.Join(m.baseDir, name) }

// validNameRe restricts venv names to a safe charset: no path separators, no
// shell metacharacters, nothing that can escape baseDir once joined.
var validNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidateName rejects venv names that are empty, reserved, or could resolve
// outside the base directory (e.g. "..", "../x", absolute paths). Every
// operation that touches the filesystem by name must pass through this —
// especially MCP tool calls, whose arguments come from an LLM.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("venv name is required")
	}
	if !validNameRe.MatchString(name) || name == "." || name == ".." {
		return fmt.Errorf("invalid venv name %q: only letters, digits, '.', '_' and '-' are allowed", name)
	}
	return nil
}

func (m *Manager) requireVenv(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	p := m.VenvPath(name)
	if !m.fs.Exists(p) {
		return "", fmt.Errorf("venv '%s' does not exist", name)
	}
	return p, nil
}

// EnsureVenv validates the name and returns the venv path, erroring when the
// venv does not exist. Exported for callers outside the package (MCP layer).
func (m *Manager) EnsureVenv(name string) (string, error) {
	return m.requireVenv(name)
}

// resolveTargets returns venv absolute paths for a single name, or all when global.
func (m *Manager) resolveTargets(name string) ([]string, error) {
	if m.global {
		venvs, err := m.List()
		if err != nil {
			return nil, err
		}
		out := make([]string, len(venvs))
		for i, v := range venvs {
			out[i] = m.VenvPath(v)
		}
		return out, nil
	}
	if name == "" {
		return nil, fmt.Errorf("please specify a venv name or use --global flag")
	}
	p, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	return []string{p}, nil
}

// Create makes a new venv. Uses uv when enabled+available, else python -m venv.
func (m *Manager) Create(name, pythonVersion string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	venvPath := m.VenvPath(name)
	if m.fs.Exists(venvPath) {
		return fmt.Errorf("'%s' already exists", name)
	}
	if err := m.fs.CreateDir(m.baseDir); err != nil {
		return fmt.Errorf("failed to create base directory: %v", err)
	}

	if pythonVersion == "" {
		pythonVersion = m.defaultPython
	}

	var cmd *exec.Cmd
	if m.useUv {
		args := []string{"venv", venvPath}
		if pythonVersion != "" {
			args = append(args, "--python", pythonVersion)
		}
		cmd = exec.Command("uv", args...)
	} else {
		cmd = exec.Command(utils.DefaultPythonCmd(pythonVersion), "-m", "venv", venvPath)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create venv: %v\n%s", err, output)
	}
	return nil
}

// List returns all venv names under baseDir.
func (m *Manager) List() ([]string, error) {
	entries, err := m.fs.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list venvs: %v", err)
	}
	var venvs []string
	for _, entry := range entries {
		if entry.IsDir() {
			venvs = append(venvs, entry.Name())
		}
	}
	sort.Strings(venvs)
	return venvs, nil
}

// Remove deletes a venv.
func (m *Manager) Remove(name string) error {
	p, err := m.requireVenv(name)
	if err != nil {
		return err
	}
	return m.fs.RemoveAll(p)
}

// Rename moves a venv to a new name.
func (m *Manager) Rename(oldName, newName string) error {
	src, err := m.requireVenv(oldName)
	if err != nil {
		return err
	}
	if err := ValidateName(newName); err != nil {
		return err
	}
	dst := m.VenvPath(newName)
	if m.fs.Exists(dst) {
		return fmt.Errorf("target venv '%s' already exists", newName)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to rename venv: %v", err)
	}
	// NOTE: internal absolute paths in the venv activate scripts still reference
	// the old path. `python -m venv` bakes them in. Fix by re-creating scripts
	// via `python -m venv --upgrade`.
	if out, err := exec.Command(utils.DefaultPythonCmd(""), "-m", "venv", "--upgrade", dst).CombinedOutput(); err != nil {
		return fmt.Errorf("renamed but activation scripts may be broken: %v\n%s", err, out)
	}
	return nil
}

// Install runs pip install -r on a venv.
func (m *Manager) Install(name, requirementsPath string) error {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return err
	}
	if !m.fs.Exists(requirementsPath) {
		return fmt.Errorf("requirements file '%s' not found", requirementsPath)
	}
	cmd := exec.Command(utils.PipPath(venvPath), "install", "-r", requirementsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install requirements: %v\n%s", err, output)
	}
	return nil
}

// Clone creates target as a copy of source (by pip freeze + install).
func (m *Manager) Clone(source, target string) error {
	sourcePath, err := m.requireVenv(source)
	if err != nil {
		return err
	}
	if m.fs.Exists(m.VenvPath(target)) {
		return fmt.Errorf("target venv '%s' already exists", target)
	}
	if err := m.Create(target, ""); err != nil {
		return err
	}
	targetPath := m.VenvPath(target)

	requirements, err := exec.Command(utils.PipPath(sourcePath), "freeze").Output()
	if err != nil {
		return fmt.Errorf("failed to get requirements: %v", err)
	}

	pipPath := utils.PipPath(targetPath)
	if runtime.GOOS == "windows" {
		tmp, err := os.CreateTemp("", "venv-clone-req-*.txt")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.Write(requirements); err != nil {
			tmp.Close()
			return err
		}
		tmp.Close()
		if output, err := exec.Command(pipPath, "install", "-r", tmp.Name()).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install requirements: %v\n%s", err, output)
		}
		return nil
	}
	cmd := exec.Command(pipPath, "install", "-r", "/dev/stdin")
	cmd.Stdin = bytes.NewReader(requirements)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install requirements: %v\n%s", err, output)
	}
	return nil
}

// Upgrade upgrades outdated packages in one or all venvs.
func (m *Manager) Upgrade(name string) error {
	targets, err := m.resolveTargets(name)
	if err != nil {
		return err
	}
	var errs []string
	for _, venvPath := range targets {
		pipPath := utils.PipPath(venvPath)
		output, err := exec.Command(pipPath, "list", "--outdated", "--format=json").Output()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: list failed: %v", venvPath, err))
			continue
		}
		var packages []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(output, &packages); err != nil {
			errs = append(errs, fmt.Sprintf("%s: parse failed: %v", venvPath, err))
			continue
		}
		for _, pkg := range packages {
			if out, err := exec.Command(pipPath, "install", "--upgrade", pkg.Name).CombinedOutput(); err != nil {
				errs = append(errs, fmt.Sprintf("%s/%s: %v\n%s", filepath.Base(venvPath), pkg.Name, err, out))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("upgrade completed with errors:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

// Clean purges pip cache and __pycache__ dirs.
func (m *Manager) Clean(name string) error {
	targets, err := m.resolveTargets(name)
	if err != nil {
		return err
	}
	for _, venvPath := range targets {
		if out, err := exec.Command(utils.PipPath(venvPath), "cache", "purge").CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clean pip cache: %v\n%s", err, out)
		}
		err := filepath.Walk(venvPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && info.Name() == "__pycache__" {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to clean pycache: %v", err)
		}
	}
	return nil
}

// GetActivationCommand returns the shell command to activate a venv.
func (m *Manager) GetActivationCommand(name, shell string) (string, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return "", err
	}
	binDir := utils.VenvBinDir(venvPath)
	switch shell {
	case "fish":
		return fmt.Sprintf("source %s/activate.fish", binDir), nil
	case "csh", "tcsh":
		return fmt.Sprintf("source %s/activate.csh", binDir), nil
	case "cmd":
		return fmt.Sprintf("%s\\activate.bat", binDir), nil
	case "pwsh", "powershell":
		return fmt.Sprintf("%s\\Activate.ps1", binDir), nil
	default:
		return fmt.Sprintf("source %s/activate", binDir), nil
	}
}

// ListPackages returns "name==version" strings for installed packages.
func (m *Manager) ListPackages(name string) ([]string, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	output, err := exec.Command(utils.PipPath(venvPath), "list", "--format=json").Output()
	if err != nil {
		return nil, err
	}
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, err
	}
	result := make([]string, len(packages))
	for i, pkg := range packages {
		result[i] = fmt.Sprintf("%s==%s", pkg.Name, pkg.Version)
	}
	return result, nil
}

// GetSize returns byte sizes keyed by venv name.
func (m *Manager) GetSize(name string) (map[string]int64, error) {
	sizes := make(map[string]int64)
	if name != "" {
		p, err := m.requireVenv(name)
		if err != nil {
			return nil, err
		}
		size, err := m.fs.GetDirSize(p)
		if err != nil {
			return nil, err
		}
		sizes[name] = size
		return sizes, nil
	}
	if !m.global {
		return nil, fmt.Errorf("please specify a venv name or use --global flag")
	}
	venvs, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, v := range venvs {
		size, err := m.fs.GetDirSize(m.VenvPath(v))
		if err != nil {
			return nil, err
		}
		sizes[v] = size
	}
	return sizes, nil
}

// Run executes a command inside a venv (PATH-prepended with venv bin dir).
// argv is the command and its arguments. Stdio is inherited.
func (m *Manager) Run(name string, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("no command provided")
	}
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return err
	}
	binDir := utils.VenvBinDir(venvPath)

	// Resolve command: prefer venv-local, fall back to system PATH.
	cmdName := argv[0]
	resolved := utils.VenvExe(venvPath, cmdName)
	if _, err := os.Stat(resolved); err != nil {
		if lp, lerr := exec.LookPath(cmdName); lerr == nil {
			resolved = lp
		} else {
			resolved = cmdName
		}
	}

	cmd := exec.Command(resolved, argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	env := os.Environ()
	env = append(env, "VIRTUAL_ENV="+venvPath)
	sep := string(os.PathListSeparator)
	newPath := binDir + sep + os.Getenv("PATH")
	env = utils.SetEnv(env, "PATH", newPath)
	// Drop PYTHONHOME as venv activate does.
	env = utils.RemoveEnv(env, "PYTHONHOME")
	cmd.Env = env
	return cmd.Run()
}

// Export writes a manifest describing a venv (or all when global).
type Manifest struct {
	Name          string   `json:"name"`
	PythonVersion string   `json:"python_version,omitempty"`
	Requirements  []string `json:"requirements"`
}

func (m *Manager) Export(name string) (*Manifest, error) {
	venvPath, err := m.requireVenv(name)
	if err != nil {
		return nil, err
	}
	pkgs, err := m.ListPackages(name)
	if err != nil {
		return nil, err
	}
	ver, _ := exec.Command(utils.PythonPath(venvPath), "-c", "import sys;print('%d.%d'%sys.version_info[:2])").Output()
	return &Manifest{
		Name:          name,
		PythonVersion: strings.TrimSpace(string(ver)),
		Requirements:  pkgs,
	}, nil
}

// Import creates a venv from a manifest and installs its requirements.
func (m *Manager) Import(mf *Manifest) error {
	if err := m.Create(mf.Name, mf.PythonVersion); err != nil {
		return err
	}
	if len(mf.Requirements) == 0 {
		return nil
	}
	tmp, err := os.CreateTemp("", "venv-req-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	for _, r := range mf.Requirements {
		if _, err := tmp.WriteString(r + "\n"); err != nil {
			return err
		}
	}
	tmp.Close()
	return m.Install(mf.Name, tmp.Name())
}

// StaleVenv describes a venv unused for a given duration.
type StaleVenv struct {
	Name    string
	ModTime time.Time
}

// FindStale lists venvs whose mtime is older than `days`.
func (m *Manager) FindStale(days int) ([]StaleVenv, error) {
	venvs, err := m.List()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	var stale []StaleVenv
	for _, v := range venvs {
		info, err := os.Stat(m.VenvPath(v))
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			stale = append(stale, StaleVenv{Name: v, ModTime: info.ModTime()})
		}
	}
	return stale, nil
}

// DoctorReport summarizes environment health.
type DoctorReport struct {
	BaseDir        string   `json:"base_dir"`
	BaseDirExists  bool     `json:"base_dir_exists"`
	UvAvailable    bool     `json:"uv_available"`
	PythonVersions []string `json:"python_versions"`
	VenvCount      int      `json:"venv_count"`
	Broken         []string `json:"broken,omitempty"`
}

// Doctor inspects the environment and returns a report.
func (m *Manager) Doctor() *DoctorReport {
	r := &DoctorReport{
		BaseDir:       m.baseDir,
		BaseDirExists: m.fs.Exists(m.baseDir),
		UvAvailable:   uvAvailable(),
	}
	for _, v := range []string{"python3", "python", "python3.8", "python3.9", "python3.10", "python3.11", "python3.12", "python3.13"} {
		if _, err := exec.LookPath(v); err == nil {
			out, _ := exec.Command(v, "--version").Output()
			r.PythonVersions = append(r.PythonVersions, strings.TrimSpace(string(out)))
		}
	}
	if venvs, err := m.List(); err == nil {
		r.VenvCount = len(venvs)
		for _, v := range venvs {
			if !m.fs.Exists(utils.PythonPath(m.VenvPath(v))) {
				r.Broken = append(r.Broken, v)
			}
		}
	}
	return r
}

func uvAvailable() bool {
	_, err := exec.LookPath("uv")
	return err == nil
}
