# shellcheck

**Upstream:** [shellcheck.net](https://www.shellcheck.net/)
**Check:** `lint`
**Stacks:** `shell`

## What it does

`shellcheck` is a static analyzer for shell scripts. Aegis uses it for the `lint` check on `shell` stack files.

## How Aegis invokes it

```text
shellcheck --format=json1 --severity=style <staged shell files>
```

- `--severity=style`: report everything from `style` up. Aegis maps down to its three-level severity scale below.

## Severity mapping

| shellcheck level | Aegis severity |
| ---------------- | -------------- |
| `error`          | `error`        |
| `warning`        | `warn`         |
| `info`           | `info`         |
| `style`          | `info`         |

## Configuration

```yaml
scanners:
  shellcheck:
    path: shellcheck
    version: "0.10.0"
    sha256:
      linux/amd64: "…"
```

`shellcheck` reads `.shellcheckrc` for per-repo config. Aegis does not intermediate.

## Which files count as shell?

Aegis classifies a file as `shell` if:

- Extension is `.sh`, `.bash`, `.zsh`, `.ksh`.
- OR the first line is a shebang matching `#!/.*sh`.

Files without a shebang and without a shell extension are skipped, even if they contain shell-looking content. Add a shebang if you want them scanned.

## Notable rule suppressions

- Disable `SC2086` (word splitting) only if you have verified the script is safe — this is the most common source of shell bugs.
- Use `# shellcheck disable=SCxxxx` with a comment explaining why, not globally.
