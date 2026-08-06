package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jacopobonomi/venv-manager/internal/manager"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	return &Server{mgr: manager.New(dir)}, dir
}

// remove_venv with a missing/empty name must fail instead of resolving to the
// base directory and wiping every venv (regression for the "" → baseDir bug).
func TestDispatchRemoveEmptyName(t *testing.T) {
	s, dir := newTestServer(t)
	os.MkdirAll(filepath.Join(dir, "keep"), 0o755)
	if _, err := s.dispatch("remove_venv", map[string]any{}); err == nil {
		t.Fatal("remove_venv without name must fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "keep")); err != nil {
		t.Fatalf("base dir contents were touched: %v", err)
	}
}

// Tool arguments come from an LLM: path traversal in venv names must be
// rejected on every tool, not silently joined into a filesystem path.
func TestDispatchRejectsTraversalNames(t *testing.T) {
	s, dir := newTestServer(t)
	outside := filepath.Join(filepath.Dir(dir), "mcp-outside-victim")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outside)

	evil := "../" + filepath.Base(outside)
	for _, tool := range []string{"remove_venv", "describe_venv", "snapshot_venv", "list_snapshots", "rollback_venv"} {
		if _, err := s.dispatch(tool, map[string]any{"name": evil}); err == nil {
			t.Errorf("%s accepted traversal name %q", tool, evil)
		}
	}
	if _, err := s.dispatch("create_venv", map[string]any{"name": evil}); err == nil {
		t.Errorf("create_venv accepted traversal name %q", evil)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("directory outside base dir was removed: %v", err)
	}
}

func TestDispatchUnknownTool(t *testing.T) {
	s, _ := newTestServer(t)
	_, err := s.dispatch("nope", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected unknown-tool error, got %v", err)
	}
}

func TestDispatchInstallRequiresInput(t *testing.T) {
	s, dir := newTestServer(t)
	os.MkdirAll(filepath.Join(dir, "v"), 0o755)
	if _, err := s.dispatch("install_packages", map[string]any{"name": "v"}); err == nil {
		t.Fatal("install_packages without packages/requirements_file must fail")
	}
}

func TestToolCatalogSchemasWellFormed(t *testing.T) {
	for _, tool := range toolCatalog() {
		if tool.Name == "" || tool.Description == "" {
			t.Errorf("tool with empty name/description: %+v", tool)
		}
		if tool.InputSchema["type"] != "object" {
			t.Errorf("%s: inputSchema.type must be object", tool.Name)
		}
	}
}

func TestDispatchListVenvs(t *testing.T) {
	s, _ := newTestServer(t)
	out, err := s.dispatch("list_venvs", nil)
	if err != nil {
		t.Fatalf("list_venvs: %v", err)
	}
	// empty list serialises as either "[]" or "null" in Go JSON
	if !strings.Contains(out, "[") && out != "null" {
		t.Fatalf("expected JSON array or null, got %s", out)
	}
}

func TestDispatchDoctor(t *testing.T) {
	s, _ := newTestServer(t)
	out, err := s.dispatch("doctor", nil)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "base_dir") {
		t.Fatalf("expected base_dir in doctor output, got %s", out)
	}
}

func TestDispatchScanImports(t *testing.T) {
	s, dir := newTestServer(t)
	pyfile := filepath.Join(dir, "app.py")
	os.WriteFile(pyfile, []byte("import requests\nimport os\n"), 0o644)
	out, err := s.dispatch("scan_imports", map[string]any{"path": pyfile})
	if err != nil {
		t.Fatalf("scan_imports: %v", err)
	}
	if !strings.Contains(out, "requests") {
		t.Fatalf("expected requests in output, got %s", out)
	}
}

func TestDispatchListSnapshots_missingVenv(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.dispatch("list_snapshots", map[string]any{"name": "nope"}); err == nil {
		t.Fatal("expected error for missing venv")
	}
}

func TestDispatchRollback_missingVenv(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.dispatch("rollback_venv", map[string]any{"name": "nope"}); err == nil {
		t.Fatal("expected error for missing venv")
	}
}

func TestDispatchCreateRequiresName(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.dispatch("create_venv", map[string]any{}); err == nil {
		t.Fatal("create_venv without name must fail")
	}
}

func TestWriteErr(t *testing.T) {
	dir := t.TempDir()
	buf := &strings.Builder{}
	s := &Server{mgr: manager.New(dir), out: buf, log: &strings.Builder{}}
	s.writeErr(json.RawMessage(`1`), -32601, "not found")
	if !strings.Contains(buf.String(), "-32601") {
		t.Fatalf("expected error code in output, got %s", buf)
	}
}
