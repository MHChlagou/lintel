# Scanners

Aegis coordinates the scanners below. Each page covers: what the scanner does, how Aegis invokes it, the check it belongs to, the stacks it serves, severity mapping, and upstream references.

| Scanner          | Check           | Stacks              | Purpose                              |
| ---------------- | --------------- | ------------------- | ------------------------------------ |
| [gitleaks](gitleaks.md)               | `secrets`        | all                 | Detect committed secrets             |
| [opengrep](opengrep.md)               | `malicious_code` | all                 | SAST via pattern matching            |
| [osv-scanner](osv-scanner.md)         | `dependencies`   | go, npm, python, others | Known-vulnerability dependency scan |
| [biome](biome.md)                     | `lint`, `format` | npm                 | JS/TS linter and formatter           |
| [ruff](ruff.md)                       | `lint`, `format` | python              | Fast Python linter and formatter     |
| [golangci-lint](golangci-lint.md)     | `lint`           | go                  | Go linter aggregator                 |
| [gofmt](gofmt.md)                     | `format`         | go                  | Canonical Go formatter               |
| [shellcheck](shellcheck.md)           | `lint`           | shell               | Shell script static analysis         |

## Adding your own

See [adding a scanner](../adding-a-scanner.md) for the contributor workflow. The short version: implement `Checker`, register it, pin hashes, add a test, add a page here.
