# Aegis — Technical Specification

**Version:** 0.1 (draft)
**Status:** design / pre-implementation
**Audience:** contributors building the tool

---

## 0. Name and Tagline

**Aegis** — *the shield your commits pass through.*

The name comes from Greek mythology (Athena's/Zeus's protective shield). It is short (5 letters), easy to type as a CLI (`aegis run`), and evokes "defensive layer in front of something valuable." If the name needs to change later, the binary name, config directory (`.aegis/`), and config filename (`aegis.yaml`) are the only identifiers that need updating.

---

## 1. Overview

Aegis is a shift-left security orchestrator that runs as a Git hook (and optionally in CI). It does **not** implement scanners itself. Instead, it coordinates best-in-class open-source scanners installed locally by the user, reads a single declarative spec file from the repository, and blocks or warns on the commit based on findings.

**Tagline:** *A single Go binary. Zero runtime dependencies. Orchestrates the scanners you trust.*

### 1.1 Problem Statement

Developer teams want shift-left security (secrets, SAST, SCA, lint, format) on every commit, but the existing landscape forces them to either (a) glue together Husky + lint-staged + N individual tools with bespoke config per tool, or (b) adopt a proprietary SaaS platform. Option (a) is brittle and hard to keep consistent across repos; option (b) is expensive and usually ships as a "black box."

### 1.2 What Aegis Is

- A **single static binary** (Go, cross-compiled for linux/darwin/windows on amd64/arm64).
- A **declarative spec**: one YAML file describes the stack, scanners, gates, and ignores.
- A **git hook manager** with a tight UX (install/uninstall/run/doctor).
- A **reporter**: normalizes JSON/SARIF output from heterogeneous scanners into a single view.

### 1.3 What Aegis Is Not

- Not a scanner. Aegis never implements regexes, rule engines, or vuln databases of its own.
- Not a package manager. Aegis does not install scanner binaries by default (v1); users bring their own.
- Not a CI server. Aegis runs in CI by being invoked from one, not by replacing one.
- Not a secret vault, SBOM platform, or compliance dashboard.

---

## 2. Goals and Non-Goals

### 2.1 Goals

1. **Zero runtime install.** A single binary works on a fresh developer machine; no Node, no Python, no Java.
2. **Zero scanner lock-in.** Any supported scanner can be swapped via config.
3. **Reproducible.** Same spec + same scanner versions = same result on every machine.
4. **Fast.** p95 pre-commit wall time under 5 seconds on a repo of 10k staged LOC.
5. **Safe by default.** All blocking gates enabled on `aegis init`; overrides require an auditable reason.
6. **Supply-chain hardened.** Every external binary is sha256-verified before execution.

### 2.2 Non-Goals

- Visual dashboards or a web UI.
- Centralized policy server (v1).
- Writing new scanning engines.
- Supporting proprietary/paid scanners in the default distribution.

---

## 3. High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                          aegis (single binary)                   │
│                                                                  │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────┐   │
│  │  CLI     │──▶│  Config  │──▶│ Detector │──▶│   Resolver   │   │
│  │  entry   │   │  loader  │   │ (stacks) │   │  (binaries + │   │
│  └──────────┘   └──────────┘   └──────────┘   │   sha256)    │   │
│                                               └──────┬───────┘   │
│                                                      ▼           │
│                    ┌─────────────────────────────────────────┐   │
│                    │             Runner (parallel)           │   │
│                    │  secrets  malicious  deps  lint  format │   │
│                    └─────────────────────────────────────────┘   │
│                                      │                           │
│                                      ▼                           │
│                    ┌─────────────────────────────────────────┐   │
│                    │  Normalizer → Gate → Reporter → Exit    │   │
│                    └─────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
                                │  subprocess exec
                ┌───────────────┼───────────────┬──────────────┐
                ▼               ▼               ▼              ▼
           gitleaks        opengrep       osv-scanner     biome / ruff / ...
         (user-installed, verified by sha256 on every run)
```

### 3.1 Core Principles

- **Orchestrator-only.** Aegis never contains a rule, a regex, or a CVE list.
- **Declarative.** Every behavior change happens in `aegis.yaml`, never in code flags at call sites.
- **Fail closed.** If a required scanner is missing, the commit is blocked with a clear install hint — never silently skipped.
- **Everything is JSON internally.** Pretty output is one of several formatters; machine formats are first-class.

---

## 4. Repository Layout

When a user runs `aegis init` in a repo, Aegis creates:

```
<repo>/
├── .aegis/
│   ├── aegis.yaml          # main spec (committed)
│   ├── rules/              # custom scanner rules (committed, optional)
│   │   ├── gitleaks.toml
│   │   └── opengrep/
│   ├── allowlist.yaml      # path globs & reasons (committed)
│   ├── baseline.json       # snapshot of existing findings to ignore (committed)
│   └── overrides.log       # audit log of bypasses (gitignored)
└── .git/hooks/             # installed by `aegis install`
    ├── pre-commit
    ├── commit-msg
    └── pre-push
```

**Hard rule:** `.aegis/` contains **only** text config. Never binaries, never vendored scanners. If a user vendors a binary there, `aegis doctor` prints a warning.

---

## 5. The Spec File (`aegis.yaml`) — Full Schema

All keys are optional unless marked **required**. Unknown keys produce a warning, not an error, so forward-compatibility is maintained.

```yaml
version: 1                    # required; schema version

# -----------------------------------------------------------
# Project identification
# -----------------------------------------------------------
project:
  name: my-service            # informational only
  type: auto                  # auto | list of: npm, yarn, pnpm, maven, gradle,
                              # pip, poetry, go, cargo, composer, bundler, mix
  # Override manifests (rarely needed; overrides auto-detection)
  manifests:
    - package.json
    - pom.xml

# -----------------------------------------------------------
# Scanner binaries — user-installed, Aegis verifies & executes
# -----------------------------------------------------------
binaries:
  gitleaks:
    command: gitleaks                  # resolved via path | $PATH | ~/.aegis/bin/
    path: ~/.aegis/bin/gitleaks        # optional absolute override
    version: "8.28.0"                  # required; displayed in `aegis doctor`
    sha256:
      linux_amd64:   "abc123..."
      linux_arm64:   "..."
      darwin_amd64:  "..."
      darwin_arm64:  "..."
      windows_amd64: "..."
    install_hint: "https://github.com/gitleaks/gitleaks/releases/tag/v8.28.0"

  opengrep:
    command: opengrep
    version: "1.4.0"
    sha256: { linux_amd64: "...", darwin_arm64: "..." }
    install_hint: "https://github.com/opengrep/opengrep/releases"

  osv-scanner:
    command: osv-scanner
    version: "2.0.3"
    sha256: { linux_amd64: "...", darwin_arm64: "..." }
    install_hint: "https://github.com/google/osv-scanner/releases"

  biome:
    command: biome
    version: "1.9.4"
    sha256: { linux_amd64: "...", darwin_arm64: "..." }
    install_hint: "https://biomejs.dev/"

  ruff:
    command: ruff
    version: "0.7.0"
    sha256: { linux_amd64: "...", darwin_arm64: "..." }

  golangci-lint:
    command: golangci-lint
    version: "1.61.0"
    sha256: { linux_amd64: "..." }

# -----------------------------------------------------------
# Checks
# -----------------------------------------------------------
checks:

  # 1) Secret detection
  secrets:
    enabled: true
    engine: gitleaks                   # gitleaks | trufflehog | detect-secrets
    mode: block                        # block | warn | off
    scan:
      staged_only: true                # scan only staged files in pre-commit
      full_on_push: true               # scan history on pre-push
    rules: .aegis/rules/gitleaks.toml  # optional custom rules
    warn_paths:
      - "**/*_test.{go,js,ts,py,rb}"
      - "**/test/**"
      - "**/__tests__/**"
      - "**/*.spec.{js,ts}"
      - "**/fixtures/**"
    inline_ignore: "aegis:ignore-secret"
    entropy_threshold: 4.3             # gitleaks-specific; optional

  # 2) Malicious / insecure code patterns (SAST)
  malicious_code:
    enabled: true
    engine: opengrep                   # opengrep | semgrep
    mode: block
    rulesets:
      - p/security-audit
      - p/owasp-top-ten
      - p/command-injection
      - p/secrets                      # additional coverage
      - .aegis/rules/opengrep/         # local custom rules dir
    severity_threshold: ERROR          # ERROR blocks; WARNING/INFO report only
    timeout_seconds: 60
    exclude_paths:
      - "vendor/**"
      - "node_modules/**"
      - "dist/**"

  # 3) Vulnerable dependencies (SCA)
  dependencies:
    enabled: true
    engine: osv-scanner                # osv-scanner | grype | trivy
    mode: block
    block_severity: [CRITICAL, HIGH]   # only these block; MEDIUM/LOW report only
    suggest_fix: true                  # emit upgrade advice per finding
    offline:
      enabled: true                    # cache the vuln DB locally
      refresh_hours: 24
      db_path: ~/.aegis/cache/osv
    ignore_cves:                       # explicit risk-accepted CVEs
      - id: CVE-2023-12345
        reason: "Not reachable in our call graph; tracked in JIRA-482"
        expires: "2026-12-31"
    manifest_globs:                    # override auto-detected manifests
      - "**/package-lock.json"
      - "**/pom.xml"
      - "**/go.sum"

  # 4) Linting
  lint:
    enabled: true
    mode: warn                         # lint issues never block by default
    auto_fix: true                     # run --fix and re-stage changed files
    fail_on_severity: off              # off | error | warning
    tools:
      javascript:  biome
      typescript:  biome
      python:      ruff
      go:          golangci-lint
      java:        spotless            # requires Maven/Gradle project
      rust:        clippy
      shell:       shellcheck
    args:                              # per-tool extra flags
      biome:        ["--reporter=json"]
      ruff:         ["check", "--output-format=json"]
      golangci-lint: ["run", "--out-format=json"]

  # 5) Formatting
  format:
    enabled: true
    mode: fix                          # fix | check | off
                                       # fix  = format & re-stage
                                       # check = fail if unformatted
    tools:
      javascript:  biome
      typescript:  biome
      python:      ruff
      go:          gofmt
      java:        spotless
      rust:        rustfmt
      shell:       shfmt

# -----------------------------------------------------------
# Scope — what files Aegis looks at
# -----------------------------------------------------------
scope:
  staged_only: true                    # default for pre-commit
  full_scan_for: [dependencies]        # always full-tree for these checks
  exclude_paths:                       # global exclusions
    - ".git/**"
    - "vendor/**"
    - "node_modules/**"
    - "dist/**"
    - "build/**"
    - "*.min.js"

# -----------------------------------------------------------
# Hooks — which checks run at which git event
# -----------------------------------------------------------
hooks:
  pre-commit:
    checks: [secrets, malicious_code, lint, format]
    fail_fast: false
  commit-msg:
    enabled: false                     # reserved for future (commit message lint)
  pre-push:
    checks: [secrets, dependencies]    # heavier checks deferred to push
    fail_fast: true

# -----------------------------------------------------------
# Output
# -----------------------------------------------------------
output:
  format: pretty                       # pretty | json | sarif | junit
  group_by: check                      # check | file | severity
  show_fix_suggestions: true
  color: auto                          # auto | always | never
  verbosity: normal                    # quiet | normal | verbose | debug
  report_file: null                    # optional path to also write JSON/SARIF

# -----------------------------------------------------------
# Override / bypass
# -----------------------------------------------------------
override:
  env_var: AEGIS_SKIP                  # e.g. AEGIS_SKIP=secrets,lint
  allow_no_verify: false               # if false, Aegis refuses when --no-verify
                                       # is detected (via a pre-commit guard)
  require_reason: true                 # prompt for reason, log to overrides.log
  log_file: .aegis/overrides.log

# -----------------------------------------------------------
# Performance
# -----------------------------------------------------------
performance:
  parallel: auto                       # auto = num CPUs; or integer
  check_timeout_seconds: 120
  total_timeout_seconds: 300
  cache:
    enabled: true
    path: ~/.aegis/cache
    ttl_hours: 24
```

### 5.1 Schema Notes

- Field names use `snake_case` consistently.
- Durations are integers in seconds (no ISO-8601).
- Paths accept `~` (home expansion) and are always relative to the repo root unless absolute.
- Globs use `doublestar` semantics (`**` recursive).
- `enabled: false` on any block short-circuits that subtree; Aegis skips it entirely.

---

## 6. Binary Management

### 6.1 Resolution Order

For a binary named `X`, Aegis looks in this order:

1. `binaries.X.path` if set (absolute or `~`-expanded)
2. `$AEGIS_BIN_DIR/X` if `AEGIS_BIN_DIR` is set
3. `~/.aegis/bin/X` (documented convention)
4. `$PATH` lookup via `exec.LookPath(binaries.X.command)`

If none match, Aegis errors with:

```
✖ required binary not found: gitleaks (v8.28.0)
  searched: /home/alice/.aegis/bin/gitleaks, $PATH
  install:  https://github.com/gitleaks/gitleaks/releases/tag/v8.28.0
  hint:     drop the binary on $PATH or at ~/.aegis/bin/gitleaks, then run `aegis doctor`
```

### 6.2 Verification

Before each execution of an external binary, Aegis:

1. Reads the file into a streaming sha256.
2. Compares against `binaries.X.sha256.<os>_<arch>`.
3. If mismatched → refuses to run, prints expected vs actual, exits non-zero.
4. If the platform key is missing → prints a warning but still runs in `permissive` mode (configurable).

The hash is cached in memory for the lifetime of the process (a single `aegis run`) to avoid re-hashing between checks.

### 6.3 Version Enforcement

`binaries.X.version` is compared against the scanner's `--version` output (pattern-matched per scanner). Mismatch → warning by default, block if `strict_versions: true` is set.

### 6.4 `aegis install <tool>` (v1.1+)

Optional convenience. Downloads the pinned release from the configured `install_hint`, verifies sha256, places it at `~/.aegis/bin/X`, sets mode 0755. Never installs system-wide. Refuses to run if the download URL is not HTTPS from a known-good host (`github.com`, `objects.githubusercontent.com`, etc.).

---

## 7. Project Type Auto-Detection

Runs once per invocation, cached in memory. Order:

1. **Explicit.** If `project.type` is a list (not `auto`), use that set. Done.
2. **Manifest scan.** Walk from repo root (excluding `scope.exclude_paths`). Map markers to stacks:

| Marker                         | Stack    |
|--------------------------------|----------|
| `package.json` + `package-lock.json` | npm   |
| `package.json` + `yarn.lock`   | yarn     |
| `package.json` + `pnpm-lock.yaml` | pnpm  |
| `pom.xml`                      | maven    |
| `build.gradle` / `build.gradle.kts` | gradle |
| `requirements.txt` / `Pipfile` | pip      |
| `pyproject.toml`               | poetry / pip (detect `[tool.poetry]`) |
| `go.mod`                       | go       |
| `Cargo.toml`                   | cargo    |
| `composer.json`                | composer |
| `Gemfile`                      | bundler  |
| `mix.exs`                      | mix      |

3. **File-extension fallback.** Only if (2) yields zero matches. Count extensions in staged files; pick the top one.

Multi-stack is supported and expected: a repo can detect as `[npm, python, docker]`. Each detected stack activates its lint/format/SCA defaults independently.

---

## 8. Checks — Detailed Specification

Every check produces zero or more `Finding` objects with a normalized schema:

```go
type Finding struct {
    Check       string        // "secrets" | "malicious_code" | "dependencies" | "lint" | "format"
    RuleID      string        // e.g. "aws-access-key", "CVE-2023-12345", "SQLi.raw-concat"
    Severity    Severity      // CRITICAL | HIGH | MEDIUM | LOW | INFO
    File        string        // repo-relative
    Line        int           // 1-based, 0 if N/A
    Column      int
    Message     string        // one-line human message
    Snippet     string        // optional code snippet (redacted for secrets)
    FixSuggest  string        // optional remediation
    Blocking    bool          // set by the Gate stage
    Engine      string        // "gitleaks" etc.
    EngineRaw   json.RawMessage // original scanner output for debugging
}
```

### 8.1 Secret Detection

- **Default engine:** `gitleaks` (fast, pre-commit native via `gitleaks protect --staged`).
- **Alternatives:** `trufflehog` (adds live API verification), `detect-secrets` (baseline workflow).
- **Pre-commit flow:**
  1. Run scanner on staged files only.
  2. For each finding, check `warn_paths`: if matched, demote to `warn`.
  3. Check inline ignore comment `// aegis:ignore-secret <reason>` on the same or previous line; if present and `reason` non-empty → suppress.
  4. Block if any finding remains and `mode: block`.
- **Pre-push flow (if configured):** run `gitleaks detect` over new commits (`HEAD...@{upstream}`).
- **Snippet redaction:** secret values are **never** printed in full. Show first/last 4 chars only.

### 8.2 Malicious / Insecure Code (SAST)

- **Default engine:** `opengrep` (Semgrep-compatible fork, fully OSS, restored taint analysis features).
- **Alternative:** `semgrep` (CE).
- Runs only on staged files in pre-commit; respects `exclude_paths`.
- Rulesets can be registry refs (`p/security-audit`) or local directories.
- Severity mapping: Semgrep/OpenGrep `ERROR → HIGH`, `WARNING → MEDIUM`, `INFO → LOW`.
- **Note on "malicious code":** Aegis positions this as "insecure or suspicious patterns" — injection sinks, eval-of-user-input, hardcoded keys, unsafe deserialization, etc. It is **not** behavioral malware detection. The CLI prints this clarification on first run.

### 8.3 Dependency Vulnerabilities (SCA)

- **Default engine:** `osv-scanner` (Google-maintained, OSV.dev, guided remediation).
- **Alternatives:** `grype` (Anchore), `trivy` (broader scope; pin to a post-incident safe version — see §16).
- **Flow:**
  1. Detect lockfiles from auto-detected stacks (`package-lock.json`, `pom.xml`, `go.sum`, `Cargo.lock`, `requirements.txt`, `poetry.lock`, etc.).
  2. Run scanner in offline mode if `offline.enabled`, else online.
  3. Filter out CVEs in `ignore_cves` (respect `expires` date).
  4. Build findings with severity + `FixSuggest` populated from `fixed_version` in scanner output.
  5. Block if any finding matches `block_severity`.
- **Scope:** always full-tree for SCA (dependency changes are global), regardless of staged file set.

### 8.4 Linting

- Per-language adapters pick the canonical tool (see defaults in §5 and appendix A).
- **Staged-only** by default. For languages with multi-file awareness (TypeScript, Java), Aegis passes the minimal context needed (project root + staged files); the adapter handles how.
- `mode: warn` by default — lint never blocks a commit. Teams that want strict lint set `mode: block`.
- `auto_fix: true` runs the tool's `--fix` (or equivalent), then `git add` the touched files before the next check runs.

### 8.5 Formatting

- Same model as linting, but with `mode: fix` as default.
- After auto-format, changed files are re-staged. If the formatter changes a file that the developer also had unstaged changes in, Aegis **aborts** with a clear error — it never silently discards unstaged work.

---

## 9. Execution Pipeline

```
git commit
    │
    ▼
.git/hooks/pre-commit  →  aegis run --hook pre-commit
                              │
                              ▼
                      ┌───────────────┐
                      │ 1. Load spec  │  parse aegis.yaml, validate schema
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 2. Detect     │  project types, staged files
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 3. Resolve    │  binary paths + sha256 verify
                      └───────────────┘   ← fails fast if any missing/mismatched
                              │
                              ▼
                      ┌───────────────┐
                      │ 4. Plan       │  which checks run, in which order
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 5. Execute    │  goroutines, one per check
                      │   (parallel)  │  bounded by performance.parallel
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 6. Normalize  │  scanner JSON → []Finding
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 7. Apply      │  warn_paths, allowlist, baseline,
                      │   filters     │  ignore_cves, inline ignores
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 8. Gate       │  mark Blocking per mode/severity
                      └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │ 9. Report     │  format + print + write report_file
                      └───────────────┘
                              │
                              ▼
                     exit 0 (allow) or non-zero (block)
```

### 9.1 Parallelism

- Each check runs in its own goroutine with a context carrying `check_timeout_seconds`.
- A global semaphore limits to `performance.parallel` concurrent scanner processes.
- If `hooks.pre-commit.fail_fast: true`, the first blocking finding cancels sibling contexts.

### 9.2 Determinism

- Findings are sorted on output by `(check, file, line, rule_id)`.
- Parallelism never affects the reported order.

---

## 10. Staged File Scope

```
git diff --cached --name-only --diff-filter=ACMR -z
```

Applied filters:

1. `scope.exclude_paths` (global).
2. Per-check `exclude_paths`.
3. Renamed files: the new path is scanned; old path ignored.
4. Binary files: skipped for text-based scanners (detected via `--numstat` showing `-`).

For checks listed in `scope.full_scan_for`, the full repo file set is used instead.

---

## 11. Output & Reporting

### 11.1 Pretty (default)

```
╭─ Aegis  v0.4.2  ──────────────────────────────────────╮
│ repo: my-service       stacks: [npm, go]             │
│ staged: 14 files (1,203 lines)                       │
╰───────────────────────────────────────────────────────╯

  ✖ secrets           2 findings    (1 blocking)
  ✖ malicious_code    1 finding     (1 blocking)
  ⚠ dependencies      3 findings    (0 blocking)
  ⚠ lint              8 findings    (0 blocking)
  ✓ format            all clean

── secrets ─────────────────────────────────────────────
  [BLOCK] src/config.ts:42
    rule:  aws-access-key
    match: AKIA****XYZ9
    fix:   remove and use AWS_ACCESS_KEY_ID env var

  [WARN]  src/config_test.ts:17   (test file — warn only)
    rule:  generic-api-key
    match: sk_test_****1234

── malicious_code ──────────────────────────────────────
  [BLOCK] src/db.go:88
    rule:  go.lang.security.audit.dangerous-command-write.dangerous-command-write
    code:  exec.Command("sh", "-c", userInput)
    fix:   avoid shell=true; pass args as list

── dependencies ────────────────────────────────────────
  [WARN]  package-lock.json
    lodash@4.17.20  →  fix available: 4.17.21
    CVE-2021-23337 (HIGH) — Command Injection

✖ commit blocked — 2 blocking findings
  bypass: AEGIS_SKIP=secrets,malicious_code git commit  (requires reason)
```

### 11.2 JSON

Stable schema, one object per run:

```json
{
  "version": 1,
  "aegis_version": "0.4.2",
  "repo": "my-service",
  "stacks": ["npm", "go"],
  "started_at": "2026-04-21T10:12:33Z",
  "duration_ms": 2481,
  "hook": "pre-commit",
  "summary": { "blocking": 2, "total": 14 },
  "findings": [ /* Finding[] */ ]
}
```

### 11.3 SARIF

SARIF 2.1.0 for CI integration (GitHub code scanning, GitLab). Each check becomes a separate `run` with its scanner as the `tool.driver`.

### 11.4 JUnit

Optional (`output.format: junit`) for CI systems that only parse JUnit XML.

### 11.5 Exit Codes

| Code | Meaning                                                       |
|------|---------------------------------------------------------------|
| 0    | Success — no blocking findings                                |
| 1    | Blocking findings detected                                    |
| 2    | Configuration error (invalid spec, missing required field)    |
| 3    | Binary resolution or verification failure                     |
| 4    | Scanner crashed or timed out                                  |
| 5    | Internal error (bug in Aegis)                                 |
| 130  | Interrupted (SIGINT)                                          |

---

## 12. CLI Reference

```
aegis <command> [flags]
```

| Command                | Purpose                                                          |
|------------------------|------------------------------------------------------------------|
| `aegis init`           | Create `.aegis/aegis.yaml` with defaults based on auto-detection |
| `aegis install`        | Install Git hooks (creates `.git/hooks/*`)                       |
| `aegis uninstall`      | Remove Git hooks                                                 |
| `aegis run`            | Run all enabled checks (no hook context)                         |
| `aegis run --hook <h>` | Run the set of checks configured for hook `h`                    |
| `aegis run --check <c>`| Run a single check                                               |
| `aegis doctor`         | Verify binaries, versions, sha256; print environment diagnostics |
| `aegis baseline`       | Snapshot current findings into `.aegis/baseline.json`            |
| `aegis ignore <rule>`  | Add a rule to the allowlist (interactive)                        |
| `aegis fmt`            | Shortcut for `aegis run --check format`                          |
| `aegis version`        | Print Aegis version and schema version                           |
| `aegis explain <rule>` | Print documentation for a rule (delegates to scanner)            |

Global flags: `--config <path>`, `--output <format>`, `--quiet`, `--verbose`, `--no-color`.

---

## 13. Git Hook Integration

### 13.1 Installation

`aegis install` writes scripts like:

```bash
#!/usr/bin/env sh
# .git/hooks/pre-commit — generated by aegis; do not edit
exec aegis run --hook pre-commit "$@"
```

It never overwrites an existing hook without `--force`. If one exists, it appends an `include` call and warns the user.

### 13.2 Supported Hooks

- `pre-commit` (primary)
- `commit-msg` (reserved; for future commit-message linting)
- `pre-push` (for heavier checks)
- `prepare-commit-msg` (v2; for AI-powered commit hints)

### 13.3 `--no-verify` Defense

Git's `--no-verify` bypasses all hooks. Aegis cannot intercept this from the hook itself. Mitigations:

1. Document this clearly; teams enforce via CI running `aegis run --hook pre-commit` on every PR.
2. Optionally, `aegis install --server-side` writes a pre-receive hook on the remote (self-hosted Git servers only).
3. `override.allow_no_verify: false` has no runtime effect on the local machine but signals intent and is logged by server-side enforcement.

---

## 14. Allowlist, Baseline, and Inline Ignores

Three layers, evaluated in order:

### 14.1 Allowlist (`.aegis/allowlist.yaml`)

Path- or rule-scoped, **requires a reason**:

```yaml
entries:
  - path: "src/legacy/*.js"
    checks: [lint]
    reason: "pending rewrite, tracked in JIRA-1021"
    expires: "2026-09-30"
  - rule: "go.lang.security.audit.dangerous-command-write"
    path: "cmd/migrate/main.go"
    reason: "trusted migration script; inputs are compile-time constants"
```

Expired entries are ignored and produce a warning.

### 14.2 Baseline (`.aegis/baseline.json`)

Snapshot of existing findings at the moment `aegis baseline` was run. New findings must be *truly new* (not in the baseline) to block. This is how a team adopts Aegis on a legacy codebase without a flag day.

Matching is done by `(check, rule_id, file, normalized_snippet_hash)`. Line numbers are not part of the key (files get reformatted).

### 14.3 Inline Ignores

Per-language comment markers on the same or preceding line:

```go
// aegis:ignore-secret  reason="test fixture, not a real key"
testKey := "AKIAFAKEFAKEFAKE1234"
```

```python
# aegis:ignore-rule=SQLi.raw-concat  reason="hardened elsewhere"
```

**A reason is mandatory.** An inline ignore without a reason is itself a finding.

---

## 15. Override / Bypass

Emergencies happen. Aegis supports skipping with an audit trail:

```bash
AEGIS_SKIP=secrets,lint AEGIS_REASON="hotfix CVE-2026-XYZ, audit ticket 9912" \
  git commit -m "..."
```

Behavior:

- Skipped checks are logged to `.aegis/overrides.log` with user, timestamp, commit hash, and reason.
- If `override.require_reason: true` (default) and `AEGIS_REASON` is unset, the skip is refused.
- `AEGIS_SKIP=all` is allowed but always requires a reason and is flagged as a critical event.
- Overrides never suppress `secrets` findings if `override.protect_secrets: true` (the only non-skippable gate).

---

## 16. Security Model (Meta)

Aegis is itself a supply-chain-sensitive tool. It executes external binaries with the full privileges of the developer's shell. Threat model:

### 16.1 Threats

- **Malicious scanner binary.** A scanner is replaced with a credential stealer (see the 2026 Trivy incident: compromised releases on GitHub, Docker Hub, and GitHub Actions exposed CI/CD secrets and credentials).
- **Malicious Aegis release.** Our own binary is compromised.
- **Malicious spec file.** A PR introduces a spec that points `binaries.X.path` at something evil.
- **Malicious custom rules.** A PR adds an OpenGrep rule that does something pathological (ReDoS, exfil via rule metadata if the scanner ever supports it).

### 16.2 Mitigations

1. **Mandatory sha256 verification** of every external binary on every run.
2. **No auto-download by default.** `aegis install <tool>` is opt-in; when used, downloads only over HTTPS from allow-listed hosts.
3. **Spec schema limits.** `binaries.X.path` is rejected if it resolves outside: repo root, `$HOME`, or absolute system paths. Relative paths inside the repo (i.e., pointing at a vendored binary) are **always rejected** — defense in depth against a malicious PR slipping a binary in.
4. **Signed releases.** Aegis binaries are published with Sigstore cosign signatures and a SLSA Level 3 provenance attestation. `aegis version --verify` checks its own binary.
5. **Reproducible builds.** The build is bit-reproducible; anyone can verify the release from source.
6. **Capability minimization.** Aegis does not open outbound network connections except when: (a) downloading the OSV DB for the `dependencies` check in online mode, (b) the user runs `aegis install <tool>`. Both are explicit.
7. **Read-only config at runtime.** Aegis never writes to `.aegis/aegis.yaml`. It only writes to `.aegis/baseline.json` (on `aegis baseline`), `.aegis/overrides.log` (on bypass), and `~/.aegis/cache/*`.
8. **Telemetry off by default, no dark patterns.** No phone-home.

### 16.3 Secure defaults

The `aegis init`-generated config sets `block` mode for secrets, malicious_code, and dependencies; `require_reason: true`; `allow_no_verify: false`; `strict_versions: true`. Users opt *down*, never up.

---

## 17. Performance Targets

| Metric                                              | Budget         |
|-----------------------------------------------------|----------------|
| `aegis run --hook pre-commit` p95 on 10k staged LOC | < 5 seconds    |
| Binary sha256 verify overhead (per binary, cached)  | < 50 ms        |
| Spec parse + validate                               | < 20 ms        |
| Memory, peak                                        | < 250 MB       |
| Cold start (no scanner processes yet)               | < 150 ms       |

Tactics: parallel scanner execution, staged-only scanning, cached OSV DB, memoized sha256, stream-parsed scanner JSON (no buffering of full output before first finding).

---

## 18. Extensibility — Plugin Interface

To add a new check type, implement the `Checker` interface:

```go
package checker

type Checker interface {
    // Name identifies the check ("secrets", "malicious_code", etc.).
    Name() string

    // Applicable returns true if this checker should run for the given
    // project context (stacks, staged files).
    Applicable(ctx ProjectContext) bool

    // Run executes the underlying scanner and returns normalized findings.
    // Must respect the provided context for cancellation/timeout.
    Run(ctx context.Context, input CheckInput) (CheckOutput, error)

    // RequiredBinaries lists the binary keys this checker needs.
    // Used by the Resolver to verify up front.
    RequiredBinaries() []string
}

type CheckInput struct {
    RepoRoot    string
    StagedFiles []string
    FullTree    bool
    Config      json.RawMessage  // the checker's subtree from aegis.yaml
    Binaries    map[string]ResolvedBinary
}

type CheckOutput struct {
    Findings []Finding
    Stats    Stats
}
```

v1 ships built-in checkers for the five features. External plugins are deferred to v2 (Go plugins have portability issues; likely solved via WASM or out-of-process RPC).

---

## 19. Error Handling & UX

Rules for every error message:

1. **What went wrong** — one-line plain English.
2. **Where** — file/line/config key involved.
3. **Why it matters** — one line.
4. **How to fix** — a command or a URL.

Example:

```
✖ spec error: checks.secrets.engine has unknown value "gitleaksz"
  file:    .aegis/aegis.yaml:18:12
  allowed: gitleaks, trufflehog, detect-secrets
  docs:    https://github.com/<org>/aegis/docs/secrets#engine
```

No stack traces by default. `--verbose` enables them.

---

## 20. Distribution

- **GitHub Releases** with binaries for `linux_{amd64,arm64}`, `darwin_{amd64,arm64}`, `windows_amd64`.
- **Homebrew tap** (`brew install <org>/tap/aegis`).
- **Scoop bucket** for Windows.
- **Debian/RPM** packages via `nfpm`.
- **Docker image** (`ghcr.io/<org>/aegis:<version>`), minimal base, single binary.
- **Install script** (`curl -sSL https://aegis.<domain>/install.sh | sh`) — with sha256 + signature verification baked in.
- **No `npm install`, no `pip install`.** Those would contradict the zero-runtime goal.

Every release includes: the binary, its sha256, its Sigstore signature, a SLSA provenance attestation, an SBOM (CycloneDX).

---

## 21. Roadmap

### v1.0 (MVP — target: ~8–10 weeks of focused work)

- Checks: secrets (gitleaks), malicious_code (opengrep), dependencies (osv-scanner), lint (biome/ruff/golangci-lint), format (biome/ruff/gofmt).
- Auto-detect: npm, go, python.
- Hooks: pre-commit, pre-push.
- Output: pretty, JSON.
- Binary verification, allowlist, baseline, inline ignores, override audit log.
- Signed releases with Sigstore.

### v1.1

- Additional engines: trufflehog, grype.
- Stacks: maven, gradle, cargo, composer, bundler.
- SARIF and JUnit output.
- `aegis install <tool>` helper.
- Windows first-class support.

### v1.2

- IaC scanning (Checkov adapter).
- Dockerfile / container scanning (Grype/Syft adapter).
- Commit message linting (commitlint or conventional-commits enforcement).

### v2.0

- Plugin system (out-of-process gRPC-based).
- Centralized policy registry (opt-in): org pulls the spec from a shared repo.
- IDE integration (LSP).
- Web dashboard for team-wide trend analysis (separate, optional component).

---

## Appendix A — Default Scanner Matrix

| Stack     | Lint              | Format      | SCA           |
|-----------|-------------------|-------------|---------------|
| npm/yarn/pnpm | biome         | biome       | osv-scanner   |
| python (pip/poetry) | ruff    | ruff        | osv-scanner   |
| go        | golangci-lint     | gofmt       | osv-scanner   |
| maven     | spotless + checkstyle | spotless | osv-scanner  |
| gradle    | spotless          | spotless    | osv-scanner   |
| cargo     | clippy            | rustfmt     | osv-scanner   |
| composer  | phpstan           | php-cs-fixer| osv-scanner   |
| bundler   | rubocop           | rubocop     | osv-scanner   |

Secrets and malicious_code scanners are language-agnostic.

## Appendix B — Exit Code Reference

See §11.5.

## Appendix C — Environment Variables

| Variable              | Purpose                                                        |
|-----------------------|----------------------------------------------------------------|
| `AEGIS_SKIP`          | Comma-separated checks to skip (e.g., `secrets,lint`)          |
| `AEGIS_REASON`        | Required justification when `override.require_reason: true`    |
| `AEGIS_BIN_DIR`       | Override directory Aegis searches for scanner binaries         |
| `AEGIS_CONFIG`        | Alternate path to the spec file                                |
| `AEGIS_NO_COLOR`      | Disable colored output                                         |
| `AEGIS_CACHE_DIR`     | Override cache directory (default `~/.aegis/cache`)            |
| `AEGIS_OFFLINE`       | Force offline mode globally                                    |
| `NO_COLOR`            | Honored per the no-color.org convention                        |

## Appendix D — Glossary

- **Check.** One of the five Aegis-defined analysis categories.
- **Engine.** The underlying scanner a check is configured to use.
- **Finding.** A single normalized result produced by a check.
- **Gate.** The stage that decides which findings are blocking.
- **Baseline.** A committed list of pre-existing findings to exclude.
- **Allowlist.** A reason-justified suppression list.
- **Spec file.** `.aegis/aegis.yaml`.

---

*End of specification.*