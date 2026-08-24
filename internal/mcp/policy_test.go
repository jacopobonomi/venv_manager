package mcp

import (
	"strings"
	"testing"
)

func TestNewPolicyValidationAndAllowlist(t *testing.T) {
	if _, err := NewPolicy("invalid", nil); err == nil {
		t.Fatal("invalid policy should fail")
	}
	if _, err := NewPolicy("safe", []string{"not-a-tool"}); err == nil {
		t.Fatal("unknown allowlisted tool should fail")
	}
	policy, err := NewPolicy("safe", []string{"list_venvs"})
	if err != nil {
		t.Fatal(err)
	}
	if !policy.visible("list_venvs") || policy.visible("doctor") {
		t.Fatal("allowlist visibility is incorrect")
	}
	if err := policy.authorize("doctor", nil); err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestReadOnlyPolicy(t *testing.T) {
	policy, _ := NewPolicy("read-only", nil)
	for _, tool := range []string{"list_venvs", "describe_venv", "diff_snapshots", "list_registry"} {
		if err := policy.authorize(tool, nil); err != nil {
			t.Errorf("read-only tool %s denied: %v", tool, err)
		}
	}
	if err := policy.authorize("create_venv", nil); err == nil {
		t.Fatal("read-only policy allowed mutation")
	}
}

func TestSafePolicyConfirmations(t *testing.T) {
	policy, _ := NewPolicy("safe", nil)
	if err := policy.authorize("create_venv", nil); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"remove_venv", "install_packages", "run_in_venv", "rollback_venv", "set_registry_metadata"} {
		if err := policy.authorize(tool, nil); err == nil {
			t.Errorf("safe policy allowed %s without confirmation", tool)
		}
		if err := policy.authorize(tool, map[string]any{"confirm": true}); err != nil {
			t.Errorf("safe policy denied confirmed %s: %v", tool, err)
		}
	}
	if err := policy.authorize("exec_ephemeral", map[string]any{"sandbox": true}); err != nil {
		t.Fatalf("sandboxed ephemeral execution denied: %v", err)
	}
}

func TestFullAndZeroPolicyAllowEverything(t *testing.T) {
	full, _ := NewPolicy("full", nil)
	for _, policy := range []Policy{full, {}} {
		if err := policy.authorize("remove_venv", nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewServerDefaultsToSafePolicy(t *testing.T) {
	server := NewServer(nil)
	if server.policy.Mode != PolicySafe {
		t.Fatalf("default policy = %q, want safe", server.policy.Mode)
	}
}
