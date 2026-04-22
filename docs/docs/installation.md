# Installation

Aegis ships as a single static binary. There is no runtime to install, no Node / Python / Ruby dependency, and no daemon — just the `aegis` executable and the scanner binaries you want it to coordinate.

## Supported platforms

| OS      | amd64          | arm64          |
| ------- | -------------- | -------------- |
| Linux   | :material-check: | :material-check: |
| macOS   | :material-check: | :material-check: |
| Windows | :material-check: | :material-minus: |

## Install from a release

1. Download the binary for your platform from the [releases page](https://github.com/aegis-sec/aegis/releases/latest).
2. Verify the SHA256 checksum (a `.sha256` file sits next to every binary).
3. Optionally verify the Sigstore signature — see [supply-chain](supply-chain.md).
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
go install github.com/aegis-sec/aegis/cmd/aegis@latest
```

The binary lands in `$(go env GOBIN)` or `$(go env GOPATH)/bin`.

## Install the scanners

Aegis coordinates scanners — it does not bundle them. Install only the ones you actually use; `aegis doctor` will tell you which are missing once you configure them.

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
