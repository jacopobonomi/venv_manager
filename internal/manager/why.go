package manager

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jacopobonomi/venv-manager/internal/utils"
)

// DepNode is one package in the dependency graph.
type DepNode struct {
	Name     string     `json:"name"`
	Version  string     `json:"version"`
	Required []string   `json:"required_by"` // packages that directly depend on this one
	Requires []*DepNode `json:"requires,omitempty"`
}

// WhyResult explains why a package is installed in a venv.
type WhyResult struct {
	Venv    string     `json:"venv"`
	Package string     `json:"package"`
	// Direct means the package appears in the top-level installed set
	// without any other installed package requiring it.
	Direct  bool       `json:"direct"`
	// RequiredBy is the chain of packages that pull this one in,
	// ordered from the top-level dependency down to the target.
	RequiredBy []string `json:"required_by"`
}

// Why returns an explanation of why pkg is installed in venv.
func (m *Manager) Why(venvName, pkg string) (*WhyResult, error) {
	venvPath, err := m.requireVenv(venvName)
	if err != nil {
		return nil, err
	}

	// pip list --format=json for the full installed set
	allPkgs, err := m.installedSet(venvPath)
	if err != nil {
		return nil, err
	}

	target := normalizePkgName(pkg)
	if _, ok := allPkgs[target]; !ok {
		return nil, fmt.Errorf("package %q is not installed in venv %q", pkg, venvName)
	}

	// Build reverse dep graph: pkg → list of packages that require it
	reverseDeps := map[string][]string{}
	for name := range allPkgs {
		requires, err := pipShowRequires(utils.PipPath(venvPath), name)
		if err != nil {
			continue
		}
		for _, req := range requires {
			r := normalizePkgName(req)
			reverseDeps[r] = append(reverseDeps[r], name)
		}
	}

	// BFS from target upward through reverseDeps to find who requires it
	chain := bfsChain(reverseDeps, target)

	return &WhyResult{
		Venv:       venvName,
		Package:    allPkgs[target], // canonical capitalisation
		Direct:     len(reverseDeps[target]) == 0,
		RequiredBy: chain,
	}, nil
}

// installedSet returns a map of normalised-name → canonical-name for all
// packages installed in the venv.
func (m *Manager) installedSet(venvPath string) (map[string]string, error) {
	out, err := exec.Command(utils.PipPath(venvPath), "list", "--format=json").Output()
	if err != nil {
		return nil, fmt.Errorf("pip list: %v", err)
	}
	var pkgs []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &pkgs); err != nil {
		return nil, err
	}
	set := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		set[normalizePkgName(p.Name)] = p.Name
	}
	return set, nil
}

// pipShowRequires returns the Requires list from `pip show <name>`.
func pipShowRequires(pipPath, name string) ([]string, error) {
	out, err := exec.Command(pipPath, "show", name).Output()
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "Requires:") {
			continue
		}
		rest := strings.TrimPrefix(line, "Requires:")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return nil, nil
		}
		var result []string
		for _, r := range strings.Split(rest, ",") {
			r = strings.TrimSpace(r)
			// strip version specifiers (e.g. "numpy>=1.0" → "numpy")
			if i := strings.IndexAny(r, ">=<!~"); i > 0 {
				r = r[:i]
			}
			r = strings.TrimSpace(r)
			if r != "" {
				result = append(result, r)
			}
		}
		return result, nil
	}
	return nil, nil
}

// bfsChain finds the shortest chain of packages that require target,
// returned as a human-readable path like ["pandas", "numpy"].
func bfsChain(reverseDeps map[string][]string, target string) []string {
	parents := reverseDeps[target]
	if len(parents) == 0 {
		return nil
	}
	// BFS: find any top-level package (one with no reverse deps) that
	// transitively requires target.
	type node struct {
		name  string
		chain []string
	}
	visited := map[string]bool{target: true}
	queue := []node{}
	for _, p := range parents {
		queue = append(queue, node{p, []string{p}})
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur.name] {
			continue
		}
		visited[cur.name] = true
		up := reverseDeps[cur.name]
		if len(up) == 0 {
			// cur.name is a top-level dependency — return the chain
			return cur.chain
		}
		for _, p := range up {
			if !visited[p] {
				chain := make([]string, len(cur.chain)+1)
				copy(chain, cur.chain)
				chain[len(cur.chain)] = p
				queue = append(queue, node{p, chain})
			}
		}
	}
	// Didn't find a root — return direct parents
	return parents
}
