# Changelog

All notable changes to Lintel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.2] - 2026-04-24

Implements the SARIF output writer that lintel's CLI flag
(`--output sarif`) had been advertising since 0.2.0. This unblocks
the upcoming `lintel-action` GitHub Action: SARIF is the format
GitHub Advanced Security ingests into the Security → Code scanning
tab and the source of inline PR annotations.

### Added

- SARIF 2.1.0 output via `lintel run --output sarif`. Emits to stdout
  so workflow steps can redirect to a file and hand the result to
  `github/codeql-action/upload-sarif`. Deduplicates rules across
  findings (one entry per unique `RuleID` in `tool.driver.rules`),
  maps lintel severities to SARIF levels (`CRITICAL`/`HIGH` →
  `error`, `MEDIUM` → `warning`, `LOW`/`INFO` → `note`), and emits
  file / line / column locations that GitHub turns into inline PR
  annotations. Windows path separators are normalized to forward
  slashes so SARIF produced on Windows runners still ingests cleanly.

### Fixed

- `--output sarif` previously fell through to pretty-printed terminal
  output because the writer was never implemented, despite the flag
  being accepted by the config validator. It now emits real SARIF.
  (`junit` remains a documented format without a writer and continues
  to fall through to pretty; tracked for a separate release.)

## [0.2.1] - 2026-04-24

Closes a gap in the `secrets` check where gitleaks' default ruleset
silently missed hardcoded credentials in declarative config files
(Dockerfile `ENV pass=Admin124`, `.env` files, `docker-compose.yml`,
Kubernetes manifests, Spring / Java properties, etc.). Gitleaks
optimises for precision on source code and backs off on short or
low-entropy values — lintel now fills that gap with a second,
config-file-aware pass that runs inside the same `secrets` check.

### Added

- Config-secret scanner (`engine: lintel-config`) running alongside
  gitleaks inside the `secrets` check. Targets declarative config
  files where hardcoded credentials are almost always a mistake:
  - `.env` and `.env.*` (excluding `.example` / `.sample` /
    `.template` / `.dist` / `.tmpl` variants),
  - `Dockerfile`, `Dockerfile.*`, `*.Dockerfile`,
  - `docker-compose*.{yml,yaml}`, `compose*.{yml,yaml}`,
  - `*.properties` (Java / Spring),
  - `*.yaml` / `*.yml` (covers Kubernetes, Helm, Ansible, Spring,
    GitHub Actions, GitLab CI, etc.),
  - `*.toml`, `*.ini`, `*.cfg`, `*.conf`, `*.json` (with a blocklist
    for known-noisy manifests such as `package.json`,
    `package-lock.json`, `tsconfig.json`),
  - `*.tf`, `*.tfvars`, `*.hcl` (Terraform / HCL),
  - `Jenkinsfile`, `ansible.cfg`.
- Keyword-based key detection matches case-insensitively on a
  `_`-bounded suffix, so `DB_PASSWORD`, `spring.datasource.password`,
  `my-app-secret`, and `awsAccessKey` (camelCase normalised) all hit
  while `passport`, `bypass`, and `compass` do not. Covered keywords:
  `password`, `passwd`, `passphrase`, `pwd`, `pass`, `secret`,
  `token`, `credential[s]`, `api_key` / `apikey`, `access_key` /
  `accesskey`, `private_key` / `privatekey` / `priv_key`, `auth`,
  `authorization`, `bearer`.
- Placeholder filtering to keep false positives down: skips
  `${VAR}` / `$VAR` / `$(cmd)` / `%(py)s` interpolation,
  `!vault` ansible-vault refs, `<no value>` Helm defaults,
  bracketed templates (`<password>`, `[secret]`, `{{.Password}}`),
  and a curated literal list (`changeme`, `xxx`, `your-password`,
  `null`, `todo`, `tbd`, `example`, `placeholder`, `redacted`, …).
- Kubernetes / Helm `env:` list-shape pairing — pairs
  `- name: DB_PASSWORD` with a subsequent `value: "Admin124"` within
  a 5-line window and runs the pair through the same policy. Catches
  a common pod-spec leak pattern that per-line scanning misses.
- Findings emit `engine: lintel-config` and rule IDs under
  `lintel.config-secret.<keyword>` (e.g. `lintel.config-secret.pass`,
  `lintel.config-secret.api_key`) so users can target them in
  allowlists and baselines independently of gitleaks rules.

### Security

- Values in reported findings are always redacted before display
  (`****` for short values, `FIRST4****LAST4` for longer ones),
  matching the redaction policy already used on gitleaks output.

## [0.2.0] - 2026-04-23

Adds a verified scanner installer so users no longer have to hand-pin
SHA256s, and closes the `strict_versions: false` escape hatch that
prior users fell back on when their manually-installed scanners
produced "no sha256 for platform" errors.

### Added

- `lintel install <scanner>` - downloads the pinned upstream release,
  verifies its SHA256 on the wire, extracts the binary, re-verifies it
  on disk, and places it at `~/.lintel/bin/<name>`.
- `lintel install --all` - installs every scanner declared in the
  loaded `lintel.yaml`. Skips `gofmt` (ships with the Go toolchain).
- `lintel upgrade` - checks the GitHub Releases API for a newer lintel
  version and prints the release notes plus a copy-paste upgrade
  command for your OS and architecture. The command never replaces
  the installed binary; self-updating verifiers are a larger trust
  surface than the UX win warrants.
- Install scripts (`scripts/install.sh`, `scripts/install.ps1`) for
  one-line installs. Both detect OS/arch, download the correct
  release asset, verify its SHA256, and optionally verify the Sigstore
  bundle when `cosign` is on `$PATH`. Configurable via flags
  (`--version`, `--install-dir`, `--no-cosign`) and environment
  variables (`LINTEL_VERSION`, `LINTEL_INSTALL_DIR`,
  `LINTEL_VERIFY_COSIGN`). The Unix script falls back to
  `$HOME/.local/bin` when `/usr/local/bin` is not writable and no
  `sudo` is available.
- Embedded scanner pin database shipped inside the binary
  (`internal/installer/scanners.yaml`). Covers gitleaks, opengrep,
  osv-scanner, biome, ruff, golangci-lint, and shellcheck across
  linux/darwin × amd64/arm64. Each entry pins two hashes:
  `archive_sha256` (verified at download time) and `binary_sha256`
  (verified on every scanner invocation, so extraction bugs or pin
  drift between the two are caught loudly).
- Resolver pin fallback: when the user's `lintel.yaml` has no
  `sha256` for a scanner/platform, the resolver consults the embedded
  pin database instead of refusing outright. Any explicit user pin
  still wins - the fallback is strictly additive.
- HTTPS host allowlist for installs. GitHub release infrastructure
  plus the two vendor CDNs currently in use (`astral.sh`, `biomejs.dev`)
  are trusted; non-HTTPS and off-list hosts are refused before any
  bytes are read.
- `lintel doctor` now prints a count-aware remediation hint after
  failures: one scanner missing → `run: lintel install <name>`;
  multiple → `run: lintel install --all`. `gofmt` is special-cased
  with a "install the Go toolchain" note.
- Release-engineering tool `scripts/refresh-pins.go` (build-ignored).
  Downloads every asset declared in `scanners.yaml`, computes both
  hashes (extracting archives inline), and rewrites the file via
  minimal textual substitution so the diff review stays hash-only.
- Release workflow embeds the `CHANGELOG.md` section for the tagged
  version into GitHub Release notes, appended to the auto-generated
  PR summary.

### Fixed

- Documented `curl` one-liner for "latest release" download now
  normalizes `uname -m` to Go's `GOARCH` (`x86_64` → `amd64`,
  `aarch64` → `arm64`). Previously three of four Unix platform
  combos 404'd because the upstream asset names use `GOARCH`.

## [0.1.0] - 2026-04-22

First public release. Implements the v1.0 scope of the project specification.

### Added

#### CLI + pipeline

- Commands: `init`, `install`, `uninstall`, `run` (with `--hook` and `--check`),
  `doctor`, `baseline`, `ignore`, `fmt`, `explain`, `version`.
- Nine-stage execution pipeline: config load → validate → staged-file detect →
  stack detect → binary resolve + SHA256 verify → parallel run → normalize →
  filter (allowlist / baseline / inline / warn_paths) → gate → report.
- Scanner adapters: `gitleaks`, `opengrep`, `osv-scanner`, `biome`, `ruff`,
  `golangci-lint`, `gofmt`, `shellcheck`.
- Stack auto-detect: `npm`, `python`, `go`, `shell`.
- Git hook integration for `pre-commit` and `pre-push`, with detection and
  delegation to pre-existing foreign hooks (Husky, lefthook, etc.).

#### Output

- Pretty terminal output with color, icons, and a deterministic sort order.
- JSON output with a stable, documented schema. Includes `checks_run` so
  consumers can distinguish "ran and clean" from "filtered out".
- Exit codes per spec §11.5: `0` ok, `1` blocking, `2` config, `3` binary
  resolve, `4` scanner crash.

#### Supply-chain model

- SHA256 pin per platform for every scanner binary; verified on every run.
- `strict_versions: true` by default - refuses to execute unverified binaries.
- `protect_secrets: true` by default - the secrets check cannot be disabled
  or bypassed (`LINTEL_SKIP=secrets`, inline ignores, and `--no-verify` all
  still run it).
- Override mechanism (`LINTEL_SKIP` + mandatory `LINTEL_REASON`) with an
  append-only audit log at `.lintel/overrides.log`.
- Release artifacts cross-compiled with `CGO_ENABLED=0` and signed with
  Sigstore keyless.

#### Project infrastructure

- MIT License.
- Community health files: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `SECURITY.md`, `CHANGELOG.md`, `CODEOWNERS`.
- Issue and pull-request templates; Dependabot config for Go modules,
  GitHub Actions, and pip.
- Reference `lintel.yaml` configs under `examples/` for go-service,
  typescript-monorepo, and python-lib.
- MkDocs-Material documentation site at
  https://mhchlagou.github.io/lintel/, deployed via native GitHub Pages.

#### CI / release workflows

All third-party actions SHA-pinned; least-privilege `permissions:` scoped
per job.

- `ci.yml`: cross-platform test (ubuntu / macos / windows), lint
  (golangci-lint v2.11.4), matrix build (5 platforms), govulncheck,
  end-to-end self-smoke.
- `codeql.yml`: Go static analysis, weekly cron + on every PR.
- `pr-checks.yml`: Conventional Commits title validation + size label.
- `docs.yml`: build and deploy docs via `actions/deploy-pages`.
- `stale.yml`: automated cleanup of inactive issues/PRs.
- `release.yml`: cross-compile, sign with Sigstore, publish GitHub Release
  on `v*` tag push.

### Requirements

- Go **1.25+** for building from source.
- External scanner binaries per your `lintel.yaml`. Lintel coordinates them
  but does not bundle or download them - install and pin each one you use.

[Unreleased]: https://github.com/MHChlagou/lintel/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/MHChlagou/lintel/releases/tag/v0.2.2
[0.2.1]: https://github.com/MHChlagou/lintel/releases/tag/v0.2.1
[0.2.0]: https://github.com/MHChlagou/lintel/releases/tag/v0.2.0
[0.1.0]: https://github.com/MHChlagou/lintel/releases/tag/v0.1.0
