// Package report turns findings into output bytes. Pretty is default; JSON is machine-first.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/aegis-sec/aegis/internal/finding"
	"github.com/aegis-sec/aegis/internal/version"
)

// ExitCodes mirror spec §11.5.
const (
	ExitOK            = 0
	ExitBlocking      = 1
	ExitConfig        = 2
	ExitBinaryResolve = 3
	ExitScannerCrash  = 4
	ExitInternal      = 5
	ExitInterrupted   = 130
)

type Summary struct {
	Version      int               `json:"version"`
	AegisVersion string            `json:"aegis_version"`
	Repo         string            `json:"repo"`
	Stacks       []string          `json:"stacks"`
	StartedAt    string            `json:"started_at"`
	DurationMS   int               `json:"duration_ms"`
	Hook         string            `json:"hook"`
	Summary      Counts            `json:"summary"`
	Findings     []finding.Finding `json:"findings"`
}

type Counts struct {
	Blocking int `json:"blocking"`
	Total    int `json:"total"`
}

func NewSummary(repo, hook string, stacks []string, start time.Time, findings []finding.Finding) Summary {
	s := Summary{
		Version:      1,
		AegisVersion: version.Version,
		Repo:         repo,
		Stacks:       stacks,
		StartedAt:    start.UTC().Format(time.RFC3339),
		DurationMS:   int(time.Since(start) / time.Millisecond),
		Hook:         hook,
		Findings:     findings,
	}
	for _, f := range findings {
		s.Summary.Total++
		if f.Blocking {
			s.Summary.Blocking++
		}
	}
	return s
}

// WriteJSON emits the stable machine schema.
func WriteJSON(w io.Writer, s Summary) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// WritePretty emits the human format shown in spec §11.1.
func WritePretty(w io.Writer, s Summary, color bool, stagedFiles int) {
	c := palette{enabled: color}
	fmt.Fprintln(w, c.dim(strings.Repeat("─", 58)))
	fmt.Fprintf(w, " Aegis %s — repo: %s  stacks: %s\n", version.Version, s.Repo, strings.Join(s.Stacks, ","))
	if s.Hook != "" {
		fmt.Fprintf(w, " hook: %s   staged: %d files\n", s.Hook, stagedFiles)
	}
	fmt.Fprintln(w, c.dim(strings.Repeat("─", 58)))

	// Summary line per check
	byCheck := groupByCheck(s.Findings)
	for _, check := range orderedChecks {
		fns, ok := byCheck[check]
		if !ok {
			fmt.Fprintf(w, "  %s %-16s all clean\n", c.green("✓"), check)
			continue
		}
		blocking := 0
		for _, f := range fns {
			if f.Blocking {
				blocking++
			}
		}
		icon := c.yellow("⚠")
		if blocking > 0 {
			icon = c.red("✖")
		}
		fmt.Fprintf(w, "  %s %-16s %d finding(s)    (%d blocking)\n", icon, check, len(fns), blocking)
	}
	fmt.Fprintln(w)

	for _, check := range orderedChecks {
		fns, ok := byCheck[check]
		if !ok {
			continue
		}
		fmt.Fprintf(w, "── %s %s\n", check, c.dim(strings.Repeat("─", 58-len(check)-4)))
		for _, f := range fns {
			label := "[WARN] "
			if f.Blocking {
				label = c.red("[BLOCK]") + " "
			}
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			fmt.Fprintf(w, "  %s%s\n", label, loc)
			if f.RuleID != "" {
				fmt.Fprintf(w, "    rule:  %s\n", f.RuleID)
			}
			if f.Message != "" {
				fmt.Fprintf(w, "    msg:   %s\n", f.Message)
			}
			if f.Snippet != "" {
				fmt.Fprintf(w, "    code:  %s\n", f.Snippet)
			}
			if f.FixSuggest != "" {
				fmt.Fprintf(w, "    fix:   %s\n", f.FixSuggest)
			}
		}
		fmt.Fprintln(w)
	}
	if s.Summary.Blocking > 0 {
		fmt.Fprintf(w, "%s commit blocked — %d blocking finding(s)\n", c.red("✖"), s.Summary.Blocking)
		fmt.Fprintln(w, "  bypass: AEGIS_SKIP=<checks> AEGIS_REASON=\"...\" git commit")
	} else {
		fmt.Fprintf(w, "%s ok — %d non-blocking finding(s)\n", c.green("✓"), s.Summary.Total)
	}
}

var orderedChecks = []string{"secrets", "malicious_code", "dependencies", "lint", "format"}

func groupByCheck(fs []finding.Finding) map[string][]finding.Finding {
	m := map[string][]finding.Finding{}
	for _, f := range fs {
		m[f.Check] = append(m[f.Check], f)
	}
	for k := range m {
		xs := m[k]
		sort.SliceStable(xs, func(i, j int) bool {
			if xs[i].File != xs[j].File {
				return xs[i].File < xs[j].File
			}
			return xs[i].Line < xs[j].Line
		})
	}
	return m
}

type palette struct{ enabled bool }

func (p palette) dim(s string) string    { return p.wrap(s, "\x1b[2m") }
func (p palette) red(s string) string    { return p.wrap(s, "\x1b[31m") }
func (p palette) green(s string) string  { return p.wrap(s, "\x1b[32m") }
func (p palette) yellow(s string) string { return p.wrap(s, "\x1b[33m") }
func (p palette) wrap(s, code string) string {
	if !p.enabled {
		return s
	}
	return code + s + "\x1b[0m"
}
