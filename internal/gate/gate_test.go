package gate

import (
	"testing"

	"github.com/MHChlagou/lintel/internal/config"
	"github.com/MHChlagou/lintel/internal/filter"
	"github.com/MHChlagou/lintel/internal/finding"
)

func baseSpec() *config.Spec {
	return &config.Spec{
		Checks: config.Checks{
			Secrets: config.SecretsCheck{
				Mode: config.ModeBlock,
				WarnPaths: []string{
					"**/*_test.go",
					"**/fixtures/**",
				},
				InlineIgnore: "lintel:ignore-secret",
			},
			MaliciousCode: config.MaliciousCodeCheck{
				Mode:              config.ModeBlock,
				SeverityThreshold: "ERROR",
			},
			Dependencies: config.DependenciesCheck{
				Mode:          config.ModeBlock,
				BlockSeverity: []string{"CRITICAL", "HIGH"},
			},
			Lint:   config.LintCheck{Mode: config.ModeWarn},
			Format: config.FormatCheck{Mode: config.ModeFix},
		},
	}
}

func TestSecretsBlocking(t *testing.T) {
	spec := baseSpec()
	fs := []finding.Finding{
		{Check: "secrets", File: "src/main.go", Line: 10, RuleID: "aws-key", Severity: finding.SevHigh},
	}
	got := Apply(spec, nil, nil, "", fs)
	if len(got) != 1 || !got[0].Blocking {
		t.Fatalf("want blocking, got %+v", got)
	}
}

func TestSecretsWarnPathDemotes(t *testing.T) {
	spec := baseSpec()
	fs := []finding.Finding{
		{Check: "secrets", File: "src/main_test.go", Line: 10, RuleID: "aws-key", Severity: finding.SevHigh},
	}
	got := Apply(spec, nil, nil, "", fs)
	if len(got) != 1 || got[0].Blocking {
		t.Fatalf("warn_paths should demote; got %+v", got)
	}
}

func TestDependenciesSeverityFilter(t *testing.T) {
	spec := baseSpec()
	fs := []finding.Finding{
		{Check: "dependencies", RuleID: "CVE-1", Severity: finding.SevMedium},
		{Check: "dependencies", RuleID: "CVE-2", Severity: finding.SevHigh},
	}
	got := Apply(spec, nil, nil, "", fs)
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d", len(got))
	}
	var blocking, nonBlocking int
	for _, f := range got {
		if f.Blocking {
			blocking++
		} else {
			nonBlocking++
		}
	}
	if blocking != 1 || nonBlocking != 1 {
		t.Fatalf("want 1 blocking + 1 non-blocking; got b=%d nb=%d", blocking, nonBlocking)
	}
}

func TestAllowlistDropsFinding(t *testing.T) {
	spec := baseSpec()
	al := &filter.Allowlist{Entries: []filter.AllowEntry{{Rule: "aws-key", Reason: "ok"}}}
	fs := []finding.Finding{
		{Check: "secrets", File: "src/main.go", RuleID: "aws-key", Severity: finding.SevHigh},
	}
	got := Apply(spec, al, nil, "", fs)
	if len(got) != 0 {
		t.Fatalf("allowlist entry should drop finding; got %+v", got)
	}
}

func TestLintNeverBlocksByDefault(t *testing.T) {
	spec := baseSpec()
	fs := []finding.Finding{
		{Check: "lint", RuleID: "E501", Severity: finding.SevHigh, File: "a.py", Line: 1},
	}
	got := Apply(spec, nil, nil, "", fs)
	if len(got) != 1 || got[0].Blocking {
		t.Fatalf("lint should not block in warn mode; got %+v", got)
	}
}

func TestMaliciousBelowThresholdNotBlocking(t *testing.T) {
	spec := baseSpec()
	fs := []finding.Finding{
		{Check: "malicious_code", RuleID: "INFO-1", Severity: finding.SevLow, File: "a.go", Line: 1},
	}
	got := Apply(spec, nil, nil, "", fs)
	if len(got) != 1 || got[0].Blocking {
		t.Fatalf("below threshold should not block; got %+v", got)
	}
}
