# biome

**Upstream:** [biomejs.dev](https://biomejs.dev/)
**Checks:** `lint`, `format`
**Stacks:** `npm`

## What it does

Biome is an all-in-one JavaScript / TypeScript toolchain that includes a linter and a formatter. Lintel uses it for both `lint` and `format` on `npm` stack files.

## How Lintel invokes it

Lint:

```text
biome lint --reporter=json <staged files>
```

Format check:

```text
biome format --reporter=json <staged files>
```

Biome is fast enough that running both on pre-commit is a non-issue for most repos.

## Severity mapping

| Biome severity | Lintel severity |
| -------------- | -------------- |
| `error`        | `error`        |
| `warning`      | `warn`         |
| `info`         | `info`         |

## Configuration

```yaml
scanners:
  biome:
    path: biome
    version: "1.9.4"
    sha256:
      linux/amd64: "…"
```

Biome reads its own `biome.json` config for rule enablement - Lintel does not intermediate the rules.

## Monorepo notes

In a workspace layout, Lintel invokes biome from the repo root so a single `biome.json` at the top-level applies everywhere. If your packages have their own configs, biome picks up the nearest one per file.
