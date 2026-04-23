package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderFixHint_SingleScanner(t *testing.T) {
	var buf bytes.Buffer
	renderFixHint(&buf, []string{"gitleaks"})
	got := buf.String()
	if !strings.Contains(got, "lintel install gitleaks") {
		t.Errorf("expected single-scanner hint naming gitleaks, got: %q", got)
	}
	if strings.Contains(got, "--all") {
		t.Errorf("did not expect --all in single-scanner hint, got: %q", got)
	}
}

func TestRenderFixHint_MultipleScanners(t *testing.T) {
	var buf bytes.Buffer
	renderFixHint(&buf, []string{"gitleaks", "ruff", "biome"})
	got := buf.String()
	if !strings.Contains(got, "lintel install --all") {
		t.Errorf("expected --all hint for multiple failures, got: %q", got)
	}
	for _, n := range []string{"gitleaks", "ruff", "biome"} {
		if strings.Contains(got, "install "+n+"\n") {
			t.Errorf("did not expect per-scanner name %q in multi-hint, got: %q", n, got)
		}
	}
}

func TestRenderFixHint_GofmtIsSpecialCased(t *testing.T) {
	var buf bytes.Buffer
	renderFixHint(&buf, []string{"gofmt"})
	got := buf.String()
	if !strings.Contains(got, "Go toolchain") {
		t.Errorf("expected Go-toolchain note for gofmt-only failure, got: %q", got)
	}
	if strings.Contains(got, "lintel install") {
		t.Errorf("should not suggest `lintel install` for gofmt, got: %q", got)
	}
}

func TestRenderFixHint_GofmtPlusOneOther(t *testing.T) {
	var buf bytes.Buffer
	renderFixHint(&buf, []string{"gofmt", "gitleaks"})
	got := buf.String()
	if !strings.Contains(got, "lintel install gitleaks") {
		t.Errorf("expected single-scanner hint for gitleaks (gofmt is excluded), got: %q", got)
	}
	if !strings.Contains(got, "Go toolchain") {
		t.Errorf("expected gofmt caveat to still appear, got: %q", got)
	}
}
