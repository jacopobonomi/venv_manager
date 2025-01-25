package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

type Manager struct {
	baseDir string
	fs      utils.FileSystem
	global  bool
}

func (m *Manager) SetGlobal(global bool) {
	m.global = global
}

func (m *Manager) GetBaseDir() string {
	return m.baseDir
}

func (m *Manager) Install(name, requirementsPath string) error {
	venvPath := filepath.Join(m.baseDir, name)
	if !m.fs.Exists(venvPath) {
		return fmt.Errorf("venv '%s' does not exist", name)
	}

	if !m.fs.Exists(requirementsPath) {
		return fmt.Errorf("requirements file '%s' not found", requirementsPath)
	}

	pipPath := filepath.Join(venvPath, "bin", "pip")
	if runtime.GOOS == "windows" {
		pipPath = filepath.Join(venvPath, "Scripts", "pip.exe")
	}

	cmd := exec.Command(pipPath, "install", "-r", requirementsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install requirements: %v\n%s", err, output)
	}

	return nil
}

func (m *Manager) Clone(source, target string) error {
	sourcePath := filepath.Join(m.baseDir, source)
	targetPath := filepath.Join(m.baseDir, target)

	if !m.fs.Exists(sourcePath) {
		return fmt.Errorf("source venv '%s' does not exist", source)
	}
	if m.fs.Exists(targetPath) {
		return fmt.Errorf("target venv '%s' already exists", target)
	}

	// Create new venv
	if err := m.Create(target, ""); err != nil {
		return err
	}

	// Get pip freeze from source
	cmd := exec.Command(filepath.Join(sourcePath, "bin", "pip"), "freeze")
	if runtime.GOOS == "windows" {
		cmd = exec.Command(filepath.Join(sourcePath, "Scripts", "pip.exe"), "freeze")
	}
	requirements, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get requirements: %v", err)
	}

	// Install requirements in target
	pipPath := filepath.Join(targetPath, "bin", "pip")
	if runtime.GOOS == "windows" {
		pipPath = filepath.Join(targetPath, "Scripts", "pip.exe")
	}
	cmd = exec.Command(pipPath, "install", "-r", "/dev/stdin")
	if runtime.GOOS == "windows" {
		tempFile := filepath.Join(os.TempDir(), "requirements.txt")
		if err := os.WriteFile(tempFile, requirements, 0644); err != nil {
			return err
		}
		defer os.Remove(tempFile)
		cmd = exec.Command(pipPath, "install", "-r", tempFile)
	}
	cmd.Stdin = bytes.NewReader(requirements)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install requirements: %v\n%s", err, output)
	}

	return nil
}

func (m *Manager) Upgrade(name string) error {
	venvPaths := []string{filepath.Join(m.baseDir, name)}
	if m.global {
		venvs, err := m.List()
		if err != nil {
			return err
		}
		venvPaths = make([]string, len(venvs))
		for i, venv := range venvs {
			venvPaths[i] = filepath.Join(m.baseDir, venv)
		}
	}

	for _, venvPath := range venvPaths {
		pipPath := filepath.Join(venvPath, "bin", "pip")
		if runtime.GOOS == "windows" {
			pipPath = filepath.Join(venvPath, "Scripts", "pip.exe")
		}

		cmd := exec.Command(pipPath, "list", "--outdated", "--format=json")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to list outdated packages: %v", err)
		}

		var packages []struct {
			Name    string `json:"name"`
			Version string `json:"latest_version"`
		}
		if err := json.Unmarshal(output, &packages); err != nil {
			return fmt.Errorf("failed to parse outdated packages: %v", err)
		}

		for _, pkg := range packages {
			cmd = exec.Command(pipPath, "install", "--upgrade", pkg.Name)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to upgrade %s: %v\n%s", pkg.Name, err, output)
			}
		}
	}

	return nil
}

func (m *Manager) Clean(name string) error {
	venvPaths := []string{filepath.Join(m.baseDir, name)}
	if m.global {
		venvs, err := m.List()
		if err != nil {
			return err
		}
		venvPaths = make([]string, len(venvs))
		for i, venv := range venvs {
			venvPaths[i] = filepath.Join(m.baseDir, venv)
		}
	}

	for _, venvPath := range venvPaths {
		// Clean pip cache
		pipPath := filepath.Join(venvPath, "bin", "pip")
		if runtime.GOOS == "windows" {
			pipPath = filepath.Join(venvPath, "Scripts", "pip.exe")
		}
		cmd := exec.Command(pipPath, "cache", "purge")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clean pip cache: %v\n%s", err, output)
		}

		// Clean __pycache__ directories
		err := filepath.Walk(venvPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && info.Name() == "__pycache__" {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to clean pycache: %v", err)
		}
	}

	return nil
}

func New(baseDir string) *Manager {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		baseDir = filepath.Join(homeDir, ".venvs")
	}
	return &Manager{
		baseDir: baseDir,
		fs:      utils.NewFileSystem(),
	}
}

func (m *Manager) GetActivationCommand(name, shell string) (string, error) {
	venvPath := filepath.Join(m.baseDir, name)
	if !m.fs.Exists(venvPath) {
		return "", fmt.Errorf("venv '%s' does not exist", name)
	}

	switch shell {
	case "bash", "zsh", "sh":
		return fmt.Sprintf("source %s/bin/activate", venvPath), nil
	case "fish":
		return fmt.Sprintf("source %s/bin/activate.fish", venvPath), nil
	case "csh", "tcsh":
		return fmt.Sprintf("source %s/bin/activate.csh", venvPath), nil
	case "cmd":
		return fmt.Sprintf("%s\\Scripts\\activate.bat", venvPath), nil
	case "pwsh", "powershell":
		return fmt.Sprintf("%s\\Scripts\\Activate.ps1", venvPath), nil
	default:
		return fmt.Sprintf("source %s/bin/activate", venvPath), nil
	}
}

func (m *Manager) Create(name, pythonVersion string) error {
	venvPath := filepath.Join(m.baseDir, name)

	if m.fs.Exists(venvPath) {
		return fmt.Errorf("'%s' already exists", name)
	}

	if err := m.fs.CreateDir(m.baseDir); err != nil {
		return fmt.Errorf("failed to create base directory: %v", err)
	}

	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}
	if pythonVersion != "" {
		pythonCmd = fmt.Sprintf("python%s", pythonVersion)
	}

	cmd := exec.Command(pythonCmd, "-m", "venv", venvPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create venv: %v\n%s", err, output)
	}

	return nil
}

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
	return venvs, nil
}

func (m *Manager) Remove(name string) error {
	venvPath := filepath.Join(m.baseDir, name)
	if !m.fs.Exists(venvPath) {
		return fmt.Errorf("venv '%s' does not exist", name)
	}
	return m.fs.RemoveAll(venvPath)
}
