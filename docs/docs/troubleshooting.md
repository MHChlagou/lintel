# Troubleshooting

A short catalog of the things that go wrong in the real world, and what to do about each.

## `lintel run` exits 3 with "hash mismatch"

```text
error: scanner gitleaks: sha256 mismatch
  expected: e3b0c44298fc…
  got:      2fa7e88…
```

Meaning: the scanner binary on disk does not match the pin in `lintel.yaml`. Either you upgraded the scanner without updating the pin, or the binary was replaced unexpectedly.

Fix:

1. Confirm the scanner's legitimate upstream release hash for the version you intend to run.
2. Either downgrade the scanner to the pinned version, or update the pin to match the new legitimate release. The choice depends on which of the two is what you meant.
3. Re-run `lintel doctor` to confirm.

## `lintel run` exits 3 with "binary not found"

Either the path in `lintel.yaml` is wrong, or the binary is not on `$PATH`. Install the scanner or adjust the path.

## `lintel doctor` says version does not match

You have the right scanner, but a different version than the pin allows. Install the pinned version, or update the pin.

## Pre-commit hook does nothing

```bash
lintel doctor
# no hooks installed
```

Run `lintel install` from inside the repo. If you are inside a worktree or submodule, run `lintel install` at the parent repo - hooks in submodules are uncommon and not auto-managed.

Also check `.git/hooks/pre-commit` exists and is executable. If a sibling tool (Husky, lefthook) overwrote it, run `lintel install --force` to reinstall with a shim that dispatches to all hook managers.

## "No staged files" on a commit with changes

Lintel relies on `git diff --cached --name-only`. If a file is modified but not `git add`-ed, it is not staged, and Lintel will not scan it. This is the intended behavior.

## Scanner hangs the pre-commit

```yaml
timeouts:
  per_check: 30s
  total: 120s
```

Raise the total if your repo is large; lower the per-check if a specific scanner misbehaves. A chronically slow scanner is usually best moved to `pre-push` or CI only.

## CI passes but local hook fails

This almost always means one of:

1. CI is using a different `lintel.yaml` (separate branch, different `--config` path).
2. CI has different scanner versions installed, and `strict_versions: true` is catching it.
3. CI is scoped to different files (for example, running against the whole tree while the hook only scans staged changes).

Run `lintel run --verbose --dry-run` in both environments and diff the logged config + scope.

## JSON output contains unexpected fields

`lintel run --output json` emits a stable schema - but scanner-specific extra fields surface under a `raw` sub-object. If you're consuming the output, key off the top-level fields only; `raw` is not versioned.

## Running inside a container

A common footgun: the container has `gitleaks` but not at the pinned version. Two paths:

- **Strict**: pin the exact upstream release in your container image, run `lintel doctor` during image build so CI catches drift early.
- **Loose**: set `strict_versions: false` in the container's `lintel.yaml` overlay. The per-scanner warning still appears.

## Getting help

- [GitHub Discussions](https://github.com/MHChlagou/lintel/discussions) for questions and setup.
- [GitHub Issues](https://github.com/MHChlagou/lintel/issues) for bugs.
- Private Security Advisory for anything with security implications - see [SECURITY.md](https://github.com/MHChlagou/lintel/blob/main/SECURITY.md).
