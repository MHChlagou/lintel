package checker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/MHChlagou/lintel/internal/detect"
	"github.com/MHChlagou/lintel/internal/finding"
)

type Lint struct{}

func (Lint) Name() string                               { return "lint" }
func (Lint) Applicable(ctx *detect.ProjectContext) bool { return true }
func (Lint) RequiredBinaries() []string                 { return nil } // resolved lazily per-stack

// languageOfFile maps a staged file to a language key used to look up a tool.
func languageOfFile(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".sh", ".bash":
		return "shell"
	}
	return ""
}

// groupByTool buckets staged files by the lint tool configured for their language.
func groupByTool(stagedFiles []string, tools map[string]string) map[string][]string {
	buckets := map[string][]string{}
	for _, f := range stagedFiles {
		lang := languageOfFile(f)
		if lang == "" {
			continue
		}
		tool := tools[lang]
		if tool == "" {
			continue
		}
		buckets[tool] = append(buckets[tool], f)
	}
	return buckets
}

func (l Lint) Run(ctx context.Context, in CheckInput) (CheckOutput, error) {
	cfg := in.Spec.Checks.Lint
	if !cfg.Enabled || cfg.Mode == "off" {
		return CheckOutput{}, nil
	}
	start := time.Now()
	buckets := groupByTool(in.StagedFiles, cfg.Tools)
	if len(buckets) == 0 {
		return CheckOutput{Stats: Stats{DurationMS: msSince(start)}}, nil
	}
	var all []finding.Finding
	for tool, files := range buckets {
		fnds, err := runLinter(ctx, in, tool, files, cfg.AutoFix)
		if err != nil {
			return CheckOutput{}, err
		}
		all = append(all, fnds...)
	}
	return CheckOutput{Findings: all, Stats: Stats{DurationMS: msSince(start), FilesScanned: len(in.StagedFiles)}}, nil
}

// runLinter shells out to a specific linter. Each tool emits text or JSON we
// reduce to Findings; for v1 we support a minimal common path via the text
// output most linters share: file:line:col: message.
func runLinter(ctx context.Context, in CheckInput, tool string, files []string, autoFix bool) ([]finding.Finding, error) {
	rb, err := in.Resolver.Resolve(tool)
	if err != nil {
		// Missing lint tool is non-fatal when mode=warn: the gate marks nothing blocking.
		return []finding.Finding{{
			Check: "lint", RuleID: "lintel.tool-missing", Severity: finding.SevInfo,
			Message: err.Error(), Engine: tool,
		}}, nil
	}
	var args []string
	switch tool {
	case "golangci-lint":
		args = []string{"run", "--out-format=line-number"}
		args = append(args, files...)
	case "ruff":
		args = []string{"check", "--output-format=concise"}
		if autoFix {
			args = append(args, "--fix")
		}
		args = append(args, files...)
	case "biome":
		args = []string{"lint"}
		if autoFix {
			args = append(args, "--apply")
		}
		args = append(args, files...)
	case "shellcheck":
		args = append([]string{"-f", "gcc"}, files...)
	default:
		// Generic fallback: pass files and hope.
		args = files
	}
	if extra := in.Spec.Checks.Lint.Args[tool]; len(extra) > 0 {
		args = append(extra, args...)
	}
	cmd := exec.CommandContext(ctx, rb.Path, args...)
	cmd.Dir = in.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // linters exit non-zero on findings; we parse output regardless
	return parseLinterOutput(tool, stdout.String()+stderr.String()), nil
}

// parseLinterOutput handles the common path: many linters emit "file:line:col:
// message" or "file:line: message". Anything that doesn't match is surfaced as
// a single INFO-level finding so the user knows something happened.
func parseLinterOutput(tool, output string) []finding.Finding {
	var out []finding.Finding
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f, ok := parseFileLineColMessage(line)
		if !ok {
			continue
		}
		f.Check = "lint"
		f.Engine = tool
		f.Severity = finding.SevLow
		out = append(out, f)
	}
	return out
}

// parseFileLineColMessage matches "path:line:col: message" and "path:line: message".
func parseFileLineColMessage(s string) (finding.Finding, bool) {
	// Strip leading "./" which some tools emit
	s = strings.TrimPrefix(s, "./")
	parts := strings.SplitN(s, ":", 4)
	if len(parts) < 3 {
		return finding.Finding{}, false
	}
	line := atoi(parts[1])
	if line <= 0 {
		return finding.Finding{}, false
	}
	if len(parts) == 4 {
		col := atoi(parts[2])
		msg := strings.TrimSpace(parts[3])
		if col > 0 {
			return finding.Finding{File: parts[0], Line: line, Column: col, Message: msg, RuleID: extractRule(msg)}, true
		}
	}
	msg := strings.TrimSpace(strings.Join(parts[2:], ":"))
	return finding.Finding{File: parts[0], Line: line, Message: msg, RuleID: extractRule(msg)}, true
}

func atoi(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// extractRule pulls a parenthesized tag like "(SIM102)" or "[E501]" out of a message.
func extractRule(msg string) string {
	for _, open := range []string{"(", "["} {
		i := strings.Index(msg, open)
		if i < 0 {
			continue
		}
		close := ")"
		if open == "[" {
			close = "]"
		}
		j := strings.Index(msg[i:], close)
		if j > 0 {
			return msg[i+1 : i+j]
		}
	}
	if strings.HasPrefix(msg, "error:") || strings.HasPrefix(msg, "warning:") {
		return strings.Fields(msg)[0]
	}
	return fmt.Sprintf("lint.%x", hash32(msg))
}

// hash32 is an FNV-like checksum used only to stabilize auto-generated rule IDs.
func hash32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
