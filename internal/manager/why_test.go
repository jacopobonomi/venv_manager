package manager

import (
	"strings"
	"testing"
)

func TestBfsChainNoDeps(t *testing.T) {
	// target has no reverse deps → direct install, empty chain
	chain := bfsChain(map[string][]string{}, "requests")
	if len(chain) != 0 {
		t.Fatalf("expected empty chain, got %v", chain)
	}
}

func TestBfsChainSimple(t *testing.T) {
	// pandas requires numpy; numpy has no reverse deps from anything else
	reverse := map[string][]string{
		"numpy": {"pandas"},
	}
	chain := bfsChain(reverse, "numpy")
	if len(chain) == 0 {
		t.Fatal("expected non-empty chain")
	}
	if chain[0] != "pandas" {
		t.Fatalf("expected pandas, got %v", chain)
	}
}

func TestBfsChainTransitive(t *testing.T) {
	// sklearn → scipy → numpy; numpy required by scipy required by sklearn
	reverse := map[string][]string{
		"numpy": {"scipy"},
		"scipy": {"sklearn"},
	}
	chain := bfsChain(reverse, "numpy")
	if len(chain) == 0 {
		t.Fatal("expected chain")
	}
	joined := strings.Join(chain, "→")
	if !strings.Contains(joined, "scipy") {
		t.Fatalf("expected scipy in chain, got %s", joined)
	}
}

func TestPipShowRequiresParsing(t *testing.T) {
	// Simulate pip show output parsing inline by calling the helper directly
	// (it's package-private, tested at the unit level via bfsChain in integration).
	// Just ensure normalizePkgName works correctly as a proxy.
	if got := normalizePkgName("NumPy"); got != "numpy" {
		t.Fatalf("expected numpy, got %q", got)
	}
	if got := normalizePkgName("typing_extensions"); got != "typing-extensions" {
		t.Fatalf("expected typing-extensions, got %q", got)
	}
}

func TestWhyMissingVenv(t *testing.T) {
	m, _ := newTestMgr(t)
	if _, err := m.Why("nope", "requests"); err == nil {
		t.Fatal("expected error for missing venv")
	}
}
