package manager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// ExecOptions configures an ephemeral run.
type ExecOptions struct {
	// Packages to install with pip before running. May be empty.
	Packages []string
	// RequirementsFile path to install with `pip install -r`. May be empty.
	RequirementsFile string
	// PythonVersion (e.g. "3.12"). Empty means system default.
	PythonVersion string
	// Sandbox enables OS-level sandboxing (macOS sandbox-exec / Linux bwrap).
	Sandbox bool
	// Keep prevents cleanup and prints the venv path on stderr.
	Keep bool
}

// Exec creates an ephemeral venv, installs the requested packages, then runs
// argv inside it with stdio inherited. Cleanup happens on return unless Keep.
func (m *Manager) Exec(opts ExecOptions, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("no command provided")
	}

	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	tempName := "eph-" + hex.EncodeToString(buf)
	venvPath := m.VenvPath(tempName)

	if err := m.Create(tempName, opts.PythonVersion); err != nil {
		return err
	}
	if !opts.Keep {
		defer m.fs.RemoveAll(venvPath)
	} else {
		defer fmt.Fprintf(os.Stderr, "kept ephemeral venv at %s\n", venvPath)
	}

	if opts.RequirementsFile != "" {
		if err := m.Install(tempName, opts.RequirementsFile); err != nil {
			return err
		}
	}
	if len(opts.Packages) > 0 {
		args := append([]string{"install"}, opts.Packages...)
		if out, err := exec.Command(utils.PipPath(venvPath), args...).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install packages: %v\n%s", err, out)
		}
	}

	binDir := utils.VenvBinDir(venvPath)
	cmdName := argv[0]
	resolved := utils.VenvExe(venvPath, cmdName)
	if _, err := os.Stat(resolved); err != nil {
		if lp, lerr := exec.LookPath(cmdName); lerr == nil {
			resolved = lp
		} else {
			resolved = cmdName
		}
	}

	env := append(os.Environ(),
		"VIRTUAL_ENV="+venvPath,
	)
	sep := string(os.PathListSeparator)
	env = utils.SetEnv(env, "PATH", binDir+sep+os.Getenv("PATH"))
	env = utils.RemoveEnv(env, "PYTHONHOME")

	var cmd *exec.Cmd
	if opts.Sandbox {
		wrapper, wargs, err := sandboxWrap(venvPath)
		if err != nil {
			return err
		}
		cmd = exec.Command(wrapper, append(wargs, append([]string{resolved}, argv[1:]...)...)...)
	} else {
		cmd = exec.Command(resolved, argv[1:]...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = env
	return cmd.Run()
}

// sandboxWrap returns the wrapper binary + its args for sandboxed execution.
// The final argv (command to run) is appended by the caller.
//
// macOS: sandbox-exec with a profile that denies network and restricts writes.
// Linux: bwrap with network unshared and filesystem read-only outside venv+tmp.
// Others: error — no supported sandbox backend.
func sandboxWrap(venvPath string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("sandbox-exec"); err != nil {
			return "", nil, fmt.Errorf("sandbox-exec not found (macOS)")
		}
		// Escape the path before splicing it into the SBPL profile: a quote
		// or backslash in the venv path would otherwise break out of the
		// (subpath "...") string literal.
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(venvPath)
		profile := strings.TrimSpace(`
(version 1)
(deny default)
(allow process*)
(allow signal)
(allow sysctl-read)
(allow file-read*)
(allow file-write* (subpath "/tmp") (subpath "/private/tmp") (subpath "/private/var/folders") (subpath "` + escaped + `"))
(allow mach-lookup)
(allow ipc-posix-shm)
`)
		return "sandbox-exec", []string{"-p", profile}, nil

	case "linux":
		if _, err := exec.LookPath("bwrap"); err != nil {
			return "", nil, fmt.Errorf("bwrap not found (install bubblewrap)")
		}
		tmp := os.TempDir()
		return "bwrap", []string{
			"--ro-bind", "/", "/",
			"--bind", tmp, tmp,
			"--bind", venvPath, venvPath,
			"--dev", "/dev",
			"--proc", "/proc",
			"--unshare-net",
			"--die-with-parent",
		}, nil

	default:
		return "", nil, fmt.Errorf("sandbox mode not supported on %s", runtime.GOOS)
	}
}
