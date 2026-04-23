package config

// DefaultSpecYAML is written by `lintel init`. Secure by default: block mode on
// secrets/malicious_code/dependencies; reason required; no-verify disallowed.
const DefaultSpecYAML = `version: 1

project:
  name: {{.ProjectName}}
  type: auto

binaries:
  gitleaks:
    command: gitleaks
    version: "8.28.0"
    install_hint: "https://github.com/gitleaks/gitleaks/releases/tag/v8.28.0"
  opengrep:
    command: opengrep
    version: "1.4.0"
    install_hint: "https://github.com/opengrep/opengrep/releases"
  osv-scanner:
    command: osv-scanner
    version: "2.0.3"
    install_hint: "https://github.com/google/osv-scanner/releases"
  biome:
    command: biome
    version: "1.9.4"
    install_hint: "https://biomejs.dev/"
  ruff:
    command: ruff
    version: "0.7.0"
    install_hint: "https://github.com/astral-sh/ruff/releases"
  golangci-lint:
    command: golangci-lint
    version: "1.61.0"
    install_hint: "https://github.com/golangci/golangci-lint/releases"
  gofmt:
    command: gofmt
    version: "0"
    install_hint: "ships with the Go toolchain"
  shellcheck:
    command: shellcheck
    version: "0.10.0"
    install_hint: "https://github.com/koalaman/shellcheck/releases"

checks:
  secrets:
    enabled: true
    engine: gitleaks
    mode: block
    scan:
      staged_only: true
      full_on_push: true
    warn_paths:
      - "**/*_test.{go,js,ts,py,rb}"
      - "**/test/**"
      - "**/__tests__/**"
      - "**/fixtures/**"
    inline_ignore: "lintel:ignore-secret"
  malicious_code:
    enabled: true
    engine: opengrep
    mode: block
    rulesets:
      - p/security-audit
      - p/owasp-top-ten
    severity_threshold: ERROR
    timeout_seconds: 60
    exclude_paths:
      - "vendor/**"
      - "node_modules/**"
      - "dist/**"
  dependencies:
    enabled: true
    engine: osv-scanner
    mode: block
    block_severity: [CRITICAL, HIGH]
    suggest_fix: true
    offline:
      enabled: true
      refresh_hours: 24
  lint:
    enabled: true
    mode: warn
    auto_fix: false
    tools:
      javascript: biome
      typescript: biome
      python: ruff
      go: golangci-lint
  format:
    enabled: true
    mode: check
    tools:
      javascript: biome
      typescript: biome
      python: ruff
      go: gofmt

scope:
  staged_only: true
  full_scan_for: [dependencies]
  exclude_paths:
    - ".git/**"
    - "vendor/**"
    - "node_modules/**"
    - "dist/**"
    - "build/**"
    - "*.min.js"

hooks:
  pre-commit:
    checks: [secrets, malicious_code, lint, format]
    fail_fast: false
  pre-push:
    checks: [secrets, dependencies]
    fail_fast: true

output:
  format: pretty
  group_by: check
  show_fix_suggestions: true
  color: auto
  verbosity: normal

override:
  env_var: LINTEL_SKIP
  allow_no_verify: false
  require_reason: true
  log_file: .lintel/overrides.log
  protect_secrets: true

performance:
  parallel: auto
  check_timeout_seconds: 120
  total_timeout_seconds: 300
  cache:
    enabled: true
    path: ~/.lintel/cache
    ttl_hours: 24

strict_versions: true
`

const DefaultAllowlistYAML = `# .lintel/allowlist.yaml
# Every entry requires a reason. Expired entries are ignored.
entries: []
`
