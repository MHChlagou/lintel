package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aegis-sec/aegis/internal/detect"
	"github.com/aegis-sec/aegis/internal/finding"
)

type Deps struct{}

func (Deps) Name() string                           { return "dependencies" }
func (Deps) Applicable(*detect.ProjectContext) bool { return true }
func (Deps) RequiredBinaries() []string             { return []string{"osv-scanner"} }

// osvOutput is the subset of osv-scanner JSON we consume.
type osvOutput struct {
	Results []struct {
		Source struct {
			Path string `json:"path"`
		} `json:"source"`
		Packages []struct {
			Package struct {
				Name      string `json:"name"`
				Version   string `json:"version"`
				Ecosystem string `json:"ecosystem"`
			} `json:"package"`
			Vulnerabilities []struct {
				ID       string   `json:"id"`
				Summary  string   `json:"summary"`
				Aliases  []string `json:"aliases"`
				Affected []struct {
					Ranges []struct {
						Events []struct {
							Introduced string `json:"introduced,omitempty"`
							Fixed      string `json:"fixed,omitempty"`
						} `json:"events"`
					} `json:"ranges"`
				} `json:"affected"`
				DatabaseSpecific struct {
					Severity string `json:"severity"`
				} `json:"database_specific"`
			} `json:"vulnerabilities"`
			Groups []struct {
				IDs         []string `json:"ids"`
				MaxSeverity string   `json:"max_severity"`
			} `json:"groups"`
		} `json:"packages"`
	} `json:"results"`
}

func (d Deps) Run(ctx context.Context, in CheckInput) (CheckOutput, error) {
	cfg := in.Spec.Checks.Dependencies
	if !cfg.Enabled || cfg.Mode == "off" {
		return CheckOutput{}, nil
	}
	start := time.Now()
	rb, err := in.Resolver.Resolve(cfg.Engine)
	if err != nil {
		return CheckOutput{}, err
	}
	args := []string{"--format=json", "-r", in.RepoRoot}
	if cfg.Offline.Enabled {
		args = append([]string{"--offline"}, args...)
	}
	cmd := exec.CommandContext(ctx, rb.Path, args...)
	cmd.Dir = in.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return CheckOutput{}, fmt.Errorf("osv-scanner: %w: %s", err, stderr.String())
		}
	}
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		return CheckOutput{Stats: Stats{DurationMS: msSince(start)}}, nil
	}
	var out osvOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return CheckOutput{}, fmt.Errorf("parse osv-scanner json: %w", err)
	}
	// Build ignore-set from spec
	ignored := map[string]bool{}
	for _, ic := range cfg.IgnoreCVEs {
		if !cveExpired(ic.Expires) {
			ignored[strings.ToUpper(ic.ID)] = true
		}
	}
	var findings []finding.Finding
	for _, res := range out.Results {
		for _, p := range res.Packages {
			for _, v := range p.Vulnerabilities {
				ids := append([]string{v.ID}, v.Aliases...)
				if anyIgnored(ids, ignored) {
					continue
				}
				fix := firstFixedVersion(v.Affected)
				findings = append(findings, finding.Finding{
					Check:      "dependencies",
					RuleID:     v.ID,
					Severity:   finding.ParseSeverity(v.DatabaseSpecific.Severity),
					File:       res.Source.Path,
					Message:    fmt.Sprintf("%s@%s — %s", p.Package.Name, p.Package.Version, v.Summary),
					FixSuggest: fixHint(p.Package.Name, p.Package.Version, fix),
					Engine:     rb.Name,
				})
			}
		}
	}
	return CheckOutput{
		Findings: findings,
		Stats:    Stats{DurationMS: msSince(start)},
	}, nil
}

func anyIgnored(ids []string, set map[string]bool) bool {
	for _, id := range ids {
		if set[strings.ToUpper(id)] {
			return true
		}
	}
	return false
}

func cveExpired(expires string) bool {
	if expires == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", expires)
	if err != nil {
		return false
	}
	return time.Now().After(t)
}

func firstFixedVersion(affected []struct {
	Ranges []struct {
		Events []struct {
			Introduced string `json:"introduced,omitempty"`
			Fixed      string `json:"fixed,omitempty"`
		} `json:"events"`
	} `json:"ranges"`
}) string {
	for _, a := range affected {
		for _, r := range a.Ranges {
			for _, e := range r.Events {
				if e.Fixed != "" {
					return e.Fixed
				}
			}
		}
	}
	return ""
}

func fixHint(name, cur, fixed string) string {
	if fixed == "" {
		return ""
	}
	return fmt.Sprintf("upgrade %s from %s → %s", name, cur, fixed)
}
