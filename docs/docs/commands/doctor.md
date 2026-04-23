# `lintel doctor`

Verify that every scanner referenced in `lintel.yaml` is installed, at the pinned version, and matches the pinned SHA256.

## Usage

```bash
lintel doctor [flags]
```

## Flags

| Flag             | Default | Purpose                                              |
| ---------------- | ------- | ---------------------------------------------------- |
| `--quiet`        | false   | Emit only failures; silent on success                |
| `--format`       | pretty  | `pretty` or `json`                                   |

## Output

```text
lintel doctor

gitleaks       8.18.2  OK        hash matches
golangci-lint  1.61.0  OK        hash matches
osv-scanner    1.9.0   MISSING   binary not on $PATH
biome          1.9.4   MISMATCH  hash differs from pin

2/4 scanners ready. Run `lintel run` after resolving 2 issues.
```

`MISSING` means the binary was not found at the configured path. `MISMATCH` means the binary is present but its SHA256 does not match the pinned hash for this platform.

## When to run

- In CI, before `lintel run`, as a fast pre-flight.
- On a developer machine after installing or upgrading a scanner.
- In an on-call runbook when `lintel run` starts failing unexpectedly with exit 3.

## Updating pins

When `doctor` reports a MISMATCH because you intentionally upgraded a scanner:

1. Verify the new binary against the upstream release's own checksum / signature.
2. Update the `sha256` block in `lintel.yaml` for your platform(s).
3. Commit the pin change in its own commit (`chore(scanners): pin gitleaks 8.18.3`).

`lintel doctor --update-pins` will automate this in v1.1.

## Exit codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | All scanners resolved and hashes match           |
| 3    | At least one scanner is missing or mismatched    |
