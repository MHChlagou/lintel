package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MHChlagou/lintel/internal/detect"
	"github.com/MHChlagou/lintel/internal/finding"
)

type Malicious struct{}

func (Malicious) Name() string                           { return "malicious_code" }
func (Malicious) Applicable(*detect.ProjectContext) bool { return true }
func (Malicious) RequiredBinaries() []string             { return []string{"opengrep"} }

// opengrepOutput matches the Semgrep/Opengrep JSON shape (--json).
type opengrepOutput struct {
	Results []struct {
		CheckID string `json:"check_id"`
		Path    string `json:"path"`
		Start   struct {
			Line int `json:"line"`
			Col  int `json:"col"`
		} `json:"start"`
		Extra struct {
			Message  string `json:"message"`
			Severity string `json:"severity"`
			Metadata struct {
				Fix string `json:"fix"`
			} `json:"metadata"`
			Lines string `json:"lines"`
		} `json:"extra"`
	} `json:"results"`
}

func (m Malicious) Run(ctx context.Context, in CheckInput) (CheckOutput, error) {
	cfg := in.Spec.Checks.MaliciousCode
	if !cfg.Enabled || cfg.Mode == "off" {
		return CheckOutput{}, nil
	}
	start := time.Now()
	rb, err := in.Resolver.Resolve(cfg.Engine)
	if err != nil {
		return CheckOutput{}, err
	}
	args := []string{"scan", "--json", "--quiet", "--disable-version-check", "--error"}
	for _, rs := range cfg.Rulesets {
		args = append(args, "--config="+rs)
	}
	for _, ex := range cfg.ExcludePaths {
		args = append(args, "--exclude="+ex)
	}
	if cfg.TimeoutSeconds > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", cfg.TimeoutSeconds))
	}
	if in.FullTree || len(in.StagedFiles) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, in.StagedFiles...)
	}
	cmd := exec.CommandContext(ctx, rb.Path, args...)
	cmd.Dir = in.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return CheckOutput{}, fmt.Errorf("opengrep: %w: %s", err, stderr.String())
		}
	}
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		return CheckOutput{Stats: Stats{DurationMS: msSince(start)}}, nil
	}
	var out opengrepOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return CheckOutput{}, fmt.Errorf("parse opengrep json: %w", err)
	}
	findings := make([]finding.Finding, 0, len(out.Results))
	for _, r := range out.Results {
		sev := mapSemgrepSeverity(r.Extra.Severity)
		findings = append(findings, finding.Finding{
			Check:      "malicious_code",
			RuleID:     r.CheckID,
			Severity:   sev,
			File:       r.Path,
			Line:       r.Start.Line,
			Column:     r.Start.Col,
			Message:    strings.TrimSpace(r.Extra.Message),
			Snippet:    truncate(r.Extra.Lines, 160),
			FixSuggest: r.Extra.Metadata.Fix,
			Engine:     rb.Name,
		})
	}
	return CheckOutput{
		Findings: findings,
		Stats:    Stats{DurationMS: msSince(start), FilesScanned: len(in.StagedFiles)},
	}, nil
}

func mapSemgrepSeverity(raw string) finding.Severity {
	switch strings.ToUpper(raw) {
	case "ERROR":
		return finding.SevHigh
	case "WARNING":
		return finding.SevMedium
	default:
		return finding.SevLow
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
