# `aegis install` / `aegis uninstall`

Install or remove the Git hooks that dispatch to `aegis run`.

## `aegis install`

### Usage

```bash
aegis install [flags]
```

### Flags

| Flag         | Default  | Purpose                                                  |
| ------------ | -------- | -------------------------------------------------------- |
| `--hooks`    | `pre-commit,pre-push` | Comma-separated list of hooks to install    |
| `--force`    | false    | Overwrite an existing foreign hook (see below)           |

### What it does

For each hook name (`pre-commit`, `pre-push`):

1. Checks whether `.git/hooks/<name>` exists.
2. If it is an **Aegis-managed** hook (identified by a magic marker), rewrites it with the current version's shim.
3. If it is a **foreign** hook (Husky, lefthook, hand-written):
    - Without `--force`: the foreign hook is preserved and Aegis adds itself as a delegating shim that calls the foreign hook first, then `aegis run --hook <name>`.
    - With `--force`: the foreign hook is replaced. A backup is saved to `.git/hooks/<name>.bak`.
4. Sets the file executable.

### Coexisting with other hook managers

Aegis's magic marker makes `install` idempotent and makes `uninstall` safe: Aegis will never remove a hook it did not install. If Husky is in charge of the hook, `aegis install` delegates; on `aegis uninstall`, the Aegis delegation is removed and the Husky hook remains.

### Example

```bash
aegis install
# installed pre-commit
# installed pre-push (delegating to husky)
```

## `aegis uninstall`

### Usage

```bash
aegis uninstall [flags]
```

### Flags

| Flag         | Default  | Purpose                                                  |
| ------------ | -------- | -------------------------------------------------------- |
| `--hooks`    | all Aegis-managed | Comma-separated list of hooks to remove         |

### What it does

For each hook Aegis installed:

1. If Aegis is the only content of the hook, deletes the hook file.
2. If Aegis was delegating to a foreign hook, removes the delegation and restores the foreign hook.
3. Leaves any non-Aegis hook untouched.

### Example

```bash
aegis uninstall
# removed pre-commit
# restored pre-push (husky)
```

## Exit codes

| Code | Meaning                                              |
| ---- | ---------------------------------------------------- |
| 0    | All requested hooks installed or uninstalled         |
| 2    | Not a Git repo; or foreign hook present and `--force` not set on install |
