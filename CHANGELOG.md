# Changelog

All notable changes to Aegis are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-04-21

### Added

- Initial MVP release covering the v1.0 scope of the project specification.
- Commands: `init`, `install`, `uninstall`, `run` (with `--hook` and `--check`),
  `doctor`, `baseline`, `ignore`, `fmt`, `explain`, `version`.
- Nine-stage execution pipeline: config load → validate → staged-file detect →
  stack detect → binary resolve + hash verify → parallel run → normalize →
  allowlist/baseline/inline filter → gate → report.
- Adapters: `gitleaks`, `opengrep`, `osv-scanner`, `biome`, `ruff`,
  `golangci-lint`, `gofmt`, `shellcheck`.
- Pretty and JSON output formats; deterministic sort; exit codes per spec §11.5.
- Single static binary for `linux/amd64`, `linux/arm64`, `darwin/amd64`,
  `darwin/arm64`, `windows/amd64` (CGO disabled).
- CI workflows: cross-platform tests, lint, matrix build, `govulncheck`, and
  an end-to-end self-smoke test.
- Release workflow with keyless signing on tagged versions.

[Unreleased]: https://github.com/aegis-sec/aegis/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/aegis-sec/aegis/releases/tag/v0.1.0
