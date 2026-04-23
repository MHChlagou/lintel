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

type Secrets struct{}

func (Secrets) Name() string                           { return "secrets" }
func (Secrets) Applicable(*detect.ProjectContext) bool { return true }
func (s Secrets) RequiredBinaries() []string {
	return []string{"gitleaks"} // default engine; adapter switches below
}

// Run dispatches to the configured engine. Only gitleaks is wired in v1.
func (s Secrets) Run(ctx context.Context, in CheckInput) (CheckOutput, error) {
	cfg := in.Spec.Checks.Secrets
	if !cfg.Enabled || cfg.Mode == "off" {
		return CheckOutput{}, nil
	}
	start := time.Now()
	switch strings.ToLower(cfg.Engine) {
	case "gitleaks", "":
		return s.runGitleaks(ctx, in, start)
	default:
		return CheckOutput{}, fmt.Errorf("secrets engine %q not supported in v1 (use gitleaks)", cfg.Engine)
	}
}

// gitleaksResult matches the fields we consume from gitleaks JSON output.
type gitleaksResult struct {
	Description string  `json:"Description"`
	RuleID      string  `json:"RuleID"`
	Match       string  `json:"Match"`
	Secret      string  `json:"Secret"`
	File        string  `json:"File"`
	StartLine   int     `json:"StartLine"`
	StartColumn int     `json:"StartColumn"`
	Entropy     float64 `json:"Entropy"`
}

func (s Secrets) runGitleaks(ctx context.Context, in CheckInput, start time.Time) (CheckOutput, error) {
	rb, err := in.Resolver.Resolve("gitleaks")
	if err != nil {
		return CheckOutput{}, err
	}
	// `protect --staged` is the pre-commit path; `detect` is used on pre-push.
	var args []string
	if in.Hook == "pre-push" {
		args = []string{"detect", "--no-banner", "--report-format=json", "--report-path=/dev/stdout", "--redact"}
	} else {
		args = []string{"protect", "--no-banner", "--staged", "--report-format=json", "--report-path=/dev/stdout", "--redact"}
	}
	if r := in.Spec.Checks.Secrets.Rules; r != "" {
		args = append(args, "--config="+r)
	}
	cmd := exec.CommandContext(ctx, rb.Path, args...)
	cmd.Dir = in.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	// gitleaks exits non-zero when leaks are found; not a real error.
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return CheckOutput{}, fmt.Errorf("gitleaks: %w: %s", err, stderr.String())
		}
	}
	// Output: a JSON array; gitleaks sometimes prints `null` when clean.
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 || string(raw) == "null" {
		return CheckOutput{Stats: Stats{DurationMS: msSince(start)}}, nil
	}
	// Skip log lines that may appear before the JSON payload.
	if i := bytes.IndexAny(raw, "[{"); i > 0 {
		raw = raw[i:]
	}
	var results []gitleaksResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return CheckOutput{}, fmt.Errorf("parse gitleaks json: %w", err)
	}
	findings := make([]finding.Finding, 0, len(results))
	for _, r := range results {
		findings = append(findings, finding.Finding{
			Check:    "secrets",
			RuleID:   r.RuleID,
			Severity: finding.SevHigh,
			File:     r.File,
			Line:     r.StartLine,
			Column:   r.StartColumn,
			Message:  r.Description,
			Snippet:  redact(r.Secret),
			Engine:   "gitleaks",
		})
	}
	return CheckOutput{
		Findings: findings,
		Stats:    Stats{DurationMS: msSince(start), FilesScanned: len(in.StagedFiles)},
	}, nil
}

// redact never prints a secret's full value. First/last 4 chars only.
func redact(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

func msSince(t time.Time) int { return int(time.Since(t) / time.Millisecond) }
