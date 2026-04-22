# osv-scanner

**Upstream:** [github.com/google/osv-scanner](https://github.com/google/osv-scanner)
**Check:** `dependencies`
**Stacks:** `go`, `npm`, `python`, and others osv-scanner supports.

## What it does

`osv-scanner` queries the [OSV](https://osv.dev) database for known-vulnerable versions of packages listed in a lockfile. It is the default `dependencies` check.

## How Aegis invokes it

```text
osv-scanner \
  --lockfile=<detected lockfile> \
  --format=json \
  --no-resolve
```

Aegis picks the lockfile automatically based on stack:

| Stack    | Lockfile                                       |
| -------- | ---------------------------------------------- |
| `go`     | `go.sum`                                       |
| `npm`    | `package-lock.json` or `pnpm-lock.yaml`        |
| `python` | `poetry.lock`, `requirements.txt`, etc.        |

If multiple lockfiles exist, osv-scanner is invoked once per lockfile.

## Severity mapping

| OSV severity score (CVSS 3) | Aegis severity |
| --------------------------- | -------------- |
| ≥ 9.0 (Critical)            | `error`        |
| 7.0–8.9 (High)              | `error`        |
| 4.0–6.9 (Medium)            | `warn`         |
| < 4.0 (Low)                 | `info`         |
| unscored                    | `warn`         |

## Configuration

```yaml
scanners:
  osv-scanner:
    path: osv-scanner
    version: "1.9.0"
    sha256:
      linux/amd64: "…"
```

## Performance

`osv-scanner` hits a network endpoint by default. On pre-commit this can be too slow for small repos. Practical pattern:

```yaml
hooks:
  pre_commit:
    checks: [secrets, lint, format]      # fast local stuff only
  pre_push:
    checks: [dependencies, malicious_code] # network-dependent stuff
```

## Offline mode

Pass `--offline` (osv-scanner 1.9+) to use an embedded database. Aegis will pass this through when `scanners.osv-scanner.offline: true` is set in `aegis.yaml`.
