// Package finding defines the normalized result type every scanner collapses into.
package finding

import (
	"encoding/json"
	"sort"
	"strings"
)

type Severity string

const (
	SevCritical Severity = "CRITICAL"
	SevHigh     Severity = "HIGH"
	SevMedium   Severity = "MEDIUM"
	SevLow      Severity = "LOW"
	SevInfo     Severity = "INFO"
)

var sevRank = map[Severity]int{
	SevCritical: 5, SevHigh: 4, SevMedium: 3, SevLow: 2, SevInfo: 1,
}

func (s Severity) Rank() int { return sevRank[s] }

func ParseSeverity(raw string) Severity {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "CRITICAL":
		return SevCritical
	case "HIGH", "ERROR":
		return SevHigh
	case "MEDIUM", "MODERATE", "WARNING", "WARN":
		return SevMedium
	case "LOW", "NOTE":
		return SevLow
	default:
		return SevInfo
	}
}

// Finding is the single shape every downstream stage (filter, gate, reporter) sees.
type Finding struct {
	Check      string          `json:"check"`
	RuleID     string          `json:"rule_id"`
	Severity   Severity        `json:"severity"`
	File       string          `json:"file"`
	Line       int             `json:"line,omitempty"`
	Column     int             `json:"column,omitempty"`
	Message    string          `json:"message"`
	Snippet    string          `json:"snippet,omitempty"`
	FixSuggest string          `json:"fix_suggest,omitempty"`
	Blocking   bool            `json:"blocking"`
	Engine     string          `json:"engine"`
	EngineRaw  json.RawMessage `json:"engine_raw,omitempty"`
}

// Sort orders findings deterministically so output is stable across parallel runs.
func Sort(xs []Finding) {
	sort.SliceStable(xs, func(i, j int) bool {
		a, b := xs[i], xs[j]
		if a.Check != b.Check {
			return a.Check < b.Check
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.RuleID < b.RuleID
	})
}
