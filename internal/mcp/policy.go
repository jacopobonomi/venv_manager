package mcp

import "fmt"

// PolicyMode controls which MCP tools may mutate environments or execute code.
type PolicyMode string

const (
	PolicyReadOnly PolicyMode = "read-only"
	PolicySafe     PolicyMode = "safe"
	PolicyFull     PolicyMode = "full"
)

// Policy combines a mode with an optional tool allowlist.
type Policy struct {
	Mode         PolicyMode
	AllowedTools map[string]bool
}

// NewPolicy validates a policy mode and builds its allowlist.
func NewPolicy(mode string, allowedTools []string) (Policy, error) {
	policy := Policy{Mode: PolicyMode(mode)}
	switch policy.Mode {
	case PolicyReadOnly, PolicySafe, PolicyFull:
	default:
		return Policy{}, fmt.Errorf("invalid MCP policy %q: use read-only, safe, or full", mode)
	}
	if len(allowedTools) > 0 {
		knownTools := map[string]bool{}
		for _, tool := range toolCatalog() {
			knownTools[tool.Name] = true
		}
		policy.AllowedTools = make(map[string]bool, len(allowedTools))
		for _, tool := range allowedTools {
			if !knownTools[tool] {
				return Policy{}, fmt.Errorf("unknown MCP tool %q in allowlist", tool)
			}
			policy.AllowedTools[tool] = true
		}
	}
	return policy, nil
}

func (p Policy) authorize(tool string, args map[string]any) error {
	mode := p.Mode
	if mode == "" {
		mode = PolicyFull
	}
	if len(p.AllowedTools) > 0 && !p.AllowedTools[tool] {
		return fmt.Errorf("tool %q is not in the MCP allowlist", tool)
	}
	if mode == PolicyFull {
		return nil
	}
	if readOnlyTools[tool] {
		return nil
	}
	if mode == PolicyReadOnly {
		return fmt.Errorf("tool %q is blocked by read-only MCP policy", tool)
	}
	if tool == "exec_ephemeral" {
		if sandbox, _ := args["sandbox"].(bool); sandbox {
			return nil
		}
	}
	if confirmationRequiredTools[tool] {
		if confirmed, _ := args["confirm"].(bool); !confirmed {
			return fmt.Errorf("tool %q requires confirm=true under safe MCP policy", tool)
		}
	}
	return nil
}

func (p Policy) visible(tool string) bool {
	mode := p.Mode
	if mode == "" {
		mode = PolicyFull
	}
	if len(p.AllowedTools) > 0 && !p.AllowedTools[tool] {
		return false
	}
	return mode != PolicyReadOnly || readOnlyTools[tool]
}

var readOnlyTools = map[string]bool{
	"list_venvs":     true,
	"describe_venv":  true,
	"doctor":         true,
	"list_snapshots": true,
	"diff_snapshots": true,
	"why":            true,
	"scan_imports":   true,
	"list_registry":  true,
}

var confirmationRequiredTools = map[string]bool{
	"remove_venv":           true,
	"install_packages":      true,
	"run_in_venv":           true,
	"exec_ephemeral":        true,
	"rollback_venv":         true,
	"set_registry_metadata": true,
}
