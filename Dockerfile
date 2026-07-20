# Multi-stage build for the venv-manager MCP server.
# Runtime image ships the Go binary plus a real python3 so that MCP tools
# (create_venv, exec_ephemeral, install_packages, …) actually work.

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/venv-manager ./cmd/venv-manager

FROM python:3.12-alpine
LABEL org.opencontainers.image.title="venv-manager"
LABEL org.opencontainers.image.source="https://github.com/jacopobonomi/venv_manager"
LABEL org.opencontainers.image.description="Python venv runtime for AI agents (MCP server on stdio)"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=build /out/venv-manager /usr/local/bin/venv-manager

# Glama introspects the server by starting it and sending an MCP initialize +
# tools/list. `venv-manager mcp` speaks JSON-RPC 2.0 on stdin/stdout with no
# arguments needed.
ENTRYPOINT ["venv-manager", "mcp"]
