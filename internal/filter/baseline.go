package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/aegis-sec/aegis/internal/finding"
)

type Baseline struct {
	CreatedAt string            `json:"created_at"`
	Keys      map[string]bool   `json:"-"`
	Raw       []BaselineFinding `json:"findings"`
}

type BaselineFinding struct {
	Check      string `json:"check"`
	RuleID     string `json:"rule_id"`
	File       string `json:"file"`
	SnippetKey string `json:"snippet_key"`
}

func baselinePath(repoRoot string) string {
	return filepath.Join(repoRoot, ".aegis", "baseline.json")
}

func LoadBaseline(repoRoot string) (*Baseline, error) {
	raw, err := os.ReadFile(baselinePath(repoRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return &Baseline{Keys: map[string]bool{}}, nil
		}
		return nil, err
	}
	var b Baseline
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	b.Keys = map[string]bool{}
	for _, bf := range b.Raw {
		b.Keys[keyFor(bf.Check, bf.RuleID, bf.File, bf.SnippetKey)] = true
	}
	return &b, nil
}

func SaveBaseline(repoRoot string, findings []finding.Finding, now string) error {
	dir := filepath.Join(repoRoot, ".aegis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b := Baseline{CreatedAt: now}
	for _, f := range findings {
		b.Raw = append(b.Raw, BaselineFinding{
			Check: f.Check, RuleID: f.RuleID, File: f.File, SnippetKey: snippetHash(f.Snippet),
		})
	}
	raw, _ := json.MarshalIndent(b, "", "  ")
	return os.WriteFile(baselinePath(repoRoot), raw, 0o644)
}

// Contains reports whether this finding was already in the baseline. Line
// numbers are deliberately not part of the key — files get reformatted.
func (b *Baseline) Contains(f finding.Finding) bool {
	if b == nil || len(b.Keys) == 0 {
		return false
	}
	return b.Keys[keyFor(f.Check, f.RuleID, f.File, snippetHash(f.Snippet))]
}

func keyFor(check, rule, file, snippetKey string) string {
	return strings.Join([]string{check, rule, file, snippetKey}, "|")
}

func snippetHash(s string) string {
	if s == "" {
		return ""
	}
	// Normalize whitespace to avoid spurious mismatches from reformatting.
	normalized := strings.Join(strings.Fields(s), " ")
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:8])
}
