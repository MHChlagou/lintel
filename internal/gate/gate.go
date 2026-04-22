// Package gate decides which findings are blocking based on mode + severity +
// filter layers (allowlist, baseline, warn_paths, inline ignores).
package gate

import (
	"strings"

	"github.com/aegis-sec/aegis/internal/config"
	"github.com/aegis-sec/aegis/internal/filter"
	"github.com/aegis-sec/aegis/internal/finding"
	"github.com/bmatcuk/doublestar/v4"
)

// Apply runs all filter layers in one pass and returns the mutated findings.
// Call order matters: allowlist → baseline → warn_paths → inline ignore →
// severity threshold → mode.
func Apply(spec *config.Spec, al *filter.Allowlist, base *filter.Baseline, repoRoot string, in []finding.Finding) []finding.Finding {
	out := make([]finding.Finding, 0, len(in))
	for _, f := range in {
		// 1. Allowlist (if covered, drop the finding entirely).
		if al != nil && allowMatch(al, f) {
			continue
		}
		// 2. Baseline (if in baseline, drop — not a new finding).
		if base != nil && base.Contains(f) {
			continue
		}
		// 3. warn_paths (demote to non-blocking on path match).
		demoted := false
		if f.Check == "secrets" {
			for _, pat := range spec.Checks.Secrets.WarnPaths {
				if m, _ := doublestar.PathMatch(pat, f.File); m {
					demoted = true
					break
				}
			}
		}
		// 4. Inline ignore: rely on File+Line. Best-effort — errors are ignored.
		if f.File != "" && f.Line > 0 && !demoted {
			marker := spec.Checks.Secrets.InlineIgnore
			if marker == "" {
				marker = "aegis:ignore-secret"
			}
			if ok, missingReason, _ := filter.InlineIgnored(f.File, f.Line, marker, f.RuleID); ok && !missingReason {
				continue
			}
		}
		// 5. Apply mode + severity to set Blocking.
		f.Blocking = decideBlocking(spec, f) && !demoted
		out = append(out, f)
	}
	return out
}

func allowMatch(al *filter.Allowlist, f finding.Finding) bool {
	for _, e := range al.Entries {
		if e.Expired() {
			continue
		}
		if e.Matches(f.Check, f.RuleID, f.File) {
			return true
		}
	}
	return false
}

// decideBlocking mirrors the per-check block rules from §8. Lint/format never
// block unless the user opts in via mode=block.
func decideBlocking(spec *config.Spec, f finding.Finding) bool {
	switch f.Check {
	case "secrets":
		return spec.Checks.Secrets.Mode == config.ModeBlock
	case "malicious_code":
		if spec.Checks.MaliciousCode.Mode != config.ModeBlock {
			return false
		}
		threshold := finding.ParseSeverity(spec.Checks.MaliciousCode.SeverityThreshold)
		return f.Severity.Rank() >= threshold.Rank()
	case "dependencies":
		if spec.Checks.Dependencies.Mode != config.ModeBlock {
			return false
		}
		for _, s := range spec.Checks.Dependencies.BlockSeverity {
			if strings.EqualFold(string(f.Severity), s) {
				return true
			}
		}
		return false
	case "lint":
		return spec.Checks.Lint.Mode == config.ModeBlock
	case "format":
		return spec.Checks.Format.Mode == "block"
	}
	return false
}
