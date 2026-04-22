package checker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/aegis-sec/aegis/internal/detect"
	"github.com/aegis-sec/aegis/internal/finding"
)

type Format struct{}

func (Format) Name() string                           { return "format" }
func (Format) Applicable(*detect.ProjectContext) bool { return true }
func (Format) RequiredBinaries() []string             { return nil }

func (f Format) Run(ctx context.Context, in CheckInput) (CheckOutput, error) {
	cfg := in.Spec.Checks.Format
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
		res, err := runFormatter(ctx, in, tool, files, cfg.Mode)
		if err != nil {
			return CheckOutput{}, err
		}
		all = append(all, res...)
	}
	return CheckOutput{Findings: all, Stats: Stats{DurationMS: msSince(start), FilesScanned: len(in.StagedFiles)}}, nil
}

// runFormatter shells out to a per-language formatter. In "check" mode we
// expect a non-zero exit to mean unformatted files; in "fix" mode we write
// changes back and re-stage via git add.
func runFormatter(ctx context.Context, in CheckInput, tool string, files []string, mode interface{}) ([]finding.Finding, error) {
	rb, err := in.Resolver.Resolve(tool)
	if err != nil {
		// Don't fail the commit just because a formatter is missing; surface as INFO.
		return []finding.Finding{{
			Check: "format", RuleID: "aegis.tool-missing", Severity: finding.SevInfo,
			Message: err.Error(), Engine: tool,
		}}, nil
	}
	isFix := fmt.Sprintf("%v", mode) == "fix"
	var args []string
	switch tool {
	case "gofmt":
		if isFix {
			args = append([]string{"-w"}, files...)
		} else {
			args = append([]string{"-l"}, files...)
		}
	case "biome":
		args = []string{"format"}
		if isFix {
			args = append(args, "--write")
		}
		args = append(args, files...)
	case "ruff":
		args = []string{"format"}
		if !isFix {
			args = append(args, "--check")
		}
		args = append(args, files...)
	case "rustfmt":
		if !isFix {
			args = append([]string{"--check"}, files...)
		} else {
			args = files
		}
	case "shfmt":
		args = append([]string{"-d"}, files...)
	default:
		args = files
	}
	cmd := exec.CommandContext(ctx, rb.Path, args...)
	cmd.Dir = in.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	// In fix mode, success. Re-stage affected files so the commit sees them.
	if isFix {
		if err == nil {
			restage(ctx, in.RepoRoot, files)
		}
		return nil, nil
	}
	// Check mode: any non-empty stdout or non-zero exit = at least one unformatted file.
	if err != nil || stdout.Len() > 0 {
		msg := "files need formatting"
		if s := stdout.String(); s != "" {
			msg = "files need formatting: " + truncate(s, 200)
		}
		return []finding.Finding{{
			Check: "format", RuleID: "format.unformatted", Severity: finding.SevLow,
			Engine: tool, Message: msg,
		}}, nil
	}
	return nil, nil
}

func restage(ctx context.Context, dir string, files []string) {
	if len(files) == 0 {
		return
	}
	args := append([]string{"add", "--"}, files...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	_ = cmd.Run()
}
