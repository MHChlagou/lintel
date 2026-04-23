# gofmt

**Upstream:** ships with the Go toolchain
**Check:** `format`
**Stacks:** `go`

## What it does

`gofmt` is the canonical Go formatter. It is the arbiter of Go code style - there is no configuration. Lintel uses it as the `format` check on Go files.

## How Lintel invokes it

```text
gofmt -l <staged .go files>
```

- `-l`: list files whose formatting differs from canonical. Lintel reports each listed file as a single `warn` finding (no line-level detail, because `gofmt` operates on the whole file).

## Severity mapping

| `gofmt` status | Lintel severity |
| -------------- | -------------- |
| file differs   | `warn`         |

Files that format cleanly produce no finding.

## Configuration

`gofmt` has no configuration. Lintel does not pin its hash because `gofmt` is part of the Go distribution you already verified when you installed Go.

```yaml
scanners:
  gofmt:
    path: gofmt
    version: ""     # shares the Go toolchain version
```

## Auto-fixing

```bash
lintel fmt --fix         # planned v1.1: run `gofmt -w`
gofmt -w .              # today: run upstream directly
```

In v1.0, `lintel fmt` reports only; you fix with the upstream tool or your editor.

## Relationship to `goimports`

`gofmt` does not manage imports. Many Go teams prefer `goimports` (which is a superset that also reformats import blocks). Lintel does not ship a `goimports` adapter in v1.0 because the two are near-identical in output - if you need `goimports`, set `scanners.gofmt.path: goimports` and the adapter will pass through unchanged.
