# gitleaks

**Upstream:** [github.com/gitleaks/gitleaks](https://github.com/gitleaks/gitleaks)
**Check:** `secrets`
**Stacks:** all (stack-agnostic)

## What it does

`gitleaks` scans files for high-entropy strings and known token shapes, reporting matches with a rule id and a short excerpt. Aegis uses it as the default `secrets` check.

## How Aegis invokes it

```text
gitleaks detect \
  --source=<repo> \
  --no-git \
  --report-format=json \
  --report-path=<tmp> \
  --redact \
  --verbose=false \
  --log-level=error \
  <staged files>
```

- `--no-git`: Aegis has already identified the staged file set; we do not need gitleaks to re-scan history.
- `--redact`: matched values are redacted in the output. Aegis never exposes the raw value.
- Staged paths are appended as positional args so only changed files are scanned.

## Severity mapping

| Upstream        | Aegis severity |
| --------------- | -------------- |
| any match       | `error`        |

Every gitleaks hit is an error by default. Use `warn_paths` to downgrade inside `testdata/` or similar.

## Configuration

```yaml
scanners:
  gitleaks:
    path: gitleaks
    version: "8.18.2"
    sha256:
      linux/amd64: "…"
      darwin/arm64: "…"
```

## Custom rules

Gitleaks supports a `.gitleaks.toml` rules file. Aegis does not manage this file; place it at the repo root and gitleaks will pick it up automatically.

!!! warning "Rule file authenticity"
    Aegis verifies the gitleaks **binary** but not its rule pack. If you rely on a community rule pack, vendor it into your repo and review changes as you would any config.

## Common false positives

- Test fixtures with placeholder keys → exclude `testdata/**` via `paths.exclude` or add an allowlist entry.
- Base64-encoded blobs in generated code → exclude generated paths.
- Lockfile digests that trip entropy heuristics → add a rule-specific allowlist.
