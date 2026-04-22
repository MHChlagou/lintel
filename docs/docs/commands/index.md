# CLI reference

Aegis' command-line interface is deliberately small. Each command has a single purpose; combining them is usually done in a Git hook or a CI script, not by stringing multiple `aegis` invocations together.

| Command          | Purpose                                                         |
| ---------------- | --------------------------------------------------------------- |
| [`init`](init.md)           | Create `.aegis/aegis.yaml` with secure defaults         |
| [`install`](install.md)     | Install Git hooks that dispatch to `aegis run`          |
| [`uninstall`](install.md#uninstall) | Remove Aegis-installed hooks                    |
| [`run`](run.md)             | Run all enabled checks (honors `--hook` and `--check`)  |
| [`doctor`](doctor.md)       | Verify scanner binaries, versions, and SHA256 hashes    |
| [`baseline`](baseline.md)   | Snapshot current findings into `.aegis/baseline.json`   |
| [`ignore`](ignore.md)       | Add a rule to `.aegis/allowlist.yaml`                   |
| [`fmt`](run.md#fmt-shortcut) | Shortcut for `aegis run --check format`                 |
| [`explain`](explain.md)     | Print documentation pointer for a rule                  |
| [`version`](version.md)     | Print `aegis` version and schema version                |

## Global flags

These flags are available on every command:

| Flag              | Default                | Purpose                                       |
| ----------------- | ---------------------- | --------------------------------------------- |
| `--config`        | `.aegis/aegis.yaml`    | Path to the config file                       |
| `--repo`          | `$PWD`                 | Repository root                               |
| `--output`        | `pretty`               | Output format: `pretty`, `json`, `sarif`, `junit` |
| `--no-color`      | auto                   | Disable ANSI colors                           |
| `--verbose`       | off                    | Include debug output                          |
| `--quiet`         | off                    | Only emit errors                              |
| `-h, --help`      |                        | Help for the command                          |

`--output sarif` and `--output junit` are v1.1 and produce an informative error on v1.0.

## Exit codes

See [exit codes reference](../reference/exit-codes.md). TL;DR:

| Code | Meaning                              |
| ---- | ------------------------------------ |
| 0    | Gate passed                          |
| 1    | Gate failed                          |
| 2    | Config / CLI error                   |
| 3    | Binary resolve or verification failed |
| 4    | Scanner crashed or timed out         |
