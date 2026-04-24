# Installation

Lintel ships as a single static binary. There is no runtime to install, no Node / Python / Ruby dependency, and no daemon - just the `lintel` executable and the scanner binaries you want it to coordinate.

## Supported platforms

| OS      | amd64          | arm64          |
| ------- | -------------- | -------------- |
| Linux   | :material-check: | :material-check: |
| macOS   | :material-check: | :material-check: |
| Windows | :material-check: | :material-minus: |

## Install with the release script (recommended)

The install scripts detect your OS and architecture, download the
right release asset, verify its SHA256, and — if `cosign` is on
`$PATH` — verify the Sigstore signature. They never modify `$PATH`
behind your back.

=== "macOS / Linux"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh
    ```

    Flags and env vars the script accepts:

    | Flag              | Env var                 | Default                              |
    | ----------------- | ----------------------- | ------------------------------------ |
    | `--version <tag>` | `LINTEL_VERSION`         | `latest`                             |
    | `--install-dir`   | `LINTEL_INSTALL_DIR`     | `/usr/local/bin` (→ `$HOME/.local/bin` when not writable and no sudo) |
    | `--no-cosign`     | `LINTEL_VERIFY_COSIGN=false` | auto (verify when cosign is on PATH) |

    ```bash
    # Pin a specific version
    curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh -s -- --version v0.2.2

    # Install into a user-local path without sudo
    LINTEL_INSTALL_DIR="$HOME/.local/bin" \
      curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh
    ```

=== "Windows (PowerShell)"

    ```powershell
    iwr https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.ps1 -UseBasicParsing | iex
    ```

    Parameters (pass by downloading and running the script, or via env
    var when piping):

    | Parameter         | Env var             | Default                      |
    | ----------------- | ------------------- | ---------------------------- |
    | `-Version`        | `LINTEL_VERSION`     | `latest`                     |
    | `-InstallDir`     | `LINTEL_INSTALL_DIR` | `$env:USERPROFILE\bin`       |
    | `-NoCosign`       | —                   | auto                         |

    ```powershell
    # Pin a specific version
    $env:LINTEL_VERSION = 'v0.2.2'
    iwr https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.ps1 -UseBasicParsing | iex
    ```

## Install from a release manually

If you prefer to run the steps yourself or your environment blocks the
`curl | sh` pattern:

1. Download the binary for your platform from the [releases page](https://github.com/MHChlagou/lintel/releases/latest).
2. Verify the SHA256 checksum (a `.sha256` file sits next to every binary).
3. Optionally verify the Sigstore signature - see [supply-chain](supply-chain.md).
4. Move the binary onto your `$PATH` (commonly `/usr/local/bin` on Unix).
5. Run `lintel version` to confirm.

```bash
sha256sum -c lintel-linux-amd64.sha256
chmod +x lintel-linux-amd64
sudo mv lintel-linux-amd64 /usr/local/bin/lintel
lintel version
```

## Install via Go

Useful for development or for systems where you prefer to track `main`.

```bash
go install github.com/MHChlagou/lintel/cmd/lintel@latest
```

The binary lands in `$(go env GOBIN)` or `$(go env GOPATH)/bin`.

## Upgrading

### Check for a newer release

```bash
lintel upgrade
```

This queries the GitHub Releases API, compares against your running
version, and — if a newer release exists — prints the release notes
and a ready-to-paste upgrade command for your OS and architecture.
It never modifies the installed binary.

### Apply the upgrade

Re-run the install script. It overwrites the existing binary in place
after verifying the new SHA256 (and cosign signature, if available).

=== "macOS / Linux"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh
    # Or pin: ... | sh -s -- --version v0.2.2
    ```

=== "Go toolchain"

    ```bash
    go install github.com/MHChlagou/lintel/cmd/lintel@latest
    # Or pin: go install github.com/MHChlagou/lintel/cmd/lintel@v0.2.2
    ```

=== "Windows (PowerShell)"

    ```powershell
    # Close any running lintel.exe first — Windows locks running binaries.
    iwr https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.ps1 -UseBasicParsing | iex
    ```

### What does not need to change on upgrade

- **Git hooks** (`.git/hooks/pre-commit`, `pre-push`) dispatch to the `lintel`
  binary via `$PATH`; they keep working after the binary is replaced.
- **`~/.lintel/bin/<scanner>`** files are independent of the lintel version.
- **`.lintel/lintel.yaml`** is stable across v0.x — new fields are opt-in.

Run `lintel doctor` after upgrading to confirm every scanner still resolves.
If your `lintel.yaml` has no `sha256` entries and you rely on the embedded
pin-database fallback, the new version may ship different pins — `doctor`
will tell you immediately.

### Verifying the downloaded binary

Every release ships a `.sha256` sidecar and a `.sigstore` bundle. Both are
worth checking if you did not install via `go install`:

```bash
ver=v0.2.2
base="https://github.com/MHChlagou/lintel/releases/download/${ver}"
curl -fsSL "${base}/lintel-linux-amd64.sha256"   -o lintel.sha256
curl -fsSL "${base}/lintel-linux-amd64.sigstore" -o lintel.sigstore

sha256sum -c <(printf '%s  lintel\n' "$(cat lintel.sha256)")

cosign verify-blob --bundle lintel.sigstore \
  --certificate-identity-regexp='^https://github.com/MHChlagou/lintel/' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  lintel
```

## Install the scanners

Lintel coordinates scanners - it does not bundle them. Install only the ones you actually use; `lintel doctor` will tell you which are missing once you configure them.

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

After installing, run `lintel doctor` to confirm each configured scanner is found **and** its published SHA256 matches the pin in your `lintel.yaml`. If either check fails, `lintel run` refuses to execute that scanner.

## Verify your installation

```bash
lintel version
# lintel 0.1.0  schema=1  commit=abc123  built=2026-04-22T10:15:00Z

lintel doctor
# ✓ gitleaks 8.18.2 found at /usr/local/bin/gitleaks
# ✓ hash matches pinned value
# …
```

If `lintel doctor` is happy, you are ready to [run your first scan](getting-started.md).
