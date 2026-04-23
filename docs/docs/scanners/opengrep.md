# opengrep

**Upstream:** [github.com/opengrep/opengrep](https://github.com/opengrep/opengrep)
**Check:** `malicious_code`
**Stacks:** all (stack-agnostic; rules are per-language)

## What it does

`opengrep` is a pattern-matching SAST tool that finds problematic code constructs using a declarative rule language (compatible with the Semgrep rule ecosystem). It is the default for Lintel's `malicious_code` check.

## How Lintel invokes it

```text
opengrep scan \
  --config=<rules> \
  --json \
  --quiet \
  --metrics=off \
  <staged files>
```

- `--metrics=off`: no telemetry leaves the machine.
- `--config`: points at the rule pack configured in `lintel.yaml`. By default Lintel points at `p/security-audit`; override to vendor your own rules.

## Severity mapping

| Upstream severity | Lintel severity |
| ----------------- | -------------- |
| `ERROR`           | `error`        |
| `WARNING`         | `warn`         |
| `INFO`            | `info`         |

## Configuration

```yaml
scanners:
  opengrep:
    path: opengrep
    version: "1.0.0"
    rules:
      - "p/security-audit"
      - "./security/rules/"        # local rules live next to your code
    sha256:
      linux/amd64: "…"
```

## Rule pack management

Opengrep can download rule packs at runtime. This is convenient but creates a supply-chain dependency that Lintel does not verify. For production use, vendor rule packs into the repo and set `rules:` to the local path.

## Tuning noise

Rule packs are often noisy out of the box. Practical steps:

1. Start with one focused pack (`p/security-audit`).
2. Seed a baseline (`lintel baseline`) so current findings become the accepted state.
3. Add allowlist entries for each rule you decide is not relevant.
4. Consider forking the pack and pruning rules you don't use.
