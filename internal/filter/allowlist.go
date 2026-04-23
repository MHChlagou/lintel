// Package filter applies allowlist, baseline, warn_paths, and inline-ignore layers.
package filter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

type Allowlist struct {
	Entries []AllowEntry `yaml:"entries"`
}

type AllowEntry struct {
	Path    string   `yaml:"path"`
	Rule    string   `yaml:"rule"`
	Checks  []string `yaml:"checks"`
	Reason  string   `yaml:"reason"`
	Expires string   `yaml:"expires"`
}

func LoadAllowlist(repoRoot string) (*Allowlist, error) {
	path := filepath.Join(repoRoot, ".lintel", "allowlist.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Allowlist{}, nil
		}
		return nil, err
	}
	var al Allowlist
	if err := yaml.Unmarshal(raw, &al); err != nil {
		return nil, fmt.Errorf("parse allowlist: %w", err)
	}
	return &al, nil
}

// Matches returns true if the entry covers the given finding.
func (e AllowEntry) Matches(check, rule, file string) bool {
	if e.Rule != "" && e.Rule != rule {
		return false
	}
	if len(e.Checks) > 0 {
		ok := false
		for _, c := range e.Checks {
			if c == check {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if e.Path != "" {
		if m, _ := doublestar.PathMatch(e.Path, file); !m {
			return false
		}
	}
	return true
}

func (e AllowEntry) Expired() bool {
	if e.Expires == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", e.Expires)
	if err != nil {
		return false
	}
	return time.Now().After(t)
}
