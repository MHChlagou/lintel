# Getting started

This 5-minute walkthrough installs Aegis, wires it into a new Git repository, runs a scan, and explains what happened.

## Prerequisites

- Git 2.30+
- A Unix-like shell (bash, zsh, or PowerShell on Windows)
- For live scanning, at least one supported scanner binary on `$PATH`. The quickstart skips this by disabling scanner-requiring checks; the full [installation guide](installation.md) covers the scanners.

## 1. Install Aegis

=== "macOS / Linux"

    ```bash
    # Once releases are published:
    curl -fsSL https://github.com/aegis-sec/aegis/releases/latest/download/aegis-$(uname -s | tr A-Z a-z)-$(uname -m) -o /usr/local/bin/aegis
    chmod +x /usr/local/bin/aegis
    aegis version
    ```

=== "Go toolchain"

    ```bash
    go install github.com/aegis-sec/aegis/cmd/aegis@latest
    aegis version
    ```

=== "Windows (PowerShell)"

    ```powershell
    Invoke-WebRequest `
      -Uri https://github.com/aegis-sec/aegis/releases/latest/download/aegis-windows-amd64.exe `
      -OutFile $env:USERPROFILE\bin\aegis.exe
    aegis version
    ```

## 2. Create a demo repo

```bash
mkdir aegis-demo && cd aegis-demo
git init -q
echo "console.log('hello')" > app.js
git add app.js
```

## 3. Initialize Aegis

```bash
aegis init
```

This creates `.aegis/aegis.yaml` with secure defaults. Open it — every section has a comment explaining what it controls. See [the configuration guide](configuration.md) for the full schema.

## 4. Install the Git hooks

```bash
aegis install
```

`pre-commit` and `pre-push` hooks are now dispatching to `aegis run --hook pre-commit` and `aegis run --hook pre-push`. If you had existing hooks, Aegis preserved them (see [install docs](commands/install.md)).

## 5. Run a scan

For the quickstart we disable the scanners that require external binaries, so the pipeline runs end-to-end with only `gofmt`-style built-ins available:

```bash
# Keep only lint + format enabled for this demo.
sed -i.bak \
  -e '/^  secrets:/,/^  [a-z]/ s/    enabled: true/    enabled: false/' \
  -e '/^  malicious_code:/,/^  [a-z]/ s/    enabled: true/    enabled: false/' \
  -e '/^  dependencies:/,/^  [a-z]/ s/    enabled: true/    enabled: false/' \
  .aegis/aegis.yaml

aegis run --output json | head -20
```

A successful run exits **0** and prints an empty `findings` array. If anything is reported, each finding points at the file, line, scanner, rule, and a remediation hint. Exit codes are documented in [exit codes](reference/exit-codes.md).

## 6. Try a real commit

```bash
git add .aegis/
git commit -m "feat: set up aegis"
```

The `pre-commit` hook runs Aegis only against the staged files. See [the pipeline](pipeline.md) for what happens inside.

---

## Where to next

- Configure real scanners: [Installation :material-arrow-right:](installation.md)
- Understand what Aegis does on each commit: [The pipeline :material-arrow-right:](pipeline.md)
- Run in CI: [CI integration :material-arrow-right:](ci-integration.md)
- Introduce Aegis to an existing repo with noise: [Baselines and allowlists :material-arrow-right:](baseline-allowlist.md)
