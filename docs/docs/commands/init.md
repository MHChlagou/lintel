# `aegis init`

Create `.aegis/aegis.yaml` with secure defaults in the current repository.

## Usage

```bash
aegis init [flags]
```

## Flags

| Flag            | Default | Purpose                                           |
| --------------- | ------- | ------------------------------------------------- |
| `--force`       | false   | Overwrite an existing `.aegis/aegis.yaml`         |
| `--minimal`     | false   | Emit a minimal config (defaults implicit) rather than the fully-annotated starter |
| `--stack`       | auto    | Comma-separated stacks (e.g. `go,npm`); overrides auto-detect |

## What it does

1. Verifies you are inside a Git repository (or refuses).
2. Creates `.aegis/` if it does not exist.
3. Writes `.aegis/aegis.yaml` seeded from the stack(s) detected in the repo (or the stacks passed with `--stack`).
4. Appends `.aegis/overrides.log` to `.gitignore` if it is not already ignored.
5. Prints next steps (`aegis install`, `aegis run`).

The generated config is fully commented so new users can read the file top-to-bottom and understand what each setting does. If you prefer a lean file, pass `--minimal`.

## Example

```bash
cd my-service
aegis init --stack go
# wrote .aegis/aegis.yaml
# next: aegis install  (to install git hooks)
#       aegis run      (to scan now)
```

## Exit codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | Config written                                   |
| 2    | Not a Git repo, or `.aegis/aegis.yaml` exists and `--force` was not passed |
