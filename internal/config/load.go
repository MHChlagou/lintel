package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultRelPath = ".aegis/aegis.yaml"

// Load reads and parses the spec file. If path is empty, it resolves to
// $AEGIS_CONFIG then <repoRoot>/.aegis/aegis.yaml.
func Load(repoRoot, path string) (*Spec, error) {
	if path == "" {
		if env := os.Getenv("AEGIS_CONFIG"); env != "" {
			path = env
		} else {
			path = filepath.Join(repoRoot, DefaultRelPath)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec %s: %w", path, err)
	}
	var s Spec
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse spec %s: %w", path, err)
	}
	ApplyDefaults(&s)
	if err := Validate(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ApplyDefaults fills in blanks to keep downstream code from checking empties.
func ApplyDefaults(s *Spec) {
	if s.Version == 0 {
		s.Version = 1
	}
	if s.Output.Format == "" {
		s.Output.Format = "pretty"
	}
	if s.Output.GroupBy == "" {
		s.Output.GroupBy = "check"
	}
	if s.Output.Color == "" {
		s.Output.Color = "auto"
	}
	if s.Output.Verbosity == "" {
		s.Output.Verbosity = "normal"
	}
	if s.Override.EnvVar == "" {
		s.Override.EnvVar = "AEGIS_SKIP"
	}
	if s.Override.LogFile == "" {
		s.Override.LogFile = ".aegis/overrides.log"
	}
	if s.Performance.CheckTimeoutSeconds == 0 {
		s.Performance.CheckTimeoutSeconds = 120
	}
	if s.Performance.TotalTimeoutSeconds == 0 {
		s.Performance.TotalTimeoutSeconds = 300
	}
	if s.Performance.Parallel == nil {
		s.Performance.Parallel = "auto"
	}
	if s.Performance.Cache.Path == "" {
		s.Performance.Cache.Path = "~/.aegis/cache"
	}
	if s.Checks.Secrets.Engine == "" {
		s.Checks.Secrets.Engine = "gitleaks"
	}
	if s.Checks.Secrets.Mode == "" {
		s.Checks.Secrets.Mode = ModeBlock
	}
	if s.Checks.Secrets.InlineIgnore == "" {
		s.Checks.Secrets.InlineIgnore = "aegis:ignore-secret"
	}
	if s.Checks.MaliciousCode.Engine == "" {
		s.Checks.MaliciousCode.Engine = "opengrep"
	}
	if s.Checks.MaliciousCode.Mode == "" {
		s.Checks.MaliciousCode.Mode = ModeBlock
	}
	if s.Checks.MaliciousCode.SeverityThreshold == "" {
		s.Checks.MaliciousCode.SeverityThreshold = "ERROR"
	}
	if s.Checks.MaliciousCode.TimeoutSeconds == 0 {
		s.Checks.MaliciousCode.TimeoutSeconds = 60
	}
	if s.Checks.Dependencies.Engine == "" {
		s.Checks.Dependencies.Engine = "osv-scanner"
	}
	if s.Checks.Dependencies.Mode == "" {
		s.Checks.Dependencies.Mode = ModeBlock
	}
	if len(s.Checks.Dependencies.BlockSeverity) == 0 {
		s.Checks.Dependencies.BlockSeverity = []string{"CRITICAL", "HIGH"}
	}
	if s.Checks.Lint.Mode == "" {
		s.Checks.Lint.Mode = ModeWarn
	}
	if s.Checks.Format.Mode == "" {
		s.Checks.Format.Mode = ModeFix
	}
}

// Validate catches spec errors early so we can fail with a pointer to the fix.
func Validate(s *Spec) error {
	if s.Version != 1 {
		return fmt.Errorf("spec version %d unsupported (expected 1)", s.Version)
	}
	if err := validateMode("checks.secrets.mode", s.Checks.Secrets.Mode, ModeBlock, ModeWarn, ModeOff); err != nil {
		return err
	}
	if err := validateMode("checks.malicious_code.mode", s.Checks.MaliciousCode.Mode, ModeBlock, ModeWarn, ModeOff); err != nil {
		return err
	}
	if err := validateMode("checks.dependencies.mode", s.Checks.Dependencies.Mode, ModeBlock, ModeWarn, ModeOff); err != nil {
		return err
	}
	if err := validateMode("checks.lint.mode", s.Checks.Lint.Mode, ModeBlock, ModeWarn, ModeOff); err != nil {
		return err
	}
	if err := validateMode("checks.format.mode", s.Checks.Format.Mode, ModeFix, ModeCheck, ModeOff); err != nil {
		return err
	}
	switch s.Output.Format {
	case "pretty", "json", "sarif", "junit":
	default:
		return fmt.Errorf("output.format %q: must be pretty|json|sarif|junit", s.Output.Format)
	}
	if err := validateBinaryPaths(s); err != nil {
		return err
	}
	return nil
}

func validateMode(field string, got Mode, allowed ...Mode) error {
	if got == "" {
		return nil
	}
	for _, a := range allowed {
		if got == a {
			return nil
		}
	}
	names := make([]string, len(allowed))
	for i, a := range allowed {
		names[i] = string(a)
	}
	return fmt.Errorf("%s: unknown value %q (allowed: %s)", field, got, strings.Join(names, ", "))
}

// validateBinaryPaths rejects relative paths that would point at a vendored
// binary inside the repo — a defense against a malicious PR slipping one in.
func validateBinaryPaths(s *Spec) error {
	for name, b := range s.Binaries {
		if b.Path == "" {
			continue
		}
		if strings.HasPrefix(b.Path, "~") || filepath.IsAbs(b.Path) {
			continue
		}
		return fmt.Errorf("binaries.%s.path %q: relative paths are rejected; use an absolute path or one starting with ~", name, b.Path)
	}
	return nil
}
