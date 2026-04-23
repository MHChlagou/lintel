# Concurrency and performance

Lintel is designed to fit inside a developer's pre-commit wait budget - ideally under 2 seconds for a small staged change, under 10 for a large one. This page describes the knobs.

## Target budgets

| Invocation             | Repo size       | Target wall time |
| ---------------------- | --------------- | ---------------- |
| `pre-commit` (1–5 files) | any             | < 2 s            |
| `pre-commit` (50 files)  | medium          | < 10 s           |
| `pre-push`              | commits being pushed | < 30 s      |
| CI (full working tree) | any             | bounded by `timeouts.total` |

These are targets, not guarantees. A full dependency scan (`osv-scanner`) can dominate on npm monorepos with thousands of transitive dependencies.

## Parallelism

Each enabled check × stack spawns a goroutine. Up to `concurrency.max_parallel` run at once (defaults to `runtime.NumCPU()`).

```yaml
concurrency:
  max_parallel: 4         # explicit cap; set to 1 for deterministic CI logs
  scanner_nice: true      # on Unix, renice scanners to 10 (lower priority than your shell)
```

- **Independent.** Scanners never share files on disk or in memory. They read the staged file list and emit findings into a per-scanner buffer.
- **Bounded.** No scanner can spawn more workers than the cap, even when the scanner's own `--jobs` flag is higher.
- **Fair.** A slow scanner does not starve fast ones; they all start in the same wave.

## Timeouts

```yaml
timeouts:
  per_check: 30s          # kill any single scanner that exceeds this
  total: 120s             # kill everything if the whole run exceeds this
```

A timed-out scanner yields **exit 4** for that check but does not fail the other checks. The gate still runs on the findings that did complete - so a timeout does not silently pass the gate.

## Staged-file scoping

On a Git hook, Lintel passes only the staged file list to scanners that accept a file argument. For scanners that always scan the whole tree (for example, `osv-scanner` on `package-lock.json`), the file list is still computed so that the filter stage knows which findings originate in changed files.

This is the single largest performance win over "just run gitleaks" approaches: a 20-file commit in a 200k-file repo still finishes in seconds.

## Measured cost

`lintel run --verbose` prints per-scanner elapsed times. A typical breakdown on a Go service with a ~1k-line pre-commit change:

```
gofmt            80ms
golangci-lint    1.2s
gitleaks         180ms
osv-scanner      900ms  (go.sum unchanged, cached)
```

Total: ~1.3 s wall, since `golangci-lint` and `osv-scanner` overlap.

## Tuning tips

- **Move `dependencies` to `pre-push`.** Lockfiles change less often than code. A `hooks.pre_commit.checks` list that omits `dependencies`, combined with `hooks.pre_push.checks: [dependencies]`, keeps pre-commit snappy.
- **Exclude `vendor/`, `node_modules/`, generated code.** `paths.exclude` is cheap - it happens before any scanner runs.
- **Set `max_parallel: 2` on CI runners with shared CPU.** More parallelism is not always faster under contention.
- **Inspect `--verbose` output** before hypothesizing. Most "slow" reports turn out to be one specific scanner on one specific file.

## What Lintel does not do

- It does not cache scanner results between runs. A scanner re-reads the file on each invocation. Caching is a v2.0 roadmap item - until then, correctness wins over cleverness.
- It does not do incremental analysis. A file is either in scope or not; there is no "diff of findings" calculation at the scanner level.
