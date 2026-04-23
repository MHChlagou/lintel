package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	sub := filepath.Join(dir, ".lintel")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(sub, "lintel.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadDefaults(t *testing.T) {
	root := writeTemp(t, "version: 1\n")
	s, err := Load(root, "")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s.Output.Format != "pretty" {
		t.Errorf("default format = %q, want pretty", s.Output.Format)
	}
	if s.Performance.CheckTimeoutSeconds != 120 {
		t.Errorf("default check timeout = %d, want 120", s.Performance.CheckTimeoutSeconds)
	}
	if s.Checks.Secrets.Engine != "gitleaks" {
		t.Errorf("default secrets engine = %q, want gitleaks", s.Checks.Secrets.Engine)
	}
}

func TestValidateRejectsRelativeBinaryPath(t *testing.T) {
	body := `version: 1
binaries:
  gitleaks:
    command: gitleaks
    path: "./vendored/gitleaks"
    version: "8.28.0"
`
	root := writeTemp(t, body)
	_, err := Load(root, "")
	if err == nil {
		t.Fatal("expected error for relative binary path")
	}
	if !strings.Contains(err.Error(), "relative paths are rejected") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestValidateRejectsUnknownMode(t *testing.T) {
	body := `version: 1
checks:
  secrets:
    mode: warning
`
	root := writeTemp(t, body)
	_, err := Load(root, "")
	if err == nil {
		t.Fatal("expected error for mode 'warning'")
	}
}

func TestValidateRejectsBadSchemaVersion(t *testing.T) {
	body := `version: 99`
	root := writeTemp(t, body)
	_, err := Load(root, "")
	if err == nil {
		t.Fatal("expected error for schema version 99")
	}
}
