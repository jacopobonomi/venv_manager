package utils

import (
	"path/filepath"
	"runtime"
)

// VenvBinDir returns the directory containing executables inside a venv.
func VenvBinDir(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts")
	}
	return filepath.Join(venvPath, "bin")
}

// VenvExe returns the full path to an executable inside a venv (adds .exe on Windows).
func VenvExe(venvPath, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", name+".exe")
	}
	return filepath.Join(venvPath, "bin", name)
}

// PipPath returns the path to pip inside a venv.
func PipPath(venvPath string) string { return VenvExe(venvPath, "pip") }

// PythonPath returns the path to python inside a venv.
func PythonPath(venvPath string) string { return VenvExe(venvPath, "python") }

// DefaultPythonCmd returns the system python command to use for `python -m venv`.
func DefaultPythonCmd(version string) string {
	if version != "" {
		return "python" + version
	}
	if runtime.GOOS == "windows" {
		return "python"
	}
	return "python3"
}
