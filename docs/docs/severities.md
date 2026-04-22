# Severities and gates

Aegis normalizes every finding to one of three severities:

| Severity | Meaning                                                          |
| -------- | ---------------------------------------------------------------- |
| `error`  | Must be fixed. Default gate is zero tolerance.                   |
| `warn`   | Should be fixed. Default gate allows a small backlog.            |
| `info`   | Informational. Never fails the gate by default.                  |

Scanners often emit richer severity scales (`critical`, `high`, `medium`, `low`, `style`, `performance`). Each adapter maps these down to the three Aegis severities — see the individual [scanner pages](scanners/index.md) for the exact mapping.

## Overriding severity

You can floor (raise) a check's minimum severity, or downgrade findings in specific paths via `warn_paths`.

### Floor at config level

```yaml
checks:
  secrets:
    severity_floor: error     # secrets are always at least error-level
```

A scanner emitting `warn` for a secret will be upgraded to `error` before the gate sees it.

### Downgrade via `warn_paths`

```yaml
paths:
  warn_paths:
    - "test/**"
    - "examples/**"
```

Findings inside these paths are downgraded one severity level (`error` → `warn`, `warn` → `info`). `info` findings in warn paths are dropped entirely.

### Per-rule ignores

See [baseline + allowlist](baseline-allowlist.md) and [inline ignores](#inline-ignores).

## The gate

The gate runs after filtering and deciding final severities. Configure it with thresholds:

```yaml
gate:
  fail_on:
    error: 0
    warn: 10
    info: -1
```

Rules:

- `-1` means **never fail on this severity**.
- `0` means **any finding of this severity fails**.
- A positive integer `N` means **more than N findings fail** (so `N` findings pass).

The gate evaluates top-down: if the `error` threshold is exceeded, the gate fails even if `warn` and `info` are fine.

## Inline ignores

Single-line and block-level ignores can be embedded in code with magic comments:

```go
// aegis:ignore rule=G101 reason="hardcoded for test fixture"
var testPassword = "hunter2"
```

```python
# aegis:ignore scanner=bandit rule=B105 reason="placeholder, see issue #42"
PASSWORD = "x"
```

An inline ignore **must** include a `reason`; Aegis refuses the directive without it. This is part of the [override model](operations-override.md) — every suppression leaves an audit trail.

## Quick reference

| Situation                                        | Where to configure                |
| ------------------------------------------------ | --------------------------------- |
| "This file is legitimately full of findings."    | `paths.exclude`                   |
| "This directory is low-risk; warn only."         | `paths.warn_paths`                |
| "We will fix these later but not block commits." | [`baseline`](baseline-allowlist.md#baseline) |
| "This specific finding is a false positive."     | [`allowlist`](baseline-allowlist.md#allowlist) |
| "Just this one line."                            | Inline `aegis:ignore` comment     |
| "Never allow anyone to disable secrets check."   | `protect_secrets: true` (default) |
