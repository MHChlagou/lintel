# Examples

Each subdirectory is a reference `lintel.yaml` for a common repo shape.
Copy into your project as a starting point; pin the scanner versions + hashes to whatever you actually have installed.

| Example                                     | Shape                                                 |
| ------------------------------------------- | ----------------------------------------------------- |
| [`go-service/`](go-service/lintel.yaml)               | Single Go service with gofmt + golangci-lint + gitleaks + osv-scanner |
| [`typescript-monorepo/`](typescript-monorepo/lintel.yaml) | pnpm/yarn workspace using biome + gitleaks + osv-scanner         |
| [`python-lib/`](python-lib/lintel.yaml)               | Python library using ruff + gitleaks + osv-scanner    |

The scanner hashes in these examples are **placeholders** (`REPLACE_WITH_UPSTREAM_SHA256`) so you can't accidentally pin to an untrusted value by copy-paste. Replace them with real hashes from each scanner's upstream release before running `lintel doctor`.
