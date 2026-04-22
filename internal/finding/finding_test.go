package finding

import "testing"

func TestParseSeverity(t *testing.T) {
	cases := map[string]Severity{
		"CRITICAL": SevCritical,
		"critical": SevCritical,
		"ERROR":    SevHigh,
		"HIGH":     SevHigh,
		"WARNING":  SevMedium,
		"moderate": SevMedium,
		"LOW":      SevLow,
		"info":     SevInfo,
		"banana":   SevInfo,
		"":         SevInfo,
	}
	for in, want := range cases {
		if got := ParseSeverity(in); got != want {
			t.Errorf("ParseSeverity(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestSortStable(t *testing.T) {
	fs := []Finding{
		{Check: "lint", File: "b.go", Line: 2, RuleID: "x"},
		{Check: "secrets", File: "a.go", Line: 5, RuleID: "y"},
		{Check: "lint", File: "b.go", Line: 1, RuleID: "x"},
		{Check: "lint", File: "a.go", Line: 99, RuleID: "x"},
	}
	Sort(fs)
	if fs[0].Check != "lint" || fs[0].File != "a.go" {
		t.Fatalf("unexpected first entry: %+v", fs[0])
	}
	if fs[3].Check != "secrets" {
		t.Fatalf("unexpected last entry: %+v", fs[3])
	}
}

func TestSeverityRank(t *testing.T) {
	if SevCritical.Rank() <= SevHigh.Rank() {
		t.Fatal("CRITICAL should outrank HIGH")
	}
	if SevInfo.Rank() >= SevLow.Rank() {
		t.Fatal("INFO should rank below LOW")
	}
}
