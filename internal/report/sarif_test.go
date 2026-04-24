package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MHChlagou/lintel/internal/finding"
)

func makeSummary(fs []finding.Finding) Summary {
	return NewSummary("myrepo", "pre-push", []string{"go"}, []string{"secrets"}, time.Now(), fs)
}

// decodeSARIF round-trips through the generic JSON shape so tests read the
// same structure GitHub's ingester would, instead of coupling to our internal
// struct field names.
func decodeSARIF(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, b)
	}
	return got
}

func TestWriteSARIF_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, makeSummary(nil)); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	if got["version"] != "2.1.0" {
		t.Errorf("version: got %v, want 2.1.0", got["version"])
	}
	runs := got["runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("runs: got %d, want 1", len(runs))
	}
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	if len(results) != 0 {
		t.Errorf("results: got %d, want 0", len(results))
	}
	driver := run["tool"].(map[string]any)["driver"].(map[string]any)
	if driver["name"] != "lintel" {
		t.Errorf("driver.name: got %v", driver["name"])
	}
	rules := driver["rules"].([]any)
	if len(rules) != 0 {
		t.Errorf("driver.rules: got %d, want 0", len(rules))
	}
}

func TestWriteSARIF_Findings(t *testing.T) {
	fs := []finding.Finding{
		{
			Check: "secrets", RuleID: "lintel.config-secret.pass",
			Severity: finding.SevHigh, File: "Dockerfile", Line: 17, Column: 5,
			Message: "Hardcoded credential in dockerfile config (key=pass)",
			Engine:  "lintel-config",
		},
		{
			Check: "malicious_code", RuleID: "dockerfile.security.missing-user.missing-user",
			Severity: finding.SevMedium, File: "Dockerfile", Line: 18,
			Message: "Missing USER directive",
			Engine:  "opengrep",
		},
		// Second finding sharing the same rule ID — should not duplicate in rules[].
		{
			Check: "secrets", RuleID: "lintel.config-secret.pass",
			Severity: finding.SevHigh, File: ".env", Line: 3,
			Message: "Hardcoded credential in dotenv config (key=pass)",
			Engine:  "lintel-config",
		},
	}
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, makeSummary(fs)); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	run := got["runs"].([]any)[0].(map[string]any)
	results := run["results"].([]any)
	if len(results) != 3 {
		t.Fatalf("results: got %d, want 3", len(results))
	}
	rules := run["tool"].(map[string]any)["driver"].(map[string]any)["rules"].([]any)
	if len(rules) != 2 {
		t.Fatalf("rules: got %d, want 2 (deduped)", len(rules))
	}

	// Rules are sorted by ID alphabetically; "dockerfile..." < "lintel..."
	first := rules[0].(map[string]any)
	if first["id"] != "dockerfile.security.missing-user.missing-user" {
		t.Errorf("rules[0].id: got %v", first["id"])
	}
	if first["name"] != "security.missing-user.missing-user" {
		t.Errorf("rules[0].name: got %v (want engine prefix stripped)", first["name"])
	}

	// Severity mapping: HIGH → error, MEDIUM → warning.
	levels := map[string]int{}
	for _, r := range results {
		levels[r.(map[string]any)["level"].(string)]++
	}
	if levels["error"] != 2 || levels["warning"] != 1 {
		t.Errorf("level counts: got %v, want error=2 warning=1", levels)
	}

	// Locations carry file/line; column set when nonzero.
	firstResult := results[0].(map[string]any)
	loc := firstResult["locations"].([]any)[0].(map[string]any)["physicalLocation"].(map[string]any)
	art := loc["artifactLocation"].(map[string]any)
	if art["uri"] == "" {
		t.Errorf("artifactLocation.uri empty")
	}
	if loc["region"] == nil {
		t.Errorf("region missing on finding with line>0")
	}
}

func TestWriteSARIF_LevelMap(t *testing.T) {
	cases := []struct {
		sev   finding.Severity
		level string
	}{
		{finding.SevCritical, "error"},
		{finding.SevHigh, "error"},
		{finding.SevMedium, "warning"},
		{finding.SevLow, "note"},
		{finding.SevInfo, "note"},
	}
	for _, c := range cases {
		if got := sarifLevel(c.sev); got != c.level {
			t.Errorf("sarifLevel(%q) = %q, want %q", c.sev, got, c.level)
		}
	}
}

func TestWriteSARIF_URINormalization(t *testing.T) {
	// Windows-style paths must come through as forward slashes.
	got := normalizeURI(`k8s\deployment.yaml`)
	if got != "k8s/deployment.yaml" {
		t.Errorf("normalizeURI: got %q, want k8s/deployment.yaml", got)
	}
}

func TestWriteSARIF_SchemaField(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, makeSummary(nil)); err != nil {
		t.Fatal(err)
	}
	// Confirm the schema URI survives JSON encoding intact — some encoders
	// mangle the leading $.
	if !strings.Contains(buf.String(), `"$schema"`) {
		t.Error("output missing $schema key")
	}
}
