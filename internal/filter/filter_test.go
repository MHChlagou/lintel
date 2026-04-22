package filter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aegis-sec/aegis/internal/finding"
)

func TestAllowEntryMatches(t *testing.T) {
	e := AllowEntry{Rule: "aws-key", Path: "src/**/*.go", Checks: []string{"secrets"}}
	if !e.Matches("secrets", "aws-key", "src/internal/cfg.go") {
		t.Fatal("expected match")
	}
	if e.Matches("secrets", "other-rule", "src/internal/cfg.go") {
		t.Fatal("rule mismatch should not match")
	}
	if e.Matches("lint", "aws-key", "src/internal/cfg.go") {
		t.Fatal("check mismatch should not match")
	}
	if e.Matches("secrets", "aws-key", "vendor/x.go") {
		t.Fatal("path mismatch should not match")
	}
}

func TestAllowEntryExpired(t *testing.T) {
	e := AllowEntry{Expires: "2000-01-01"}
	if !e.Expired() {
		t.Fatal("2000-01-01 should be expired")
	}
	e2 := AllowEntry{Expires: "2999-12-31"}
	if e2.Expired() {
		t.Fatal("2999 should not be expired")
	}
}

func TestBaselineRoundtrip(t *testing.T) {
	root := t.TempDir()
	fs := []finding.Finding{
		{Check: "secrets", RuleID: "aws-key", File: "a.go", Snippet: "AKIA****XYZ9"},
		{Check: "malicious_code", RuleID: "r1", File: "b.go", Snippet: "exec.Command"},
	}
	if err := SaveBaseline(root, fs, "2026-04-21T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	// confirm file is created under .aegis
	if _, err := os.Stat(filepath.Join(root, ".aegis", "baseline.json")); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBaseline(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if !loaded.Contains(f) {
			t.Fatalf("baseline should contain %+v", f)
		}
	}
	unseen := finding.Finding{Check: "secrets", RuleID: "new", File: "c.go", Snippet: "different"}
	if loaded.Contains(unseen) {
		t.Fatal("baseline should not contain new finding")
	}
}

func TestInlineIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	body := `package main

// aegis:ignore-secret  reason="test fixture"
var k = "AKIAEXAMPLEEXAMPLE"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, missingReason, err := InlineIgnored(path, 4, "aegis:ignore-secret", "aws-key")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || missingReason {
		t.Fatalf("expected ignored with reason; got ok=%v missing=%v", ok, missingReason)
	}
}

func TestInlineIgnoredMissingReason(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	body := `package main

// aegis:ignore-secret
var k = "AKIAEXAMPLEEXAMPLE"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, missingReason, err := InlineIgnored(path, 4, "aegis:ignore-secret", "aws-key")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !missingReason {
		t.Fatalf("expected ignored but missing reason; got ok=%v missing=%v", ok, missingReason)
	}
}
