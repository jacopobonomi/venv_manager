package mcp

import (
	"bufio"
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

func TestServeProcessesFinalLineWithoutNewline(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		`not-json`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	}, "\n")
	out := &strings.Builder{}
	s := &Server{
		mgr: manager.New(t.TempDir()),
		in:  bufio.NewReader(strings.NewReader(input)),
		out: out,
		log: &strings.Builder{},
	}
	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}
	result := out.String()
	for _, expected := range []string{`"id":1`, `"id":2`, `"protocolVersion"`, `"code":-32700`} {
		if !strings.Contains(result, expected) {
			t.Errorf("missing %q in %s", expected, result)
		}
	}
}

func TestHandleMethodsAndNotifications(t *testing.T) {
	out := &strings.Builder{}
	s := &Server{mgr: manager.New(t.TempDir()), out: out, log: &strings.Builder{}}
	for _, req := range []rpcRequest{
		{ID: json.RawMessage(`1`), Method: "tools/list"},
		{ID: json.RawMessage(`2`), Method: "missing/method"},
		{Method: "notifications/initialized"},
		{Method: "unknown-notification"},
	} {
		s.handle(req)
	}
	result := out.String()
	if !strings.Contains(result, "list_venvs") || !strings.Contains(result, "-32601") {
		t.Fatalf("unexpected handler output: %s", result)
	}
}

func TestHandleToolCallResults(t *testing.T) {
	out := &strings.Builder{}
	s := &Server{mgr: manager.New(t.TempDir()), out: out, log: &strings.Builder{}}
	s.handleToolCall(rpcRequest{ID: json.RawMessage(`1`), Params: json.RawMessage(`{`)})
	s.handleToolCall(rpcRequest{ID: json.RawMessage(`2`), Params: json.RawMessage(`{"name":"unknown","arguments":{}}`)})
	s.handleToolCall(rpcRequest{ID: json.RawMessage(`3`), Params: json.RawMessage(`{"name":"list_venvs","arguments":{}}`)})
	result := out.String()
	if !strings.Contains(result, "-32602") || !strings.Contains(result, `"isError":true`) || !strings.Contains(result, `"id":3`) {
		t.Fatalf("unexpected tool-call output: %s", result)
	}
}

func TestServerHelpers(t *testing.T) {
	s := NewServer(manager.New(t.TempDir()))
	if s.in == nil || s.out == nil || s.log == nil {
		t.Fatal("NewServer left fields uninitialized")
	}
	out := &strings.Builder{}
	s.out = out
	s.writeResult(nil, map[string]any{})
	if out.Len() != 0 {
		t.Fatal("notification result should not be written")
	}

	args := map[string]any{"items": []any{"one", 2, "three"}, "bad": "value"}
	items := strSlice(args, "items")
	if strings.Join(items, ",") != "one,three" || strSlice(args, "bad") != nil {
		t.Fatalf("unexpected strSlice results: %v", items)
	}
}

func TestDispatchValidationBranches(t *testing.T) {
	s, _ := newTestServer(t)
	if _, err := s.dispatch("run_in_venv", map[string]any{"name": "v"}); err == nil {
		t.Fatal("run_in_venv should require a command")
	}
	if _, err := s.dispatch("exec_ephemeral", map[string]any{"command": []any{"true"}, "sandbox": true}); err == nil {
		t.Fatal("MCP sandbox should report unsupported capture")
	}
	if _, err := s.dispatch("describe_venv", map[string]any{"name": "missing"}); err == nil {
		t.Fatal("describe_venv should reject missing venv")
	}
}

func TestPolicyBlocksBeforeDispatch(t *testing.T) {
	dir := t.TempDir()
	venv := filepath.Join(dir, "keep")
	os.MkdirAll(venv, 0o755)
	os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("home = test\n"), 0o644)
	policy, _ := NewPolicy("safe", nil)
	out := &strings.Builder{}
	s := &Server{mgr: manager.New(dir), out: out, log: &strings.Builder{}, policy: policy}
	s.handleToolCall(rpcRequest{
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`{"name":"remove_venv","arguments":{"name":"keep"}}`),
	})
	if !strings.Contains(out.String(), "confirm=true") {
		t.Fatalf("expected policy denial, got %s", out)
	}
	if _, err := os.Stat(venv); err != nil {
		t.Fatalf("policy denial still executed removal: %v", err)
	}
}

func TestVisibleToolCatalogRespectsPolicy(t *testing.T) {
	policy, _ := NewPolicy("read-only", []string{"list_venvs", "doctor"})
	s := &Server{policy: policy}
	tools := s.visibleToolCatalog()
	if len(tools) != 2 || tools[0].Name != "list_venvs" || tools[1].Name != "doctor" {
		t.Fatalf("unexpected visible tools: %+v", tools)
	}
}

func TestDispatchRegistryAndSnapshotDiff(t *testing.T) {
	s, dir := newTestServer(t)
	venv := filepath.Join(dir, "v")
	os.MkdirAll(filepath.Join(venv, ".venv-manager", "snapshots"), 0o755)
	os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("home = test\n"), 0o644)
	snapshotDir := filepath.Join(venv, ".venv-manager", "snapshots")
	os.WriteFile(filepath.Join(snapshotDir, "a.txt"), []byte("one==1\n"), 0o644)
	os.WriteFile(filepath.Join(snapshotDir, "b.txt"), []byte("one==2\ntwo==1\n"), 0o644)

	out, err := s.dispatch("diff_snapshots", map[string]any{"name": "v", "from_snapshot_id": "a", "to_snapshot_id": "b"})
	if err != nil || !strings.Contains(out, `"two==1"`) || !strings.Contains(out, `"changed"`) {
		t.Fatalf("unexpected diff output: %s %v", out, err)
	}
	out, err = s.dispatch("list_registry", nil)
	if err != nil || !strings.Contains(out, `"name": "v"`) {
		t.Fatalf("unexpected registry output: %s %v", out, err)
	}
	out, err = s.dispatch("set_registry_metadata", map[string]any{"name": "v", "project": dir, "tags": []any{"test"}})
	if err != nil || !strings.Contains(out, `"test"`) {
		t.Fatalf("unexpected metadata output: %s %v", out, err)
	}
}
