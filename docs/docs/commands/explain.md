# `aegis explain`

Print the documentation pointer for a scanner rule. Aegis does not ship its own rule knowledge base — each scanner maintains its own — so `explain` is a thin redirect to the upstream rule's canonical URL plus any adapter-specific notes.

## Usage

```bash
aegis explain <scanner>/<rule>
```

## Examples

```bash
aegis explain gitleaks/generic-api-key
# gitleaks rule: generic-api-key
# Detects high-entropy strings that look like API keys.
# Upstream: https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml

aegis explain golangci-lint/errcheck
# golangci-lint rule: errcheck
# Flags unchecked error return values.
# Upstream: https://github.com/kisielk/errcheck
```

When a finding is reported, `aegis run` includes a short `remediation` hint. If you want the full upstream doc, `aegis explain` gives the pointer.

## Listing rules

```bash
aegis explain gitleaks                     # lists rules Aegis knows about for gitleaks
aegis explain                              # lists all scanners
```

## Exit codes

| Code | Meaning                                                   |
| ---- | --------------------------------------------------------- |
| 0    | Explanation printed                                       |
| 2    | Unknown scanner or rule                                   |
