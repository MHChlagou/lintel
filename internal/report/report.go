// Package report turns findings into output bytes. Pretty is default; JSON is machine-first.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/MHChlagou/lintel/internal/finding"
	"github.com/MHChlagou/lintel/internal/version"
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
	Version       int      `json:"version"`
	LintelVersion string   `json:"lintel_version"`
	Repo          string   `json:"repo"`
	Stacks        []string `json:"stacks"`
	StartedAt     string   `json:"started_at"`
	DurationMS    int      `json:"duration_ms"`
	Hook          string   `json:"hook"`
	// ChecksRun lists the checks that were actually executed in this run,
	// in spec order. Checks that were filtered out (via --check, --hook, or
	// LINTEL_SKIP) or disabled in config are intentionally absent.
	ChecksRun []string          `json:"checks_run"`
	Summary   Counts            `json:"summary"`
	Findings  []finding.Finding `json:"findings"`
}

type Counts struct {
	Blocking int `json:"blocking"`
	Total    int `json:"total"`
}

func NewSummary(repo, hook string, stacks, checksRun []string, start time.Time, findings []finding.Finding) Summary {
	s := Summary{
		Version:       1,
		LintelVersion: version.Version,
		Repo:          repo,
		Stacks:        stacks,
		StartedAt:     start.UTC().Format(time.RFC3339),
		DurationMS:    int(time.Since(start) / time.Millisecond),
		Hook:          hook,
		ChecksRun:     orderedIntersect(checksRun),
		Findings:      findings,
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

// Writer errors from Fprint* on stdout/stderr are non-actionable, so these
// helpers swallow them. Keeps the call sites readable.
func fpf(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
func fpln(w io.Writer, args ...any)               { _, _ = fmt.Fprintln(w, args...) }

// WritePretty emits the human format shown in spec §11.1.
func WritePretty(w io.Writer, s Summary, color bool, stagedFiles int) {
	c := palette{enabled: color}
	fpln(w, c.dim(strings.Repeat("─", 58)))
	fpf(w, " Lintel %s - repo: %s  stacks: %s\n", version.Version, s.Repo, strings.Join(s.Stacks, ","))
	if s.Hook != "" {
		fpf(w, " hook: %s   staged: %d files\n", s.Hook, stagedFiles)
	}
	fpln(w, c.dim(strings.Repeat("─", 58)))

	// Summary line per check - only for checks that actually ran.
	byCheck := groupByCheck(s.Findings)
	for _, check := range s.ChecksRun {
		fns, ok := byCheck[check]
		if !ok {
			fpf(w, "  %s %-16s all clean\n", c.green("✓"), check)
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
		fpf(w, "  %s %-16s %d finding(s)    (%d blocking)\n", icon, check, len(fns), blocking)
	}
	fpln(w)

	for _, check := range s.ChecksRun {
		fns, ok := byCheck[check]
		if !ok {
			continue
		}
		fpf(w, "── %s %s\n", check, c.dim(strings.Repeat("─", 58-len(check)-4)))
		for _, f := range fns {
			label := "[WARN] "
			if f.Blocking {
				label = c.red("[BLOCK]") + " "
			}
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			fpf(w, "  %s%s\n", label, loc)
			if f.RuleID != "" {
				fpf(w, "    rule:  %s\n", f.RuleID)
			}
			if f.Message != "" {
				fpf(w, "    msg:   %s\n", f.Message)
			}
			if f.Snippet != "" {
				fpf(w, "    code:  %s\n", f.Snippet)
			}
			if f.FixSuggest != "" {
				fpf(w, "    fix:   %s\n", f.FixSuggest)
			}
		}
		fpln(w)
	}
	if s.Summary.Blocking > 0 {
		fpf(w, "%s commit blocked - %d blocking finding(s)\n", c.red("✖"), s.Summary.Blocking)
		fpln(w, "  bypass: LINTEL_SKIP=<checks> LINTEL_REASON=\"...\" git commit")
	} else {
		fpf(w, "%s ok - %d non-blocking finding(s)\n", c.green("✓"), s.Summary.Total)
	}
}

var orderedChecks = []string{"secrets", "malicious_code", "dependencies", "lint", "format"}

// orderedIntersect returns checksRun in the canonical spec order and drops any
// unknown names. Keeps the output deterministic regardless of the caller's
// input order.
func orderedIntersect(checksRun []string) []string {
	want := make(map[string]bool, len(checksRun))
	for _, c := range checksRun {
		want[c] = true
	}
	out := make([]string, 0, len(checksRun))
	for _, c := range orderedChecks {
		if want[c] {
			out = append(out, c)
		}
	}
	return out
}

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
