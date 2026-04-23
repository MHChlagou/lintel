# Your first scan

Once you have `lintel` on `$PATH` and at least one scanner installed, running a scan is one command. This page walks through the output in detail.

## Scenario

A small Go service with `gofmt` and `golangci-lint` installed. No hooks installed yet - we will invoke Lintel directly.

```bash
cd my-service
lintel init
lintel run
```

## Anatomy of a pretty report

```text
Lintel v0.1.0  schema=1

Scope: 12 staged files (2 stacks: go, shell)

gofmt            1 finding (0 error, 1 warn)  200ms
golangci-lint    3 findings (1 error, 2 warn) 1.4s
shellcheck       0 findings                   120ms

--- findings ---

error  golangci-lint  errcheck             internal/server/handler.go:42
       error return value of `db.Close()` is not checked
       -> wrap the call or assign to `_`

warn   golangci-lint  staticcheck (SA1019) internal/server/handler.go:88
       sql.ErrNoRows comparison should use errors.Is
       -> errors.Is(err, sql.ErrNoRows)

warn   golangci-lint  govet                internal/server/auth.go:17
       composite literal uses unkeyed fields

warn   gofmt          -                    internal/server/auth.go
       file is not gofmt-compliant (run `gofmt -w`)

1 error  3 warnings  scanned in 1.7s

✗ gate failed: 1 error (threshold: error=0)
exit 1
```

- The **scope** line reports how many files were handed to Lintel and which stacks they belong to.
- Each scanner is listed with a finding count, a severity breakdown, and its own elapsed time. Slow scanners stand out.
- **Findings** are grouped by severity (`error` > `warn` > `info`), then by scanner, then by file, deterministically.
- The **gate** line is the decision: in the default config, any single `error` finding fails the commit. Configure this in `lintel.yaml` under `gate`.

## JSON output

Use `--output json` for CI or machine consumption. The shape is stable and documented in the [specification](reference/spec.md#112-json).

```bash
lintel run --output json | jq '.findings[0]'
```

```json
{
  "check": "lint",
  "scanner": "golangci-lint",
  "rule": "errcheck",
  "severity": "error",
  "file": "internal/server/handler.go",
  "line": 42,
  "column": 12,
  "message": "error return value of `db.Close()` is not checked",
  "remediation": "wrap the call or assign to `_`",
  "fingerprint": "a31e…"
}
```

`fingerprint` is a stable hash across (scanner, rule, file, line, normalized message) - that is what [baselines](baseline-allowlist.md) use to recognize a pre-existing finding between runs.

## Exit codes at a glance

| Code | Meaning                                  |
| ---- | ---------------------------------------- |
| 0    | Gate passed, no action needed.           |
| 1    | Gate failed.                             |
| 2    | Config or CLI error (bad flag, bad yaml). |
| 3    | Scanner binary missing or hash mismatch. |
| 4    | Scanner crashed or timed out.            |

Full breakdown: [exit codes reference](reference/exit-codes.md).

## What's next

- [Wire scanners into Git hooks](commands/install.md) so every commit is scanned.
- [Add a baseline](baseline-allowlist.md) if your repo already has findings you want to accept now and fix later.
- [Run Lintel in CI](ci-integration.md).
