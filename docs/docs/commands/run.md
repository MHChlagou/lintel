# `aegis run`

The main command. Executes the [nine-stage pipeline](../pipeline.md) and emits a report.

## Usage

```bash
aegis run [flags]
```

## Flags

| Flag            | Default  | Purpose                                                 |
| --------------- | -------- | ------------------------------------------------------- |
| `--hook`        | none     | Run in hook mode: `pre-commit` or `pre-push`            |
| `--check`       | all      | Comma-separated checks to run: `secrets,malicious_code,dependencies,lint,format` |
| `--override`    | false    | Run with the gate disabled; requires `--reason`         |
| `--reason`      | none     | Audit reason for override (min 8 chars)                 |
| `--dry-run`     | false    | Validate config + print effective scope; do not invoke scanners |
| `--output`      | pretty   | `pretty` | `json` | `sarif` | `junit`                  |

Global flags (`--config`, `--repo`, `--verbose`, …) also apply. See the [CLI index](index.md#global-flags).

## Hook mode vs. working-tree mode

- `--hook pre-commit`: scope is `git diff --cached --name-only` (staged files).
- `--hook pre-push`: scope is the commits being pushed, via `git rev-list <remote>..<local>`.
- No `--hook` flag: scope is the modified working tree — equivalent to `git status --porcelain` filtered for tracked files.

## Restricting checks

```bash
aegis run --check lint,format          # run only lint and format
aegis run --check secrets              # secrets only — useful for targeted debugging
```

## `fmt` shortcut

```bash
aegis fmt      # equivalent to: aegis run --check format
```

Because formatters are the fastest check, `aegis fmt` is a useful quick sanity pass.

## Dry-run

```bash
aegis run --dry-run
```

Prints the fully merged config, the detected stacks, the staged file list, and the scanners that **would** run, then exits. Useful for debugging config precedence or CI environments.

## Examples

```bash
# Standard local run.
aegis run

# CI run with JSON output for downstream tools.
aegis run --output json | tee aegis-report.json

# Scope to secrets + dependencies.
aegis run --check secrets,dependencies

# Emergency commit with an override + mandatory reason.
aegis run --override --reason "rotating creds, ticket SEC-1234"
```

## Exit codes

| Code | Meaning                                         |
| ---- | ----------------------------------------------- |
| 0    | Gate passed                                     |
| 1    | Gate failed                                     |
| 2    | Config or CLI error                             |
| 3    | Scanner binary missing or hash mismatch         |
| 4    | Scanner crashed or timed out                    |
