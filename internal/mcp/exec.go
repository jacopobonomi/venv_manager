package mcp

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/jacopobonomi/venv-manager/internal/manager"
	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// installPackages installs a set of packages into an existing venv and
// returns pip's combined output.
func (s *Server) installPackages(name string, packages []string) (string, error) {
	venvPath := s.mgr.VenvPath(name)
	if _, err := os.Stat(venvPath); err != nil {
		return "", fmt.Errorf("venv %q does not exist", name)
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
	if name == "" || len(argv) == 0 {
		return "", fmt.Errorf("name and command are required")
	}
	venvPath := s.mgr.VenvPath(name)
	if _, err := os.Stat(venvPath); err != nil {
		return "", fmt.Errorf("venv %q does not exist", name)
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
	env = setEnv(env, "PATH", binDir+sep+os.Getenv("PATH"))
	env = removeEnv(env, "PYTHONHOME")
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("command failed: %v", err)
	}
	return out.String(), nil
}

// execEphemeral runs the tools/call variant: capture output rather than
// inherit stdio (which the CLI variant does).
func (s *Server) execEphemeral(packages []string, pythonVersion string, argv []string, sandbox bool) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("command is required")
	}
	// Delegate creation/install by borrowing manager.Exec logic — but we
	// need captured output. We roll a small variant here.
	var buf bytes.Buffer
	stdinR, stdinW, _ := os.Pipe()
	stdinW.Close()
	defer stdinR.Close()

	// We reuse manager.Exec by redirecting stdio through a pipe. Simpler:
	// use temporary files. Here we just reuse os/exec directly:
	tmpName := fmt.Sprintf("mcp-eph-%d", os.Getpid())
	if err := s.mgr.Create(tmpName, pythonVersion); err != nil {
		return "", err
	}
	defer func() {
		_ = s.mgr.Remove(tmpName)
	}()
	if len(packages) > 0 {
		if _, err := s.installPackages(tmpName, packages); err != nil {
			return buf.String(), err
		}
	}
	if sandbox {
		// Sandboxed exec uses the manager helper; we can't easily capture
		// output while also sandboxing without duplication, so we accept a
		// simpler path: run via manager.Exec with Keep=true, then read logs
		// isn't feasible. Report the constraint.
		return "", fmt.Errorf("sandbox mode over MCP not yet supported; use CLI `venv-manager exec --sandbox`")
	}
	out, err := s.runInVenv(tmpName, argv)
	if err != nil {
		return out, err
	}
	return out, nil
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func removeEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, e := range env {
		if len(e) < len(prefix) || e[:len(prefix)] != prefix {
			out = append(out, e)
		}
	}
	return out
}

// suppress unused import warning across build tags
var _ = manager.Options{}
