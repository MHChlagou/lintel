# `lintel ignore`

Add a rule to `.lintel/allowlist.yaml`. See [baseline + allowlist](../baseline-allowlist.md#allowlist) for when to prefer this over a baseline or an inline ignore.

## Usage

```bash
lintel ignore add [flags]
lintel ignore list
lintel ignore remove --id <entry-id>
```

## `lintel ignore add`

| Flag           | Required | Purpose                                                            |
| -------------- | -------- | ------------------------------------------------------------------ |
| `--scanner`    | yes      | Scanner name (e.g. `gitleaks`, `golangci-lint`)                    |
| `--rule`       | yes      | Rule ID to suppress (e.g. `generic-api-key`, `errcheck`)           |
| `--files`      | no       | Glob restricting the suppression (e.g. `testdata/**`). Omit for repo-wide. |
| `--reason`     | yes      | Audit reason, min 8 chars                                          |

```bash
lintel ignore add \
  --scanner gitleaks \
  --rule generic-api-key \
  --files 'testdata/**' \
  --reason "Test fixtures contain placeholder keys"
```

Adding without `--reason` is refused with exit 2.

## `lintel ignore list`

Prints every entry in `.lintel/allowlist.yaml` with a computed stable `id`:

```text
id         scanner         rule              files             reason
a1b2c3d4   gitleaks        generic-api-key   testdata/**       Test fixtures contain placeholder keys
f9e8d7c6   golangci-lint   errcheck          internal/legacy/**  Legacy module scheduled for rewrite in Q3
```

## `lintel ignore remove`

```bash
lintel ignore remove --id a1b2c3d4
```

Removes the entry with the given id. IDs are stable across runs; they are a short hash of `(scanner, rule, files)`, not a line number.

## Authoring directly

The file is plain YAML - you can also edit it by hand. See [the allowlist docs](../baseline-allowlist.md#allowlist) for the schema. The CLI exists for tab-completion, reason enforcement, and avoiding typos.

## Exit codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | Entry added, listed, or removed                  |
| 2    | Missing required flag, invalid syntax, bad id    |
