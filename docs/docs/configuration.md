# The `lintel.yaml` file

Everything Lintel does is controlled by one file at `.lintel/lintel.yaml`. This page is a reference; for the rationale behind each field see the [full spec](reference/spec.md).

## Minimal example

```yaml
schema: 1

stacks:
  auto: true

checks:
  secrets:
    enabled: true
  malicious_code:
    enabled: true
  dependencies:
    enabled: true
  lint:
    enabled: true
  format:
    enabled: true

gate:
  fail_on:
    error: 0
    warn: 10
```

That is enough to run. Every other field has a sensible default - see [`defaults_spec.go`](https://github.com/MHChlagou/lintel/blob/main/internal/config/defaults_spec.go) for the full baked-in defaults.

## Top-level sections

| Section           | Purpose                                                             |
| ----------------- | ------------------------------------------------------------------- |
| `schema`          | Schema version. Current: `1`.                                       |
| `stacks`          | Which stacks to detect or pin.                                      |
| `checks`          | Per-check configuration (`secrets`, `malicious_code`, `dependencies`, `lint`, `format`). |
| `scanners`        | Per-scanner binary paths, versions, and SHA256 pins.                |
| `gate`            | Severity thresholds that decide pass/fail.                          |
| `paths`           | Include/exclude globs and `warn_paths` for soft-fail zones.         |
| `allowlist`       | Path-based or rule-based suppressions (see [baseline-allowlist](baseline-allowlist.md)). |
| `timeouts`        | Per-check and total timeouts.                                       |
| `concurrency`     | Parallelism settings (see [performance](performance.md)).           |
| `strict_versions` | If `true` (default), scanner version + SHA256 mismatches are fatal. |
| `protect_secrets` | If `true` (default), the secrets check cannot be disabled by hook bypass. |
| `hooks`           | Which Git hooks Lintel installs and what they dispatch to.           |

## Stacks

Stacks control which scanners are candidates for which files.

```yaml
stacks:
  auto: true                  # detect stacks from staged files
  include: [go, npm, python]  # restrict to these even if auto detects more
  exclude: [php]
```

When `auto: false`, you must list every stack explicitly under `include`.

## Checks

Each check can be individually `enabled` or disabled, have its severity floor raised, and select a preferred scanner when multiple are available for a stack.

```yaml
checks:
  secrets:
    enabled: true
    severity_floor: error       # downgrades are not allowed below this
  lint:
    enabled: true
    scanners:
      go: golangci-lint
      js: biome
      python: ruff
```

## Scanners

This section pins scanner identity. The defaults cover every shipped adapter; override only when you need to.

```yaml
scanners:
  gitleaks:
    path: gitleaks           # resolved via $PATH; absolute paths also allowed
    version: "8.18.2"
    sha256:
      linux/amd64: "e3b0c442…"
      darwin/arm64: "af9a7cbd…"
  golangci-lint:
    path: golangci-lint
    version: "1.61.0"
    sha256:
      linux/amd64: "1234abcd…"
```

!!! warning "Hash discipline"
    A scanner with a missing or mismatched `sha256` pin will be refused at runtime when `strict_versions: true`. This is the default and the whole point of the [supply-chain model](supply-chain.md). Do not turn it off to silence a hash error - update the pin via `lintel doctor --update-pins` (v1.1) or by hand from the upstream release checksum.

## Gates

```yaml
gate:
  fail_on:
    error: 0       # fail if any error-severity finding exists
    warn: 10       # fail if more than 10 warn-severity findings exist
    info: -1       # -1 means never fail on info
```

The gate is evaluated after all filtering (allowlist, baseline, inline ignores, warn_paths).

## Paths

```yaml
paths:
  include: ["**/*"]
  exclude:
    - "vendor/**"
    - "node_modules/**"
    - "**/*.generated.go"
  warn_paths:
    - "test/**"     # findings here are downgraded one severity level
```

Patterns use [doublestar](https://github.com/bmatcuk/doublestar) glob syntax, relative to the repo root.

## Environment variable overrides

Any config value can be overridden by an environment variable at runtime - useful for CI parameter tuning without editing YAML. See [environment variables](reference/env-vars.md) for the full list.

## Validating a config

```bash
lintel run --dry-run
# parses, validates, and prints the merged effective config
```

An invalid config exits with code **2** and a line-accurate error message.
