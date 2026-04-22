# Contributor guide

The canonical contributor guide lives in [`CONTRIBUTING.md`](https://github.com/aegis-sec/aegis/blob/main/CONTRIBUTING.md) at the repo root. This page is a short orientation for readers of the documentation site.

## Quick links

- Full guide: [`CONTRIBUTING.md`](https://github.com/aegis-sec/aegis/blob/main/CONTRIBUTING.md)
- Code of conduct: [`CODE_OF_CONDUCT.md`](https://github.com/aegis-sec/aegis/blob/main/CODE_OF_CONDUCT.md)
- Security reports: [`SECURITY.md`](https://github.com/aegis-sec/aegis/blob/main/SECURITY.md)
- Architecture map: [Architecture](architecture.md)

## Three things to know

1. **PR titles must follow Conventional Commits.** A `pr-checks.yml` workflow enforces it. `feat(runner): parallelize scanners` — good. `Changes` — bad.
2. **`make ci` must be green locally** before you push. CI runs the same gates; passing locally is the fast feedback loop.
3. **New scanners need pinned SHA256 hashes** for every platform you claim to support. Aegis will refuse to run unverified binaries, and we will not merge an adapter without pins. See [adding a scanner](adding-a-scanner.md).

## Areas that need review beyond two approvals

These paths have [CODEOWNERS](https://github.com/aegis-sec/aegis/blob/main/.github/CODEOWNERS) guarding them and require reviews from the listed teams in addition to regular maintainer approval:

- `internal/resolve/` — binary resolution and SHA256 verification
- `internal/checker/` — all scanner adapters
- `internal/config/defaults_spec.go` — default pins and scanner definitions
- `internal/cli/run.go` — the override and audit-log flow
- `.github/workflows/release.yml` — the signing workflow

Changes here are deliberately slower to merge. That is the point — the integrity of the whole tool rests on these paths.
