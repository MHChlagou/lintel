# `lintel install` / `lintel uninstall`

Install or remove the Git hooks that dispatch to `lintel run`.

## `lintel install`

### Usage

```bash
lintel install [flags]
```

### Flags

| Flag         | Default  | Purpose                                                  |
| ------------ | -------- | -------------------------------------------------------- |
| `--hooks`    | `pre-commit,pre-push` | Comma-separated list of hooks to install    |
| `--force`    | false    | Overwrite an existing foreign hook (see below)           |

### What it does

For each hook name (`pre-commit`, `pre-push`):

1. Checks whether `.git/hooks/<name>` exists.
2. If it is an **Lintel-managed** hook (identified by a magic marker), rewrites it with the current version's shim.
3. If it is a **foreign** hook (Husky, lefthook, hand-written):
    - Without `--force`: the foreign hook is preserved and Lintel adds itself as a delegating shim that calls the foreign hook first, then `lintel run --hook <name>`.
    - With `--force`: the foreign hook is replaced. A backup is saved to `.git/hooks/<name>.bak`.
4. Sets the file executable.

### Coexisting with other hook managers

Lintel's magic marker makes `install` idempotent and makes `uninstall` safe: Lintel will never remove a hook it did not install. If Husky is in charge of the hook, `lintel install` delegates; on `lintel uninstall`, the Lintel delegation is removed and the Husky hook remains.

### Example

```bash
lintel install
# installed pre-commit
# installed pre-push (delegating to husky)
```

## `lintel uninstall`

### Usage

```bash
lintel uninstall [flags]
```

### Flags

| Flag         | Default  | Purpose                                                  |
| ------------ | -------- | -------------------------------------------------------- |
| `--hooks`    | all Lintel-managed | Comma-separated list of hooks to remove         |

### What it does

For each hook Lintel installed:

1. If Lintel is the only content of the hook, deletes the hook file.
2. If Lintel was delegating to a foreign hook, removes the delegation and restores the foreign hook.
3. Leaves any non-Lintel hook untouched.

### Example

```bash
lintel uninstall
# removed pre-commit
# restored pre-push (husky)
```

## Exit codes

| Code | Meaning                                              |
| ---- | ---------------------------------------------------- |
| 0    | All requested hooks installed or uninstalled         |
| 2    | Not a Git repo; or foreign hook present and `--force` not set on install |
