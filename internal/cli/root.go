// Package cli wires cobra subcommands for the aegis binary.
package cli

import (
	"github.com/spf13/cobra"
)

type globalFlags struct {
	configPath string
	output     string
	quiet      bool
	verbose    bool
	noColor    bool
	repoRoot   string
}

var flags = &globalFlags{}

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "aegis",
		Short: "Aegis - the shield your commits pass through",
		Long: `Aegis is a shift-left security orchestrator that runs as a Git hook.
It coordinates best-in-class open-source scanners (gitleaks, opengrep,
osv-scanner, biome, ruff, golangci-lint, …) driven by one declarative spec.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&flags.configPath, "config", "", "path to aegis.yaml (default: .aegis/aegis.yaml)")
	root.PersistentFlags().StringVar(&flags.output, "output", "", "output format: pretty|json|sarif|junit")
	root.PersistentFlags().BoolVar(&flags.quiet, "quiet", false, "only emit errors")
	root.PersistentFlags().BoolVar(&flags.verbose, "verbose", false, "include debug output")
	root.PersistentFlags().BoolVar(&flags.noColor, "no-color", false, "disable colored output")
	root.PersistentFlags().StringVar(&flags.repoRoot, "repo", "", "path to repo root (default: $PWD)")

	root.AddCommand(
		cmdVersion(),
		cmdInit(),
		cmdInstall(),
		cmdUninstall(),
		cmdRun(),
		cmdDoctor(),
		cmdBaseline(),
		cmdIgnore(),
		cmdFmt(),
		cmdExplain(),
		cmdUpgrade(),
	)
	return root
}
