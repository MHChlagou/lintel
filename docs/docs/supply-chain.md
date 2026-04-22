# Supply-chain model

Aegis runs third-party scanner binaries on every commit. That is a trust decision. This page explains what Aegis does to make that trust explicit and auditable, and what it does not do.

## What Aegis verifies

Every time `aegis run` invokes a scanner, it does four things in order before handing control over:

1. **Resolves the binary path.** The path must be absolute, or a simple name that resolves deterministically on `$PATH`. Relative paths (`./tool`, `bin/tool`) are refused.
2. **Runs the scanner's version command** and compares the output against the pinned version string in `aegis.yaml`.
3. **Computes the SHA256 of the binary on disk** and compares against the pinned hash for the running `(os, arch)`.
4. **Only then** executes the scanner with the staged file list.

If any of steps 2–3 fail with `strict_versions: true` (the default), the whole run exits with code `3` — not the individual scanner. This is by design: if a pin is wrong, you want to know before producing a possibly-tainted report.

## The pin file

Pins live inside `aegis.yaml`:

```yaml
scanners:
  gitleaks:
    path: gitleaks
    version: "8.18.2"
    sha256:
      linux/amd64: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
      linux/arm64: "a6f9b3…"
      darwin/amd64: "…"
      darwin/arm64: "…"
      windows/amd64: "…"
```

Pins are **per platform**, because upstream releases binaries per platform with distinct hashes. A missing platform pin means "refuse to run on this platform."

## How to update pins

When a new scanner version is released:

1. Download the upstream release artifacts.
2. Verify the upstream's own signature / checksum (Sigstore, PGP, or GitHub Releases SHA) before accepting the new blob.
3. Compute `sha256sum` for every platform you support.
4. Update the `sha256` block in `aegis.yaml`.
5. Commit with a message like `chore(scanners): pin gitleaks 8.18.3 hashes`.

`aegis doctor --update-pins` is planned for v1.1 to automate steps 3–4 against a trusted release source.

## What Aegis does **not** do

- It does not **download** scanner binaries. That is your responsibility (or your CI's).
- It does not **sandbox** the scanner. Scanners run with Aegis's own privileges. Do not run Aegis as root.
- It does not **validate scanner rule packs.** If `gitleaks` reads a community rules file at runtime, Aegis has no opinion about that rules file's authenticity. Pin or vendor rule packs via your own process.
- It does not **guarantee upstream integrity.** If an upstream project's release is published with a compromised hash, Aegis will pin that compromised hash. The defense is step 2 of "how to update pins": verify upstream's own attestations before trusting a new release.

## Aegis's own supply chain

The `aegis` binary itself is built by a SHA-pinned GitHub Actions workflow (`release.yml`), cross-compiled with `CGO_ENABLED=0`, and signed with [Sigstore keyless signing](https://www.sigstore.dev/). To verify a release binary:

```bash
cosign verify-blob \
  --certificate aegis-linux-amd64.sig.crt \
  --signature  aegis-linux-amd64.sig \
  --certificate-identity-regexp 'https://github.com/aegis-sec/aegis/.github/workflows/release.yml@.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  aegis-linux-amd64
```

A successful `Verified OK` message proves the binary was produced by the release workflow on a tagged commit.

## Threat model summary

| Concern                                                 | Mitigation                                    |
| ------------------------------------------------------- | --------------------------------------------- |
| Malicious scanner binary on `$PATH`                     | SHA256 pin must match                         |
| Scanner upgraded silently to a different version        | Version string pin must match                 |
| Typo'd or unpinned scanner                              | `strict_versions: true` refuses to run        |
| Compromised Aegis release                               | Sigstore keyless signature                    |
| Compromised CI workflow mutating a signed binary        | `release.yml` SHA-pins every action           |
| Secrets check disabled by a developer to bypass a block | `protect_secrets: true` prevents this         |
| Override abuse                                          | Mandatory reason + append-only `.aegis/overrides.log` |

The full threat model is in [`spec.md` §16](reference/spec.md#16-security-model-meta).
