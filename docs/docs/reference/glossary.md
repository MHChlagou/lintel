# Glossary

**Adapter**
:   The Go code in `internal/checker/` that invokes a scanner, parses its output, and produces normalized `Finding`s.

**Allowlist**
:   A persistent list of (scanner, rule, path glob, reason) entries that suppress matching findings. Lives in `.lintel/allowlist.yaml`.

**Baseline**
:   A JSON snapshot of every finding at a point in time. Findings present in the baseline are excluded from the gate on future runs; new findings are not. Lives in `.lintel/baseline.json`.

**Check**
:   One of Lintel's five logical check categories: `secrets`, `malicious_code`, `dependencies`, `lint`, `format`. Each check has one or more scanners per stack.

**Fingerprint**
:   A stable hash of `(scanner, rule, file, line, normalized message)` used to match a finding across runs even after cosmetic code movement. Generated per finding in the normalization stage.

**Gate**
:   The final-stage evaluator that compares filtered findings against `gate.fail_on` thresholds and sets the process exit code.

**Hook**
:   A Git hook - `pre-commit` or `pre-push` - that `lintel install` manages.

**Inline ignore**
:   A magic comment (`lintel:ignore rule=X reason=Y`) in source code that suppresses a finding on the marked line or block. Requires a reason.

**Normalize**
:   The stage that converts each scanner's heterogeneous output into the uniform `Finding` struct.

**Override**
:   An operator-initiated bypass of the gate for a single run, gated by a mandatory reason and recorded in `.lintel/overrides.log`.

**Pin**
:   The combination of `version` + per-platform `sha256` fields in `lintel.yaml` that identifies the exact scanner binary Lintel is willing to execute.

**`protect_secrets`**
:   Top-level config flag (default `true`) that prevents the `secrets` check from being disabled by any mechanism.

**Scanner**
:   An external tool (`gitleaks`, `biome`, …) that Lintel coordinates. Lintel does not contain its own rules or databases.

**Stack**
:   A language or ecosystem tag (`go`, `npm`, `python`, `shell`, …) inferred from staged files and used to route scanners.

**`strict_versions`**
:   Top-level config flag (default `true`) that makes version / hash mismatches fatal.

**`warn_paths`**
:   Globs under `paths.warn_paths`. Findings in these paths are downgraded one severity level (and `info` becomes dropped entirely).
