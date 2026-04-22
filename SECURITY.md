# Reporting a Security Issue

Thanks for helping keep Aegis and its users safe.

## How to report

Please use **GitHub Private Vulnerability Reporting** on this repository:

1. Go to the **Security** tab of the repository.
2. Click **Report a vulnerability**.
3. Fill in the form with a description, steps to reproduce, and impact.

Please do not open a public GitHub issue for a security matter.

## What to include

- A short description of the issue and its impact.
- A minimal reproduction: repo layout, `aegis.yaml`, commands, and observed
  vs. expected output.
- Your assessment of the affected versions and any mitigations you know of.
- Whether you are willing to be credited in the published advisory.

## What to expect

| Step                        | Target                                |
| --------------------------- | ------------------------------------- |
| Initial acknowledgement     | within **3 business days**            |
| Triage + severity rating    | within **10 business days**           |
| Fix + coordinated advisory  | within **90 days** of acknowledgement |

We publish advisories via GitHub Security Advisories and credit reporters who
want to be named.

## Scope

In scope:

- The `aegis` binary and all code under this repository.
- Official release artifacts published on GitHub Releases.
- CI / release workflows under `.github/workflows/`.

Out of scope (please report upstream):

- Bugs in third-party scanner binaries (`gitleaks`, `opengrep`, `osv-scanner`,
  `biome`, `ruff`, `golangci-lint`, `gofmt`, `shellcheck`). Aegis is a
  coordinator, not the scanner author.
- Issues in the Go standard library or dependencies — file against those
  upstream projects, and we will pick up fixes via `go mod tidy` + releases.

## Hardening notes for operators

If you run Aegis in CI or as a hook:

- Keep `strict_versions: true` in `aegis.yaml` so binary hashes are checked on
  every invocation.
- Pin `aegis` itself to a released version with a published checksum; do not
  build from `main` in production.
- Verify release artifacts with the published Sigstore signatures (see
  [`docs/docs/supply-chain.md`](docs/docs/supply-chain.md)).
