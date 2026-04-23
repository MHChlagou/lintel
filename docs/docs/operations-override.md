# Override and audit log

Sometimes a commit must land despite a finding - a rotation in progress, a known-good placeholder, an incident response. Lintel supports this, but every bypass leaves an auditable trail.

## What can be overridden

| Control                           | Can be overridden?                                   |
| --------------------------------- | ---------------------------------------------------- |
| `gate.fail_on` thresholds         | Yes, via `LINTEL_OVERRIDE` (one-shot) or `--override`. |
| Single finding                    | Yes, via inline `lintel:ignore` with a reason.        |
| Check disable (`enabled: false`)  | Yes, per-check - **except** `secrets` when `protect_secrets: true` (default). |
| `secrets` check when `protect_secrets: true` | **No.** This is deliberate. See below.    |
| Binary hash mismatch              | No. Update the pin in `lintel.yaml` instead.          |
| `--no-verify` on the Git hook     | Lintel still runs a minimal protected pass - see [`spec.md` §13.3](reference/spec.md#133-no-verify-defense). |

## Using a one-shot override

```bash
LINTEL_OVERRIDE_REASON="rotating creds, ticket SEC-1234" \
LINTEL_OVERRIDE=true \
  git commit -m "fix: rotate API token"
```

Or via the CLI:

```bash
lintel run --override --reason "rotating creds, ticket SEC-1234"
```

Both paths require a `reason` of at least 8 characters. Lintel refuses the override otherwise.

## The audit log

Every override is appended to `.lintel/overrides.log` in plain text. It is append-only and gitignored by default - if you want it versioned, remove it from `.gitignore` explicitly. Each line:

```
2026-04-22T10:15:00Z  user=alice@example.com  commit=pending  checks=[secrets,lint]  reason="rotating creds, ticket SEC-1234"
```

The format is append-only and one line per override. For tamper-evident logging (append-only on a remote) you'll want to forward the log to your ingest pipeline of choice - Lintel does not ship a built-in forwarder.

## `protect_secrets`

```yaml
protect_secrets: true   # default
```

When true, the `secrets` check cannot be disabled by any mechanism:

- `checks.secrets.enabled: false` in `lintel.yaml` is refused with an exit 2.
- `LINTEL_SKIP_CHECKS=secrets` is ignored.
- `--override` skips the gate but still **runs** the secrets scanner and still **logs** findings.

Turn `protect_secrets` off only if your threat model genuinely requires it - for example, a quarantined repository with intentionally committed test fixtures. Prefer `paths.exclude` for that case instead.

## Why mandatory reasons

A bypass without a reason is indistinguishable from an accident. Mandating a free-text reason:

- Creates a trail for the eventual post-incident review.
- Forces a moment of intentionality before the bypass.
- Lets a policy layer (CI, code review) pattern-match reasons for compliance checks.

`lintel` does not judge the *content* of the reason - 8 characters and non-whitespace is the full validation. Your review process judges it.

## Bypass in CI

CI should not carry a blanket `LINTEL_OVERRIDE=true`. If a CI run needs to bypass, do it at the job level with a reason pinned to the pull request:

```yaml
- run: LINTEL_OVERRIDE=true LINTEL_OVERRIDE_REASON="release freeze; see PR body" lintel run
  if: contains(github.event.pull_request.labels.*.name, 'bypass-lintel')
```

This keeps the escape hatch visible and reviewable via a label rather than an implicit environment.
