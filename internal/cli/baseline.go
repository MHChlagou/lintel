package cli

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/MHChlagou/lintel/internal/checker"
	"github.com/MHChlagou/lintel/internal/config"
	"github.com/MHChlagou/lintel/internal/detect"
	"github.com/MHChlagou/lintel/internal/filter"
	"github.com/MHChlagou/lintel/internal/finding"
	"github.com/MHChlagou/lintel/internal/runner"
)

func cmdBaseline() *cobra.Command {
	return &cobra.Command{
		Use:   "baseline",
		Short: "Snapshot current findings into .lintel/baseline.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := resolveRepoRoot()
			spec, err := config.Load(root, flags.configPath)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			staged, _ := detect.StagedFiles(ctx, root)
			proj, _ := detect.Detect(root, nil, spec.Scope.ExcludePaths, staged)
			res := newResolverWithPinFallback(root, spec.Binaries, spec.StrictVers)
			reg := checker.Registry()
			checks := []string{"secrets", "malicious_code", "dependencies"}
			mkIn := func(name string) checker.CheckInput {
				return checker.CheckInput{
					RepoRoot: root, StagedFiles: staged, FullTree: true,
					Config: json.RawMessage("{}"), Spec: spec, Project: proj, Resolver: res,
				}
			}
			results := runner.Run(ctx, checks, mkIn, reg, runner.Options{Spec: spec})
			var all []finding.Finding
			for _, r := range results {
				all = append(all, r.Output.Findings...)
			}
			if err := filter.SaveBaseline(root, all, time.Now().UTC().Format(time.RFC3339)); err != nil {
				return err
			}
			fpf(os.Stdout, "✓ baseline captured: %d findings\n", len(all))
			return nil
		},
	}
}
