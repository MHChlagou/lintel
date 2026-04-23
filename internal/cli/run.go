package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MHChlagou/lintel/internal/checker"
	"github.com/MHChlagou/lintel/internal/config"
	"github.com/MHChlagou/lintel/internal/detect"
	"github.com/MHChlagou/lintel/internal/filter"
	"github.com/MHChlagou/lintel/internal/finding"
	"github.com/MHChlagou/lintel/internal/gate"
	"github.com/MHChlagou/lintel/internal/report"
	"github.com/MHChlagou/lintel/internal/resolve"
	"github.com/MHChlagou/lintel/internal/runner"
)

func cmdRun() *cobra.Command {
	var hookName string
	var single string
	c := &cobra.Command{
		Use:   "run",
		Short: "Run all enabled checks (honors --hook and --check)",
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			root := resolveRepoRoot()
			spec, err := config.Load(root, flags.configPath)
			if err != nil {
				fpln(cmd.ErrOrStderr(), "✖ spec error:", err)
				os.Exit(report.ExitConfig)
			}
			if flags.output != "" {
				spec.Output.Format = flags.output
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			staged, _ := detect.StagedFiles(ctx, root)
			projType, _ := projectTypeFromSpec(spec.Project.Type)
			proj, err := detect.Detect(root, projType, spec.Scope.ExcludePaths, staged)
			if err != nil {
				return err
			}

			checks := checksForHook(spec, hookName, single)
			// Drop disabled checks.
			enabled := make([]string, 0, len(checks))
			for _, c := range checks {
				if isCheckEnabled(spec, c) {
					enabled = append(enabled, c)
				}
			}
			checks = enabled
			// Env-var skip list
			skips := parseSkips(os.Getenv(spec.Override.EnvVar))
			checks = applySkips(checks, skips, spec, cmd.ErrOrStderr())

			// Resolve required binaries up front so we fail fast on missing/mismatched.
			res := newResolverWithPinFallback(root, spec.Binaries, spec.StrictVers)
			if err := preflightBinaries(checks, res, spec); err != nil {
				fpln(cmd.ErrOrStderr(), "✖", err)
				os.Exit(report.ExitBinaryResolve)
			}

			reg := checker.Registry()
			input := func(name string) checker.CheckInput {
				fullTree := hookName == "" || contains(spec.Scope.FullScanFor, name) || name == "dependencies"
				return checker.CheckInput{
					RepoRoot:    root,
					StagedFiles: staged,
					FullTree:    fullTree,
					Config:      json.RawMessage("{}"),
					Spec:        spec,
					Project:     proj,
					Resolver:    res,
					Hook:        hookName,
				}
			}
			failFast := hookName != "" && spec.Hooks[hookName].FailFast
			results := runner.Run(ctx, checks, input, reg, runner.Options{Spec: spec, FailFast: failFast})

			// Collect, filter, gate.
			var allFindings []finding.Finding
			var runErr error
			for _, r := range results {
				if r.Err != nil && runErr == nil {
					runErr = fmt.Errorf("%s: %w", r.Name, r.Err)
				}
				allFindings = append(allFindings, r.Output.Findings...)
			}
			al, _ := filter.LoadAllowlist(root)
			base, _ := filter.LoadBaseline(root)
			gated := gate.Apply(spec, al, base, root, allFindings)
			finding.Sort(gated)

			summary := report.NewSummary(projectName(root), hookName, proj.Stacks, checks, start, gated)

			switch spec.Output.Format {
			case "json":
				_ = report.WriteJSON(os.Stdout, summary)
			default:
				color := shouldUseColor(spec.Output.Color, flags.noColor)
				report.WritePretty(os.Stdout, summary, color, len(staged))
			}
			if runErr != nil {
				fpln(cmd.ErrOrStderr(), "⚠ scanner error:", runErr)
				os.Exit(report.ExitScannerCrash)
			}
			if summary.Summary.Blocking > 0 {
				os.Exit(report.ExitBlocking)
			}
			return nil
		},
	}
	c.Flags().StringVar(&hookName, "hook", "", "hook context: pre-commit|pre-push|commit-msg")
	c.Flags().StringVar(&single, "check", "", "run a single check: secrets|malicious_code|dependencies|lint|format")
	return c
}

func projectTypeFromSpec(t any) ([]string, error) {
	switch v := t.(type) {
	case string:
		if v == "" || v == "auto" {
			return nil, nil
		}
		return []string{v}, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			out = append(out, fmt.Sprint(x))
		}
		return out, nil
	case []string:
		return v, nil
	}
	return nil, nil
}

func checksForHook(spec *config.Spec, hook, single string) []string {
	if single != "" {
		return []string{single}
	}
	if hook != "" {
		if h, ok := spec.Hooks[hook]; ok {
			return h.Checks
		}
		return nil
	}
	return []string{"secrets", "malicious_code", "dependencies", "lint", "format"}
}

func parseSkips(env string) map[string]bool {
	m := map[string]bool{}
	for _, p := range strings.Split(env, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "all" {
			m["*"] = true
			continue
		}
		m[p] = true
	}
	return m
}

func applySkips(checks []string, skips map[string]bool, spec *config.Spec, stderr interface{ Write([]byte) (int, error) }) []string {
	if len(skips) == 0 {
		return checks
	}
	reason := os.Getenv("LINTEL_REASON")
	if spec.Override.RequireReason && strings.TrimSpace(reason) == "" {
		fpln(stderr, "✖ LINTEL_SKIP set but LINTEL_REASON is empty (override.require_reason=true)")
		os.Exit(report.ExitConfig)
	}
	out := make([]string, 0, len(checks))
	var logged []string
	for _, c := range checks {
		if skips["*"] || skips[c] {
			if c == "secrets" && spec.Override.ProtectSecrets {
				fpln(stderr, "! refused to skip secrets (override.protect_secrets=true)")
				out = append(out, c)
				continue
			}
			logged = append(logged, c)
			continue
		}
		out = append(out, c)
	}
	if len(logged) > 0 {
		writeOverrideLog(spec.Override.LogFile, reason, logged)
	}
	return out
}

func writeOverrideLog(path, reason string, checks []string) {
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	user := os.Getenv("USER")
	ts := time.Now().UTC().Format(time.RFC3339)
	head, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	fpf(f, "%s\tuser=%s\tcommit=%s\tskipped=%s\treason=%q\n",
		ts, user, strings.TrimSpace(string(head)), strings.Join(checks, ","), reason)
}

func preflightBinaries(checks []string, r *resolve.Resolver, spec *config.Spec) error {
	reg := checker.Registry()
	seen := map[string]bool{}
	for _, name := range checks {
		if !isCheckEnabled(spec, name) {
			continue
		}
		c, ok := reg[name]
		if !ok {
			continue
		}
		for _, b := range c.RequiredBinaries() {
			if seen[b] {
				continue
			}
			seen[b] = true
			if _, err := r.Resolve(b); err != nil {
				return err
			}
		}
	}
	return nil
}

func isCheckEnabled(spec *config.Spec, name string) bool {
	switch name {
	case "secrets":
		return spec.Checks.Secrets.Enabled && spec.Checks.Secrets.Mode != "off"
	case "malicious_code":
		return spec.Checks.MaliciousCode.Enabled && spec.Checks.MaliciousCode.Mode != "off"
	case "dependencies":
		return spec.Checks.Dependencies.Enabled && spec.Checks.Dependencies.Mode != "off"
	case "lint":
		return spec.Checks.Lint.Enabled && spec.Checks.Lint.Mode != "off"
	case "format":
		return spec.Checks.Format.Enabled && spec.Checks.Format.Mode != "off"
	}
	return false
}

func shouldUseColor(cfg string, flagNoColor bool) bool {
	if flagNoColor || os.Getenv("NO_COLOR") != "" || os.Getenv("LINTEL_NO_COLOR") != "" {
		return false
	}
	switch cfg {
	case "always":
		return true
	case "never":
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func projectName(root string) string {
	// best-effort: last path segment
	parts := strings.Split(strings.ReplaceAll(root, "\\", "/"), "/")
	return parts[len(parts)-1]
}

func contains(xs []string, needle string) bool {
	for _, x := range xs {
		if x == needle {
			return true
		}
	}
	return false
}
