package mcp

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// installPackages installs a set of packages into an existing venv and
// returns pip's combined output.
func (s *Server) installPackages(name string, packages []string) (string, error) {
	venvPath, err := s.mgr.EnsureVenv(name)
	if err != nil {
		return "", err
	}
	args := append([]string{"install"}, packages...)
	out, err := exec.Command(utils.PipPath(venvPath), args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("pip install failed: %v", err)
	}
	return string(out), nil
}

// runInVenv runs a command inside a venv and captures its combined output.
func (s *Server) runInVenv(name string, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("name and command are required")
	}
	venvPath, err := s.mgr.EnsureVenv(name)
	if err != nil {
		return "", err
	}
	binDir := utils.VenvBinDir(venvPath)
	resolved := utils.VenvExe(venvPath, argv[0])
	if _, err := os.Stat(resolved); err != nil {
		if lp, lerr := exec.LookPath(argv[0]); lerr == nil {
			resolved = lp
		} else {
			resolved = argv[0]
		}
	}
	cmd := exec.Command(resolved, argv[1:]...)
	env := append(os.Environ(), "VIRTUAL_ENV="+venvPath)
	sep := string(os.PathListSeparator)
	env = utils.SetEnv(env, "PATH", binDir+sep+os.Getenv("PATH"))
	env = utils.RemoveEnv(env, "PYTHONHOME")
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("command failed: %v", err)
	}
	return out.String(), nil
}

// execEphemeral is the tools/call variant of `venv-manager exec`: it captures
// output rather than inheriting stdio.
func (s *Server) execEphemeral(packages []string, pythonVersion string, argv []string, sandbox bool) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("command is required")
	}
	if sandbox {
		// Sandboxed execution inherits stdio in manager.Exec; capturing it
		// here would require duplicating the sandbox wiring. Not supported yet.
		return "", fmt.Errorf("sandbox mode over MCP not yet supported; use CLI `venv-manager exec --sandbox`")
	}

	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	tmpName := "mcp-eph-" + hex.EncodeToString(buf)
	if err := s.mgr.Create(tmpName, pythonVersion); err != nil {
		return "", err
	}
	defer func() {
		_ = s.mgr.Remove(tmpName)
	}()
	if len(packages) > 0 {
		if out, err := s.installPackages(tmpName, packages); err != nil {
			return out, err
		}
	}
	return s.runInVenv(tmpName, argv)
}
