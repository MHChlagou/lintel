package report

import (
	"encoding/json"
	"io"
	"sort"
	"strings"

	"github.com/MHChlagou/lintel/internal/finding"
)

// WriteSARIF emits SARIF 2.1.0 suitable for GitHub Advanced Security code
// scanning uploads. The schema is the minimum subset GitHub's ingester
// requires: one run per invocation, a rule catalog built from the findings'
// distinct RuleIDs, and one result per finding pointing at file / line /
// column in the working tree.
//
// Severity → SARIF level mapping: CRITICAL/HIGH → "error",
// MEDIUM/WARNING → "warning", LOW/INFO → "note". GitHub's PR-annotation
// threshold defaults to "error", so blocking findings always annotate.
func WriteSARIF(w io.Writer, s Summary) error {
	rules, indexByID := buildRules(s.Findings)
	results := make([]sarifResult, 0, len(s.Findings))
	for _, f := range s.Findings {
		loc := sarifLocation{
			PhysicalLocation: sarifPhysLoc{
				ArtifactLocation: sarifArtLoc{URI: normalizeURI(f.File)},
			},
		}
		if f.Line > 0 {
			loc.PhysicalLocation.Region = &sarifRegion{
				StartLine:   f.Line,
				StartColumn: f.Column,
			}
		}
		msg := f.Message
		if msg == "" {
			msg = f.RuleID
		}
		results = append(results, sarifResult{
			RuleID:    f.RuleID,
			RuleIndex: indexByID[f.RuleID],
			Level:     sarifLevel(f.Severity),
			Message:   sarifText{Text: msg},
			Locations: []sarifLocation{loc},
		})
	}

	doc := sarifDoc{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "lintel",
				Version:        s.LintelVersion,
				InformationURI: "https://github.com/MHChlagou/lintel",
				Rules:          rules,
			}},
			Results: results,
		}},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// buildRules gathers distinct rules referenced by the findings, preserving
// a stable order so the SARIF output is reproducible across runs. Each rule
// borrows the message from the first finding that cites it as its short
// description — imperfect when multiple findings share a rule with varied
// messages, but better than an empty description which some SARIF viewers
// render as "Untitled rule".
func buildRules(fs []finding.Finding) ([]sarifRule, map[string]int) {
	indexByID := make(map[string]int)
	rules := make([]sarifRule, 0)
	for _, f := range fs {
		if _, seen := indexByID[f.RuleID]; seen {
			continue
		}
		indexByID[f.RuleID] = len(rules)
		rules = append(rules, sarifRule{
			ID:   f.RuleID,
			Name: ruleDisplayName(f.RuleID),
			ShortDescription: sarifText{
				Text: firstNonEmptyString(f.Message, f.RuleID),
			},
			DefaultConfiguration: &sarifRuleConfig{Level: sarifLevel(f.Severity)},
		})
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	// Rebuild the index to match the sorted slice.
	for i, r := range rules {
		indexByID[r.ID] = i
	}
	return rules, indexByID
}

// ruleDisplayName strips a leading `<engine>.` prefix so SARIF viewers render
// the short form (`config-secret.pass`) rather than the fully-qualified ID
// (`lintel.config-secret.pass`). The fully-qualified ID stays on `rule.id`.
func ruleDisplayName(id string) string {
	if i := strings.Index(id, "."); i >= 0 {
		return id[i+1:]
	}
	return id
}

// sarifLevel maps the lintel severity scale onto SARIF's four-valued enum.
// GitHub's code-scanning defaults trigger a PR annotation at "error" and
// above; "note" does not annotate by default.
func sarifLevel(sev finding.Severity) string {
	switch sev {
	case finding.SevCritical, finding.SevHigh:
		return "error"
	case finding.SevMedium:
		return "warning"
	case finding.SevLow, finding.SevInfo:
		return "note"
	default:
		return "note"
	}
}

// normalizeURI converts OS-specific path separators to forward slashes so
// artifactLocation.uri matches SARIF's URI convention.
func normalizeURI(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func firstNonEmptyString(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

// ---- SARIF 2.1.0 subset ----

type sarifDoc struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string           `json:"id"`
	Name                 string           `json:"name,omitempty"`
	ShortDescription     sarifText        `json:"shortDescription"`
	DefaultConfiguration *sarifRuleConfig `json:"defaultConfiguration,omitempty"`
}

type sarifRuleConfig struct {
	Level string `json:"level"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	RuleIndex int             `json:"ruleIndex"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysLoc `json:"physicalLocation"`
}

type sarifPhysLoc struct {
	ArtifactLocation sarifArtLoc  `json:"artifactLocation"`
	Region           *sarifRegion `json:"region,omitempty"`
}

type sarifArtLoc struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
}
