# ruff

**Upstream:** [docs.astral.sh/ruff](https://docs.astral.sh/ruff/)
**Checks:** `lint`, `format`
**Stacks:** `python`

## What it does

Ruff is a fast Python linter and formatter written in Rust. Lintel uses it for both `lint` and `format` on `python` stack files.

## How Lintel invokes it

Lint:

```text
ruff check --output-format=json <staged files>
```

Format check:

```text
ruff format --check <staged files>
```

## Severity mapping

| Ruff severity | Lintel severity |
| ------------- | -------------- |
| `error`       | `error`        |
| `warning`     | `warn`         |

Ruff does not emit a distinct `info` level; any rule classified as informational in your `pyproject.toml` becomes an Lintel `info`.

## Configuration

```yaml
scanners:
  ruff:
    path: ruff
    version: "0.8.0"
    sha256:
      linux/amd64: "…"
```

Rule selection lives in `pyproject.toml` under `[tool.ruff]` - Lintel does not intermediate.

## Common setup

```toml
# pyproject.toml
[tool.ruff]
target-version = "py311"
line-length = 100

[tool.ruff.lint]
select = ["E", "F", "I", "B", "UP", "S"]     # S = bandit-style security rules
```

The `S` rules overlap with dedicated security scanners but are cheap - keep them on unless they duplicate findings from another check.
