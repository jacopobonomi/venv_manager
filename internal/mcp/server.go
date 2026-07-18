// Package mcp implements a minimal Model Context Protocol server over stdio,
// exposing venv-manager operations as MCP tools that agentic clients
// (Claude Desktop, Cursor, Zed, etc.) can invoke natively.
//
// Protocol: JSON-RPC 2.0, newline-delimited over stdin/stdout.
// Reference: https://modelcontextprotocol.io/
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jacopobonomi/venv-manager/internal/manager"
)

const protocolVersion = "2024-11-05"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server wraps a Manager and speaks MCP.
type Server struct {
	mgr *manager.Manager
	in  *bufio.Reader
	out io.Writer
	log io.Writer
}

// NewServer builds an MCP server that reads from stdin and writes to stdout.
func NewServer(mgr *manager.Manager) *Server {
	return &Server{
		mgr: mgr,
		in:  bufio.NewReader(os.Stdin),
		out: os.Stdout,
		log: os.Stderr,
	}
}

// Serve runs the request loop until stdin closes.
func (s *Server) Serve() error {
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.writeErr(nil, -32700, "parse error: "+err.Error())
			continue
		}
		s.handle(req)
	}
}

func (s *Server) handle(req rpcRequest) {
	switch req.Method {
	case "initialize":
		s.writeResult(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "venv-manager", "version": "0.1"},
		})
	case "notifications/initialized":
		// no response for notifications
	case "tools/list":
		s.writeResult(req.ID, map[string]any{"tools": toolCatalog()})
	case "tools/call":
		s.handleToolCall(req)
	case "ping":
		s.writeResult(req.ID, map[string]any{})
	default:
		if len(req.ID) > 0 {
			s.writeErr(req.ID, -32601, "method not found: "+req.Method)
		}
	}
}

func (s *Server) writeResult(id json.RawMessage, result any) {
	if len(id) == 0 {
		return
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	b, _ := json.Marshal(resp)
	fmt.Fprintln(s.out, string(b))
}

func (s *Server) writeErr(id json.RawMessage, code int, msg string) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
	b, _ := json.Marshal(resp)
	fmt.Fprintln(s.out, string(b))
}

func toolCatalog() []toolDef {
	strProp := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	arrStr := func(desc string) map[string]any {
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
	}
	return []toolDef{
		{
			Name:        "list_venvs",
			Description: "List all Python virtual environments managed by venv-manager.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "create_venv",
			Description: "Create a new virtual environment with an optional Python version (e.g. '3.12').",
			InputSchema: map[string]any{
				"type":       "object",
				"required":   []string{"name"},
				"properties": map[string]any{"name": strProp("venv name"), "python_version": strProp("optional python version")},
			},
		},
		{
			Name:        "remove_venv",
			Description: "Delete a virtual environment.",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"name"},
				"properties": map[string]any{"name": strProp("venv name")},
			},
		},
		{
			Name:        "describe_venv",
			Description: "Return a full JSON description of a venv: python version, packages, size, activation commands, freeze hash.",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"name"},
				"properties": map[string]any{"name": strProp("venv name")},
			},
		},
		{
			Name:        "install_packages",
			Description: "Install packages into a venv. Provide either 'packages' (list) or 'requirements_file' (path).",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"name"},
				"properties": map[string]any{
					"name":              strProp("venv name"),
					"packages":          arrStr("pip package specifiers"),
					"requirements_file": strProp("path to requirements.txt"),
				},
			},
		},
		{
			Name:        "run_in_venv",
			Description: "Run a command inside a venv without activating it. Returns combined stdout+stderr.",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"name", "command"},
				"properties": map[string]any{
					"name":    strProp("venv name"),
					"command": arrStr("command and args, e.g. ['python','-c','print(1)']"),
				},
			},
		},
		{
			Name:        "exec_ephemeral",
			Description: "Create a temporary venv, install packages, run a command, then delete the venv. Great for running AI-generated code in isolation.",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"command"},
				"properties": map[string]any{
					"packages":       arrStr("pip packages to install"),
					"python_version": strProp("optional python version"),
					"command":        arrStr("command and args"),
					"sandbox":        map[string]any{"type": "boolean", "description": "run in OS-level sandbox (macOS/Linux only)"},
				},
			},
		},
		{
			Name:        "doctor",
			Description: "Report environment health: available python versions, uv presence, broken venvs.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

func (s *Server) handleToolCall(req rpcRequest) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.writeErr(req.ID, -32602, "invalid params: "+err.Error())
		return
	}

	out, err := s.dispatch(p.Name, p.Arguments)
	if err != nil {
		s.writeResult(req.ID, toolResult{
			Content: []toolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
		return
	}
	s.writeResult(req.ID, toolResult{Content: []toolContent{{Type: "text", Text: out}}})
}

func str(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func strSlice(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func toJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func (s *Server) dispatch(name string, args map[string]any) (string, error) {
	switch name {
	case "list_venvs":
		venvs, err := s.mgr.List()
		if err != nil {
			return "", err
		}
		return toJSON(venvs), nil

	case "create_venv":
		n := str(args, "name")
		if n == "" {
			return "", fmt.Errorf("name is required")
		}
		if err := s.mgr.Create(n, str(args, "python_version")); err != nil {
			return "", err
		}
		return fmt.Sprintf("created venv %q", n), nil

	case "remove_venv":
		n := str(args, "name")
		if err := s.mgr.Remove(n); err != nil {
			return "", err
		}
		return fmt.Sprintf("removed venv %q", n), nil

	case "describe_venv":
		d, err := s.mgr.Describe(str(args, "name"))
		if err != nil {
			return "", err
		}
		return toJSON(d), nil

	case "install_packages":
		n := str(args, "name")
		if rf := str(args, "requirements_file"); rf != "" {
			if err := s.mgr.Install(n, rf); err != nil {
				return "", err
			}
			return "installed from " + rf, nil
		}
		pkgs := strSlice(args, "packages")
		if len(pkgs) == 0 {
			return "", fmt.Errorf("provide packages or requirements_file")
		}
		return s.installPackages(n, pkgs)

	case "run_in_venv":
		return s.runInVenv(str(args, "name"), strSlice(args, "command"))

	case "exec_ephemeral":
		sandbox, _ := args["sandbox"].(bool)
		return s.execEphemeral(strSlice(args, "packages"), str(args, "python_version"), strSlice(args, "command"), sandbox)

	case "doctor":
		return toJSON(s.mgr.Doctor()), nil
	}
	return "", fmt.Errorf("unknown tool: %s", name)
}
