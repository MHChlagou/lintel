# Stacks and detection

A **stack** is a named language or ecosystem (`go`, `npm`, `python`, `shell`, etc.). Lintel uses the stack to decide which scanners are eligible for which files.

## How detection works

With `stacks.auto: true`, Lintel scans the staged (or working-tree, in non-hook mode) file list and infers stacks based on manifests and extensions:

| Stack    | Trigger                                                    |
| -------- | ---------------------------------------------------------- |
| `go`     | `go.mod` anywhere above the file, or `*.go` extension.     |
| `npm`    | `package.json`, `*.{js,jsx,ts,tsx,mjs,cjs}`.               |
| `python` | `pyproject.toml`, `setup.cfg`, `requirements*.txt`, `*.py`.|
| `shell`  | `*.sh`, `*.bash`, or a shebang line beginning with `#!/`. |

A file can belong to more than one stack (for example, a shell script inside an npm repo will be in both `npm` and `shell` for different scanners).

## Restricting stacks

You can pin stacks explicitly when auto-detect is noisy in a mixed repo:

```yaml
stacks:
  auto: true
  include: [go]              # only go scanners run, regardless of what else is staged
```

Or disable auto-detect entirely:

```yaml
stacks:
  auto: false
  include: [go, npm]
```

When `auto: false` and a staged file does not belong to any listed stack, it is skipped silently. Lintel does not warn on unclassified files - this is intentional so that mixed repos don't spam output.

## Stacks and scanners

Each scanner declares which stacks it serves. `biome` serves `npm`, `ruff` serves `python`, `golangci-lint` serves `go`. Some scanners (`gitleaks`, `opengrep`) are stack-agnostic and run against all staged files regardless.

The [scanner pages](scanners/index.md) document each scanner's stack mapping.

## Debugging detection

```bash
lintel run --verbose
# …
# stage=detect  staged=24  stacks=[go shell]  skipped=3
```

With `--verbose`, the detect stage logs the full per-file classification - useful when a file you expect to be scanned is being skipped.
