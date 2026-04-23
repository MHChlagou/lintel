# `lintel version`

Print the version, schema version, commit hash, and build timestamp.

## Usage

```bash
lintel version [flags]
```

## Flags

| Flag         | Default | Purpose                                 |
| ------------ | ------- | --------------------------------------- |
| `--format`   | pretty  | `pretty` or `json`                      |

## Output

```text
lintel 0.1.0  schema=1  commit=abc1234  built=2026-04-22T10:15:00Z  go1.25.3
```

- `schema=1` is the config schema version the binary expects in `lintel.yaml`.
- `commit` is the short Git SHA this binary was built from.
- `built` is the UTC build timestamp, set via `-ldflags` at build time.

## JSON

```bash
lintel version --format json
```

```json
{
  "version": "0.1.0",
  "schema": 1,
  "commit": "abc1234",
  "built": "2026-04-22T10:15:00Z",
  "go": "go1.25.3"
}
```

## Exit codes

| Code | Meaning                        |
| ---- | ------------------------------ |
| 0    | Always                         |
