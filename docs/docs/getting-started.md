# Getting started

This 5-minute walkthrough installs Lintel, wires it into a new Git repository, runs a scan, and explains what happened.

## Prerequisites

- Git 2.30+
- A Unix-like shell (bash, zsh, or PowerShell on Windows)
- For live scanning, at least one supported scanner binary on `$PATH`. The quickstart skips this by disabling scanner-requiring checks; the full [installation guide](installation.md) covers the scanners.

## 1. Install Lintel

Pick the install method that fits your environment. All three produce a
`lintel` binary on `$PATH`. See [installation](installation.md) for pinning
a version, alternate install paths, and manual SHA256 / cosign verification.

=== "macOS / Linux"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh | sh
    lintel version
    ```

    The script detects your OS and architecture, downloads the release
    binary, verifies its SHA256, and installs to `/usr/local/bin`
    (falling back to `$HOME/.local/bin` when that is not writable).

=== "Go toolchain"

    ```bash
    go install github.com/MHChlagou/lintel/cmd/lintel@latest
    lintel version
    ```

=== "Windows (PowerShell)"

    ```powershell
    iwr https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.ps1 -UseBasicParsing | iex
    lintel version
    ```

    Installs to `$env:USERPROFILE\bin\lintel.exe` by default. The script
    prints the command to add that directory to your user PATH if it is
    not already there.

## 2. Create a demo repo

```bash
mkdir lintel-demo && cd lintel-demo
git init -q
echo "console.log('hello')" > app.js
git add app.js
```

## 3. Initialize Lintel

```bash
lintel init
```

This creates `.lintel/lintel.yaml` with secure defaults. Open it - every section has a comment explaining what it controls. See [the configuration guide](configuration.md) for the full schema.

## 4. Install the Git hooks

```bash
lintel install
```

`pre-commit` and `pre-push` hooks are now dispatching to `lintel run --hook pre-commit` and `lintel run --hook pre-push`. If you had existing hooks, Lintel preserved them (see [install docs](commands/install.md)).

## 5. Run a scan

For the quickstart we disable the scanners that require external binaries, so the pipeline runs end-to-end with only `gofmt`-style built-ins available:

```bash
# Keep only lint + format enabled for this demo.
sed -i.bak \
  -e '/^  secrets:/,/^  [a-z]/ s/    enabled: true/    enabled: false/' \
  -e '/^  malicious_code:/,/^  [a-z]/ s/    enabled: true/    enabled: false/' \
  -e '/^  dependencies:/,/^  [a-z]/ s/    enabled: true/    enabled: false/' \
  .lintel/lintel.yaml

lintel run --output json | head -20
```

A successful run exits **0** and prints an empty `findings` array. If anything is reported, each finding points at the file, line, scanner, rule, and a remediation hint. Exit codes are documented in [exit codes](reference/exit-codes.md).

## 6. Try a real commit

```bash
git add .lintel/
git commit -m "feat: set up lintel"
```

The `pre-commit` hook runs Lintel only against the staged files. See [the pipeline](pipeline.md) for what happens inside.

---

## Where to next

- Configure real scanners: [Installation :material-arrow-right:](installation.md)
- Understand what Lintel does on each commit: [The pipeline :material-arrow-right:](pipeline.md)
- Run in CI: [CI integration :material-arrow-right:](ci-integration.md)
- Introduce Lintel to an existing repo with noise: [Baselines and allowlists :material-arrow-right:](baseline-allowlist.md)
