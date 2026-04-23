# Environment variables

Lintel recognizes a small number of environment variables. They override the corresponding config file values at runtime and are documented here in full - no hidden knobs.

## Runtime behavior

| Variable                  | Effect                                                                    |
| ------------------------- | ------------------------------------------------------------------------- |
| `LINTEL_CONFIG`            | Alternate path to the config file. Equivalent to `--config`.              |
| `LINTEL_REPO`              | Alternate repo root. Equivalent to `--repo`.                              |
| `LINTEL_OUTPUT`            | Output format: `pretty` | `json` | `sarif` | `junit`.                     |
| `LINTEL_NO_COLOR` = `1`    | Disable ANSI colors. Also respects the cross-tool `NO_COLOR` env var.     |
| `LINTEL_VERBOSE` = `1`     | Equivalent to `--verbose`.                                                |
| `LINTEL_QUIET` = `1`       | Equivalent to `--quiet`.                                                  |

## Override and audit

| Variable                    | Effect                                                                |
| --------------------------- | --------------------------------------------------------------------- |
| `LINTEL_OVERRIDE` = `true`   | Enables gate override; requires a reason.                             |
| `LINTEL_OVERRIDE_REASON`     | Reason string for the override (min 8 characters).                    |

See [override and audit log](../operations-override.md).

## Tuning

| Variable                    | Effect                                                                |
| --------------------------- | --------------------------------------------------------------------- |
| `LINTEL_MAX_PARALLEL`        | Integer. Overrides `concurrency.max_parallel` in `lintel.yaml`.        |
| `LINTEL_TIMEOUT_TOTAL`       | Duration (e.g. `2m`, `90s`). Overrides `timeouts.total`.              |
| `LINTEL_TIMEOUT_PER_CHECK`   | Duration. Overrides `timeouts.per_check`.                             |
| `LINTEL_SKIP_CHECKS`         | Comma-separated check names to skip (`lint,format`). Ignored for `secrets` when `protect_secrets: true`. |

## Build-time identifiers

These are set by the release workflow via `-ldflags` and surface in `lintel version`. They are not consumed at runtime as environment variables; they are baked in.

- `github.com/MHChlagou/lintel/internal/version.Version`
- `github.com/MHChlagou/lintel/internal/version.Commit`
- `github.com/MHChlagou/lintel/internal/version.Date`

## Nothing else

Lintel does not read any other environment variables. If you find a behavior that appears to be env-driven and is not documented here, file a bug - that would be an undocumented hidden knob and a spec violation.
