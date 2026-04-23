# Lintel

*The shield your commits pass through.*

[![CI](https://github.com/MHChlagou/lintel/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/MHChlagou/lintel/actions/workflows/ci.yml)
[![CodeQL](https://github.com/MHChlagou/lintel/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/MHChlagou/lintel/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MHChlagou/lintel)](https://goreportcard.com/report/github.com/MHChlagou/lintel)
[![Go Reference](https://pkg.go.dev/badge/github.com/MHChlagou/lintel.svg)](https://pkg.go.dev/github.com/MHChlagou/lintel)
[![License: MIT](https://img.shields.io/badge/license-MIT-yellow.svg)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/MHChlagou/lintel)](go.mod)
[![Docs](https://img.shields.io/badge/docs-mhchlagou.github.io%2Flintel-blue)](https://MHChlagou.github.io/lintel/)

A single-binary, shift-left security orchestrator that runs as a Git hook (and in CI). One declarative spec file, zero runtime dependencies, and the best-in-class OSS scanners you already trust.

**[Documentation](https://MHChlagou.github.io/lintel/)** · **[Changelog](CHANGELOG.md)** · **[Contributing](CONTRIBUTING.md)** · **[Security](SECURITY.md)**

---

## Introduction

Most teams want shift-left security - secrets, SAST, SCA, lint, format - on every commit. The existing landscape forces a choice:

- **Glue it yourself.** Husky + lint-staged + N tools, each with its own config. Brittle, drifts between repos, breaks when anyone upgrades anything.
- **Adopt a SaaS.** Expensive, opaque, and almost always a "black box" running rules you can't audit.

Lintel is a third path: **one Go binary** that coordinates the scanners you choose, driven by one YAML file, with a tight CLI and strong supply-chain hygiene built in.

---

## What Lintel Is (and Isn't)

Lintel **is**:

- A single static binary (Linux/macOS/Windows on amd64/arm64).
- A declarative spec (`lintel.yaml`) describing stacks, scanners, gates, and ignores.
- A Git hook manager with an opinionated CLI: `init`, `install`, `run`, `doctor`, `baseline`.
- A reporter that normalizes heterogeneous scanner output into one `Finding` shape, emitted as pretty text or JSON.

Lintel **is not**:

- A scanner. It contains no regexes, rule engines, or CVE databases.
- A package manager. You bring your own scanner binaries (v1). Lintel verifies them.
- A CI server. It runs *in* CI by being invoked from one.
- A secret vault, SBOM platform, or compliance dashboard.

Currently wired engines: `gitleaks` (secrets), `opengrep` (SAST), `osv-scanner` (SCA), `biome`, `ruff`, `golangci-lint`, `gofmt`, `shellcheck`.

---

## How to Use

### Install the binary

**macOS / Linux** (pipe the install script):

```bash
curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh
```

**Windows** (PowerShell):

```powershell
iwr https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.ps1 -UseBasicParsing | iex
```

**Go toolchain** (for development builds):

```bash
go install github.com/MHChlagou/lintel/cmd/lintel@latest
```

The install scripts download the binary for your OS/arch, verify its SHA256, optionally verify the Sigstore signature (if `cosign` is on PATH), and place `lintel` on your `$PATH`. See [installation](https://mhchlagou.github.io/lintel/installation/) for pinning a version, alternate install paths, and manual verification.

Verify:

```bash
lintel version
# lintel v0.1.0  schema=1  commit=abc123  built=2026-04-21T…
```

### Bootstrap a repo

```bash
cd my-repo
lintel init          # writes .lintel/lintel.yaml with secure defaults
lintel install       # writes .git/hooks/{pre-commit, commit-msg, pre-push}
lintel doctor        # verifies scanner binaries + sha256 hashes
```

`lintel init` never overwrites an existing spec (pass `--force` to replace). `lintel install` never overwrites a foreign hook (pass `--force` to replace).

### Install the scanners (your choice of engines)

Lintel verifies each scanner's sha256 before running it. The default spec names them; you install them:

```bash
# Examples - use the real releases for your platform
mkdir -p ~/.lintel/bin
curl -fsSL https://github.com/gitleaks/gitleaks/releases/download/v8.28.0/... -o ~/.lintel/bin/gitleaks && chmod +x ~/.lintel/bin/gitleaks
# …same for opengrep, osv-scanner, biome/ruff/golangci-lint as your stack needs
```

Then pin their hashes in `lintel.yaml` under `binaries.<name>.sha256.<os>_<arch>`. `lintel doctor` tells you what's missing.

### Run

Hooks dispatch automatically on `git commit` / `git push`. Manual invocations:

```bash
lintel run                        # run every enabled check
lintel run --hook pre-commit      # run the pre-commit hook's configured set
lintel run --check secrets        # run a single check
lintel run --output json          # machine-readable output
lintel fmt                        # shortcut for --check format
```

### Baseline and ignores (for adopting on a legacy repo)

```bash
lintel baseline                    # snapshot current findings into .lintel/baseline.json
lintel ignore <rule-id> \          # add to .lintel/allowlist.yaml
  --path 'src/legacy/**' --reason "pending rewrite, JIRA-1021"
```

Inline ignores (per-language comments on the flagged or preceding line):

```go
// lintel:ignore-secret  reason="test fixture, not a real key"
testKey := "AKIAFAKEFAKEFAKE1234"
```

```python
# lintel:ignore-rule=SQLi.raw-concat  reason="hardened elsewhere"
```

A reason is **mandatory** - an inline ignore without one is itself a finding.

### Emergency bypass

```bash
LINTEL_SKIP=secrets,lint \
LINTEL_REASON="hotfix for incident #9912" \
  git commit -m "..."
```

Skipped checks are logged to `.lintel/overrides.log` with user, timestamp, commit hash, and reason. If `override.protect_secrets: true`, the `secrets` check is always enforced, even with `LINTEL_SKIP=all`.

### Uninstall

```bash
lintel uninstall          # removes only hooks carrying the lintel marker
```

---

## How to Configure

All behavior lives in `.lintel/lintel.yaml`. `lintel init` writes a secure-by-default template. Every key is optional except `version`.

### Minimal example

```yaml
version: 1

checks:
  secrets:        { enabled: true, mode: block }
  malicious_code: { enabled: true, mode: block }
  dependencies:   { enabled: true, mode: block, block_severity: [CRITICAL, HIGH] }
  lint:           { enabled: true, mode: warn }
  format:         { enabled: true, mode: check }

hooks:
  pre-commit: { checks: [secrets, malicious_code, lint, format] }
  pre-push:   { checks: [secrets, dependencies], fail_fast: true }
```

### Key sections

**`binaries`** - declare the scanners Lintel may execute. Each needs a `version`, optional `path`, per-platform `sha256`, and `install_hint`:

```yaml
binaries:
  gitleaks:
    command: gitleaks
    version: "8.28.0"
    sha256:
      linux_amd64:  "abc123…"
      darwin_arm64: "def456…"
    install_hint: "https://github.com/gitleaks/gitleaks/releases/tag/v8.28.0"
```

Resolution order: `binaries.X.path` → `$LINTEL_BIN_DIR/X` → `~/.lintel/bin/X` → `$PATH`. Sha256 is verified on every run.

**`checks`** - one subtree per check. Each supports `enabled`, `mode` (`block`/`warn`/`off`; `fix`/`check`/`off` for format), an `engine` where applicable, and check-specific knobs like `warn_paths`, `severity_threshold`, `block_severity`, `ignore_cves`.

**`scope`** - which files are scanned. Staged-only by default; globbed `exclude_paths` apply globally; `full_scan_for: [dependencies]` forces full-tree for specified checks.

**`hooks`** - map each hook (`pre-commit`, `pre-push`, `commit-msg`) to a check list and `fail_fast` flag.

**`output`** - `format: pretty|json|sarif|junit`, `group_by`, `color`, `verbosity`.

**`override`** - bypass controls: `env_var` (default `LINTEL_SKIP`), `require_reason`, `log_file`, `protect_secrets`, `allow_no_verify`.

**`performance`** - `parallel: auto|<int>`, per-check and total timeouts, cache settings.

**`strict_versions: true`** - refuses to run any binary that has no sha256 declared for the current platform. Recommended.

The full schema is documented inline in [`docs/docs/reference/spec.md`](docs/docs/reference/spec.md), and rendered at the [documentation site](https://MHChlagou.github.io/lintel/configuration/).

### Environment variables

| Variable         | Purpose                                                   |
|------------------|-----------------------------------------------------------|
| `LINTEL_SKIP`     | Comma-separated checks to skip (e.g. `secrets,lint`, or `all`) |
| `LINTEL_REASON`   | Required justification when `override.require_reason: true` |
| `LINTEL_BIN_DIR`  | Extra dir searched for scanner binaries                    |
| `LINTEL_CONFIG`   | Alternate path to the spec file                            |
| `LINTEL_CACHE_DIR`| Override cache directory                                   |
| `LINTEL_NO_COLOR` | Disable colored output (`NO_COLOR` also honored)           |

---

## How It Works

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    lintel (single binary)                 │
│                                                          │
│  CLI → Config → Detector → Resolver → Runner             │
│                                 (sha256-verified exec)   │
│                                 ↓                        │
│                        ┌───────────────────────┐         │
│                        │ gitleaks  opengrep    │         │
│                        │ osv-scanner  biome    │         │
│                        │ ruff  golangci-lint … │         │
│                        └───────────────────────┘         │
│                                 ↓                        │
│             Normalize → Filter → Gate → Report → Exit    │
└──────────────────────────────────────────────────────────┘
```

### Execution pipeline

When you `git commit`, the pre-commit hook invokes `lintel run --hook pre-commit`, which runs nine stages:

1. **Load spec.** Parse `.lintel/lintel.yaml`, apply defaults, validate (e.g., reject relative paths in `binaries.X.path`).
2. **Detect staged files.** `git diff --cached --name-only -z`, NUL-split for odd paths.
3. **Detect stacks.** Explicit `project.type` first; otherwise walk manifests (`package.json` + lock variant, `pom.xml`, `go.mod`, `pyproject.toml`, `Cargo.toml`, …); extension-count fallback if zero manifests match.
4. **Resolve binaries.** Locate each scanner via the resolution order; read-into-sha256-stream; compare to spec's expected hash for `<os>_<arch>`. Mismatch → refuse to run. No hash + `strict_versions: true` → refuse. Memoized for the rest of the process.
5. **Plan.** Drop disabled/off checks; apply `LINTEL_SKIP` (refused for `secrets` if `protect_secrets: true`); write to `overrides.log` if skipping.
6. **Execute, in parallel.** Each check runs in its own goroutine, bounded by `performance.parallel` and a per-check `context.WithTimeout`. `fail_fast: true` cancels sibling contexts on the first blocking finding.
7. **Normalize.** Each adapter decodes its scanner's JSON or text output into the common `Finding` shape: `Check`, `RuleID`, `Severity`, `File`, `Line`, `Message`, `Snippet`, `FixSuggest`, `Engine`.
8. **Filter + gate.** Apply allowlist → baseline → `warn_paths` (demote) → inline ignores → per-check severity threshold → mode. Decide `Blocking` per finding.
9. **Report.** Sort findings deterministically by `(check, file, line, rule_id)`, render pretty or JSON, write to `output.report_file` if configured, exit with the right code.

### Exit codes

| Code | Meaning                                              |
|------|------------------------------------------------------|
| 0    | Success - no blocking findings                       |
| 1    | Blocking findings detected                           |
| 2    | Configuration error                                  |
| 3    | Binary resolution or verification failure            |
| 4    | Scanner crashed or timed out                         |
| 5    | Internal error                                       |
| 130  | Interrupted (SIGINT)                                 |

### Supply-chain model

Lintel is itself a supply-chain-sensitive tool (it executes external binaries at developer privilege). Mitigations:

- **Mandatory sha256** on every external binary, on every run.
- **No auto-download by default.** You install scanners; Lintel verifies them.
- **Spec path limits.** `binaries.X.path` must be absolute or home-relative; relative paths into the repo are rejected (defense against a malicious PR slipping in a vendored binary).
- **Signed Lintel releases.** Keyless cosign signatures on every artifact.
- **Capability minimization.** No outbound network except when `dependencies.offline: false` (OSV DB fetch) or `lintel install <tool>` (explicit opt-in, v1.1+).
- **No telemetry.**

### Concurrency + determinism

Scanners run concurrently for speed. Findings are sorted before output, so parallelism never changes the report. The `-race` flag in CI guards against regressions.

### Performance targets

| Metric                                               | Budget        |
|------------------------------------------------------|---------------|
| `lintel run --hook pre-commit` p95 on 10k staged LOC  | < 5 seconds   |
| sha256 verify (per binary, cached)                   | < 50 ms       |
| Spec parse + validate                                | < 20 ms       |
| Cold start                                           | < 150 ms      |

---

## Development

```bash
make ci          # vet + gofmt check + test -race + build
make test-race   # tests with the race detector
make smoke       # build and run `lintel init` in a temp repo
make build       # binary → bin/lintel
```

Cross-compile:

```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o dist/lintel ./cmd/lintel
```

CI runs on every push and PR: matrix tests on Linux/macOS/Windows, golangci-lint, cross-platform builds with sha256 sidecars, govulncheck, CodeQL static analysis, and an end-to-end self-smoke against the built binary. See [`.github/workflows/`](.github/workflows/).

---

## Project layout

```
lintel/
├── cmd/lintel/        # main package
├── internal/         # all logic (cli, config, checker, runner, filter, gate, …)
├── testdata/         # canned scanner outputs for adapter tests
├── examples/         # reference lintel.yaml configs for common stacks
├── docs/             # MkDocs-Material documentation site
└── .github/          # workflows, issue/PR templates, CODEOWNERS
```

See [`docs/docs/architecture.md`](docs/docs/architecture.md) for the dependency map between packages.

---

## Community

- **Documentation:** <https://MHChlagou.github.io/lintel/>
- **Discussions:** [GitHub Discussions](https://github.com/MHChlagou/lintel/discussions) - questions, setup help, roadmap.
- **Issues:** [GitHub Issues](https://github.com/MHChlagou/lintel/issues) - bugs, feature requests.
- **Security reports:** [GitHub Private Vulnerability Reporting](https://github.com/MHChlagou/lintel/security/advisories/new) - see [`SECURITY.md`](SECURITY.md).
- **Contributing:** [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).
- **Release notes:** [`CHANGELOG.md`](CHANGELOG.md).

---

## License

Lintel is released under the [MIT License](LICENSE). By contributing, you agree that your contributions will be licensed under the same terms.
