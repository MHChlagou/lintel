# golangci-lint

**Upstream:** [golangci-lint.run](https://golangci-lint.run/)
**Check:** `lint`
**Stacks:** `go`

## What it does

`golangci-lint` is a wrapper around many Go linters (staticcheck, govet, errcheck, revive, …) with shared config and caching. Lintel uses it for the `lint` check on Go files.

## How Lintel invokes it

```text
golangci-lint run \
  --out-format=json \
  --no-config=${LINTEL_GOLANGCI_NO_CONFIG:-false} \
  --timeout=<per_check> \
  <package paths from staged files>
```

- Lintel translates staged file paths to Go package paths (`./internal/foo/...`), since `golangci-lint` is package-oriented.
- `--no-config` is off by default so your existing `.golangci.yml` is honored.

## Severity mapping

| golangci-lint severity | Lintel severity |
| ---------------------- | -------------- |
| `error`                | `error`        |
| `warning`              | `warn`         |
| `info`                 | `info`         |

Linters that don't emit an explicit severity are treated as `warn`.

## Configuration

Lintel pins the binary; rule selection is in your repo's `.golangci.yml` (or `golangci.yaml`, `.golangci.toml`).

```yaml
scanners:
  golangci-lint:
    path: golangci-lint
    version: "1.61.0"
    sha256:
      linux/amd64: "…"
```

## Common config

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosec          # security-focused linter; overlaps with opengrep but narrower
    - revive
  disable:
    - typecheck      # already handled by `go vet`
issues:
  exclude-use-default: false
```

## Performance

`golangci-lint` maintains its own cache (`$GOLANGCI_LINT_CACHE`). Preserve that cache across CI runs for a significant speedup - see the upstream CI examples.
