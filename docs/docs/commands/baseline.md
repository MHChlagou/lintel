# `aegis baseline`

Snapshot the current findings into `.aegis/baseline.json` so they are treated as "known" on future runs. See [baseline + allowlist](../baseline-allowlist.md) for the concept.

## Usage

```bash
aegis baseline [flags]
```

## Flags

| Flag           | Default | Purpose                                                          |
| -------------- | ------- | ---------------------------------------------------------------- |
| `--check`      | false   | Don't write; exit non-zero if the current findings would change the baseline |
| `--prune`      | false   | Remove baseline entries for findings that no longer exist        |
| `--output`     | `.aegis/baseline.json` | Where to write (rarely changed)                   |

## Typical workflows

### First adoption

```bash
aegis baseline      # snapshot everything currently found
git add .aegis/baseline.json
git commit -m "chore(aegis): seed baseline"
```

From now on, commits fail only on **new** findings.

### Checking the baseline in CI

```bash
aegis baseline --check
```

Exit 0 if the baseline matches current findings, exit 1 otherwise. This is the check that prevents a PR from accidentally regressing (adding new findings to the baseline instead of fixing them).

### Pruning after fixes

```bash
aegis baseline --prune
git add .aegis/baseline.json
git commit -m "chore(aegis): drop fixed findings from baseline"
```

`--prune` never adds entries; it only removes entries for findings that no longer exist. Use it after a cleanup PR.

## File format

`baseline.json` is a stable JSON schema:

```json
{
  "schema": 1,
  "generated_at": "2026-04-22T10:15:00Z",
  "entries": [
    {
      "fingerprint": "a31e…",
      "scanner": "gitleaks",
      "rule":    "generic-api-key",
      "file":    "testdata/example.go",
      "message": "…"
    }
  ]
}
```

`fingerprint` is the match key — see [first scan](../first-scan.md#anatomy-of-a-pretty-report).

## Exit codes

| Code | Meaning                                                      |
| ---- | ------------------------------------------------------------ |
| 0    | Baseline written (or `--check` matches)                      |
| 1    | `--check` detected a drift                                   |
| 2    | Config or CLI error                                          |
