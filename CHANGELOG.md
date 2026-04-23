# Changelog

All notable changes to Aegis are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-04-23

Adds a verified scanner installer so users no longer have to hand-pin
SHA256s, and closes the `strict_versions: false` escape hatch that
prior users fell back on when their manually-installed scanners
produced "no sha256 for platform" errors.

### Added

- `aegis install <scanner>` — downloads the pinned upstream release,
  verifies its SHA256 on the wire, extracts the binary, re-verifies it
  on disk, and places it at `~/.aegis/bin/<name>`.
- `aegis install --all` — installs every scanner declared in the
  loaded `aegis.yaml`. Skips `gofmt` (ships with the Go toolchain).
- Embedded scanner pin database shipped inside the binary
  (`internal/installer/scanners.yaml`). Covers gitleaks, opengrep,
  osv-scanner, biome, ruff, golangci-lint, and shellcheck across
  linux/darwin × amd64/arm64. Each entry pins two hashes:
  `archive_sha256` (verified at download time) and `binary_sha256`
  (verified on every scanner invocation, so extraction bugs or pin
  drift between the two are caught loudly).
- Resolver pin fallback: when the user's `aegis.yaml` has no
  `sha256` for a scanner/platform, the resolver consults the embedded
  pin database instead of refusing outright. Any explicit user pin
  still wins — the fallback is strictly additive.
- HTTPS host allowlist for installs. GitHub release infrastructure
  plus the two vendor CDNs currently in use (`astral.sh`, `biomejs.dev`)
  are trusted; non-HTTPS and off-list hosts are refused before any
  bytes are read.
- `aegis doctor` now prints a count-aware remediation hint after
  failures: one scanner missing → `run: aegis install <name>`;
  multiple → `run: aegis install --all`. `gofmt` is special-cased
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
  or bypassed (`AEGIS_SKIP=secrets`, inline ignores, and `--no-verify` all
  still run it).
- Override mechanism (`AEGIS_SKIP` + mandatory `AEGIS_REASON`) with an
  append-only audit log at `.aegis/overrides.log`.
- Release artifacts cross-compiled with `CGO_ENABLED=0` and signed with
  Sigstore keyless.

#### Project infrastructure

- MIT License.
- Community health files: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `SECURITY.md`, `CHANGELOG.md`, `CODEOWNERS`.
- Issue and pull-request templates; Dependabot config for Go modules,
  GitHub Actions, and pip.
- Reference `aegis.yaml` configs under `examples/` for go-service,
  typescript-monorepo, and python-lib.
- MkDocs-Material documentation site at
  https://mhchlagou.github.io/aegis/, deployed via native GitHub Pages.

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
- External scanner binaries per your `aegis.yaml`. Aegis coordinates them
  but does not bundle or download them - install and pin each one you use.

[Unreleased]: https://github.com/MHChlagou/aegis/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/MHChlagou/aegis/releases/tag/v0.2.0
[0.1.0]: https://github.com/MHChlagou/aegis/releases/tag/v0.1.0
