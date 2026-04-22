# Exit codes

Aegis uses a small, stable set of exit codes. Shell scripts and CI jobs can branch on them.

| Code | Name                        | Meaning                                                                 |
| ---- | --------------------------- | ----------------------------------------------------------------------- |
| `0`  | `ok`                        | Gate passed. No action needed.                                          |
| `1`  | `gate_failed`               | One or more thresholds in `gate.fail_on` were exceeded.                 |
| `2`  | `config_error`              | Config file malformed, missing required field, or CLI flag invalid.     |
| `3`  | `binary_error`              | A scanner binary was missing, the wrong version, or had a wrong SHA256. |
| `4`  | `scanner_error`             | A scanner crashed, hit its per-check timeout, or returned garbled output. |
| `5`  | `override_denied`           | `--override` refused (missing reason, or `protect_secrets` blocks it).  |
| `10` | `internal_error`            | Unexpected bug in Aegis. Please file an issue.                          |

## Using exit codes in a shell

```bash
aegis run --output json > /tmp/report.json
rc=$?
case "$rc" in
  0) echo "clean" ;;
  1) echo "gate failed" ;;
  2) echo "bad config"; exit 1 ;;
  3) echo "binary/supply-chain problem"; exit 1 ;;
  4) echo "scanner misbehaved"; exit 1 ;;
  *) echo "unexpected rc=$rc"; exit 1 ;;
esac
```

## Stability

Exit codes are part of the public contract. A change in this table between minor versions would be a breaking change and will be called out in `CHANGELOG.md` if it ever happens. Currently the exit codes have not changed since the initial release.
