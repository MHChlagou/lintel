# Installation

Aegis ships as a single static binary. There is no runtime to install, no Node / Python / Ruby dependency, and no daemon - just the `aegis` executable and the scanner binaries you want it to coordinate.

## Supported platforms

| OS      | amd64          | arm64          |
| ------- | -------------- | -------------- |
| Linux   | :material-check: | :material-check: |
| macOS   | :material-check: | :material-check: |
| Windows | :material-check: | :material-minus: |

## Install from a release

1. Download the binary for your platform from the [releases page](https://github.com/MHChlagou/aegis/releases/latest).
2. Verify the SHA256 checksum (a `.sha256` file sits next to every binary).
3. Optionally verify the Sigstore signature - see [supply-chain](supply-chain.md).
4. Move the binary onto your `$PATH` (commonly `/usr/local/bin` on Unix).
5. Run `aegis version` to confirm.

```bash
sha256sum -c aegis-linux-amd64.sha256
chmod +x aegis-linux-amd64
sudo mv aegis-linux-amd64 /usr/local/bin/aegis
aegis version
```

## Install via Go

Useful for development or for systems where you prefer to track `main`.

```bash
go install github.com/MHChlagou/aegis/cmd/aegis@latest
```

The binary lands in `$(go env GOBIN)` or `$(go env GOPATH)/bin`.

## Upgrading

### Check for a newer release

```bash
aegis upgrade
```

This queries the GitHub Releases API, compares against your running
version, and — if a newer release exists — prints the release notes
and a ready-to-paste upgrade command for your OS and architecture.
It never modifies the installed binary.

### Apply the upgrade

The command printed by `aegis upgrade` is the same one used to install
originally, pinned to the new tag. Re-run it to overwrite the existing
binary.

=== "macOS / Linux"

    ```bash
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    case "$arch" in
      x86_64|amd64)   arch=amd64 ;;
      aarch64|arm64)  arch=arm64 ;;
    esac
    # Replace v0.2.0 with the tag `aegis upgrade` reported, or use /latest/
    curl -fsSL "https://github.com/MHChlagou/aegis/releases/download/v0.2.0/aegis-${os}-${arch}" \
      -o /usr/local/bin/aegis
    chmod +x /usr/local/bin/aegis
    aegis version
    ```

=== "Go toolchain"

    ```bash
    go install github.com/MHChlagou/aegis/cmd/aegis@latest
    # Or pin: go install github.com/MHChlagou/aegis/cmd/aegis@v0.2.0
    ```

=== "Windows (PowerShell)"

    ```powershell
    # Close any running aegis.exe first — Windows locks running binaries.
    Invoke-WebRequest `
      -Uri https://github.com/MHChlagou/aegis/releases/download/v0.2.0/aegis-windows-amd64.exe `
      -OutFile $env:USERPROFILE\bin\aegis.exe
    aegis version
    ```

### What does not need to change on upgrade

- **Git hooks** (`.git/hooks/pre-commit`, `pre-push`) dispatch to the `aegis`
  binary via `$PATH`; they keep working after the binary is replaced.
- **`~/.aegis/bin/<scanner>`** files are independent of the aegis version.
- **`.aegis/aegis.yaml`** is stable across v0.x — new fields are opt-in.

Run `aegis doctor` after upgrading to confirm every scanner still resolves.
If your `aegis.yaml` has no `sha256` entries and you rely on the embedded
pin-database fallback, the new version may ship different pins — `doctor`
will tell you immediately.

### Verifying the downloaded binary

Every release ships a `.sha256` sidecar and a `.sigstore` bundle. Both are
worth checking if you did not install via `go install`:

```bash
ver=v0.2.0
base="https://github.com/MHChlagou/aegis/releases/download/${ver}"
curl -fsSL "${base}/aegis-linux-amd64.sha256"   -o aegis.sha256
curl -fsSL "${base}/aegis-linux-amd64.sigstore" -o aegis.sigstore

sha256sum -c <(printf '%s  aegis\n' "$(cat aegis.sha256)")

cosign verify-blob --bundle aegis.sigstore \
  --certificate-identity-regexp='^https://github.com/MHChlagou/aegis/' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  aegis
```

## Install the scanners

Aegis coordinates scanners - it does not bundle them. Install only the ones you actually use; `aegis doctor` will tell you which are missing once you configure them.

| Scanner          | Purpose              | Install                                                                                  |
| ---------------- | -------------------- | ---------------------------------------------------------------------------------------- |
| `gitleaks`       | Secret detection     | [github.com/gitleaks/gitleaks](https://github.com/gitleaks/gitleaks)                     |
| `opengrep`       | SAST (pattern match) | [github.com/opengrep/opengrep](https://github.com/opengrep/opengrep)                     |
| `osv-scanner`    | Dependency CVEs      | [github.com/google/osv-scanner](https://github.com/google/osv-scanner)                   |
| `biome`          | JS/TS lint + format  | [biomejs.dev](https://biomejs.dev/)                                                      |
| `ruff`           | Python lint + format | [docs.astral.sh/ruff](https://docs.astral.sh/ruff/)                                      |
| `golangci-lint`  | Go linting           | [golangci-lint.run](https://golangci-lint.run/)                                          |
| `shellcheck`     | Shell linting        | [shellcheck.net](https://www.shellcheck.net/)                                            |

`gofmt` ships with the Go toolchain.

After installing, run `aegis doctor` to confirm each configured scanner is found **and** its published SHA256 matches the pin in your `aegis.yaml`. If either check fails, `aegis run` refuses to execute that scanner.

## Verify your installation

```bash
aegis version
# aegis 0.1.0  schema=1  commit=abc123  built=2026-04-22T10:15:00Z

aegis doctor
# ✓ gitleaks 8.18.2 found at /usr/local/bin/gitleaks
# ✓ hash matches pinned value
# …
```

If `aegis doctor` is happy, you are ready to [run your first scan](getting-started.md).
