# Agents Guide

This document provides context for AI agents working on this codebase.

## Important: Keep Docs in Sync

When making changes to this repo, **always update documentation** (README.md, this file) to reflect those changes immediately. Don't wait until the end.

## Project Overview

Euthanaisa is a simple Kubernetes cleanup utility. It:
1. Lists Kubernetes resources with the `euthanaisa.nais.io/kill-after` label
2. Parses the label value as a Unix timestamp (seconds since epoch)
3. Deletes resources where the timestamp has passed

## Architecture

```
cmd/euthanaisa/main.go    - Entry point, config, wiring
internal/
  client/                 - Kubernetes dynamic client wrapper
  euthanaiser/            - Core deletion logic + metrics
```

## Key Design Decisions

- **Minimal packages**: Only split code when there's a clear reason
- **Dynamic client**: Uses `k8s.io/client-go/dynamic` to work with any resource type
- **Label-based filtering**: Server-side filtering via label selector for efficiency
- **No mocks**: Tests use simple fakes defined in test files
- **Time injection**: `Euthanaiser.now` function allows deterministic testing
- **Standard library**: Uses `log/slog` for logging, `os.Getenv` for config

## Testing

Tests are in `internal/euthanaiser/euthanaiser_test.go`. They use:
- A `fakeClient` struct defined in the test file
- Fixed timestamps for deterministic behavior
- Table-driven tests

Run tests:
```bash
mise run test
# or
go test ./...
```

## Code Quality Tools

Via mise (preferred):
```bash
mise run check    # Run all checks
mise run fmt      # Format code
```

Or directly via Go tools:
```bash
go tool gofumpt -w .        # Format
go tool staticcheck ./...   # Static analysis
go tool deadcode ./...      # Dead code detection
go tool govulncheck ./...   # Vulnerability check
go tool gosec ./...         # Security check
```

## Building

```bash
mise run build              # Build binary
# or
go build ./cmd/euthanaisa

# Docker
docker build -t euthanaisa .
```

## Configuration

All via environment variables (see `cmd/euthanaisa/main.go`):
- `LOG_LEVEL`, `LOG_FORMAT` - Logging
- `RESOURCES_FILE` - Path to resources.yaml
- `PUSHGATEWAY_ENDPOINT` - If set, enables metrics push

## Common Tasks

### Adding a new resource type
Edit `hack/resources.yaml` (local dev) or Helm values - no code changes needed.

### Modifying deletion logic
Edit `internal/euthanaiser/euthanaiser.go`, specifically `shouldDelete()`.

### Adding metrics
Add to the `var` block in `internal/euthanaiser/euthanaiser.go`.
