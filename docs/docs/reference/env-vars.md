# Environment variables

Aegis recognizes a small number of environment variables. They override the corresponding config file values at runtime and are documented here in full — no hidden knobs.

## Runtime behavior

| Variable                  | Effect                                                                    |
| ------------------------- | ------------------------------------------------------------------------- |
| `AEGIS_CONFIG`            | Alternate path to the config file. Equivalent to `--config`.              |
| `AEGIS_REPO`              | Alternate repo root. Equivalent to `--repo`.                              |
| `AEGIS_OUTPUT`            | Output format: `pretty` | `json` | `sarif` | `junit`.                     |
| `AEGIS_NO_COLOR` = `1`    | Disable ANSI colors. Also respects the cross-tool `NO_COLOR` env var.     |
| `AEGIS_VERBOSE` = `1`     | Equivalent to `--verbose`.                                                |
| `AEGIS_QUIET` = `1`       | Equivalent to `--quiet`.                                                  |

## Override and audit

| Variable                    | Effect                                                                |
| --------------------------- | --------------------------------------------------------------------- |
| `AEGIS_OVERRIDE` = `true`   | Enables gate override; requires a reason.                             |
| `AEGIS_OVERRIDE_REASON`     | Reason string for the override (min 8 characters).                    |

See [override and audit log](../operations-override.md).

## Tuning

| Variable                    | Effect                                                                |
| --------------------------- | --------------------------------------------------------------------- |
| `AEGIS_MAX_PARALLEL`        | Integer. Overrides `concurrency.max_parallel` in `aegis.yaml`.        |
| `AEGIS_TIMEOUT_TOTAL`       | Duration (e.g. `2m`, `90s`). Overrides `timeouts.total`.              |
| `AEGIS_TIMEOUT_PER_CHECK`   | Duration. Overrides `timeouts.per_check`.                             |
| `AEGIS_SKIP_CHECKS`         | Comma-separated check names to skip (`lint,format`). Ignored for `secrets` when `protect_secrets: true`. |

## Build-time identifiers

These are set by the release workflow via `-ldflags` and surface in `aegis version`. They are not consumed at runtime as environment variables; they are baked in.

- `github.com/aegis-sec/aegis/internal/version.Version`
- `github.com/aegis-sec/aegis/internal/version.Commit`
- `github.com/aegis-sec/aegis/internal/version.Date`

## Nothing else

Aegis does not read any other environment variables. If you find a behavior that appears to be env-driven and is not documented here, file a bug — that would be an undocumented hidden knob and a spec violation.
