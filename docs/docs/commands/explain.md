# `lintel explain`

Print the documentation pointer for a scanner rule. Lintel does not ship its own rule knowledge base - each scanner maintains its own - so `explain` is a thin redirect to the upstream rule's canonical URL plus any adapter-specific notes.

## Usage

```bash
lintel explain <scanner>/<rule>
```

## Examples

```bash
lintel explain gitleaks/generic-api-key
# gitleaks rule: generic-api-key
# Detects high-entropy strings that look like API keys.
# Upstream: https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml

lintel explain golangci-lint/errcheck
# golangci-lint rule: errcheck
# Flags unchecked error return values.
# Upstream: https://github.com/kisielk/errcheck
```

When a finding is reported, `lintel run` includes a short `remediation` hint. If you want the full upstream doc, `lintel explain` gives the pointer.

## Listing rules

```bash
lintel explain gitleaks                     # lists rules Lintel knows about for gitleaks
lintel explain                              # lists all scanners
```

## Exit codes

| Code | Meaning                                                   |
| ---- | --------------------------------------------------------- |
| 0    | Explanation printed                                       |
| 2    | Unknown scanner or rule                                   |
