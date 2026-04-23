package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/MHChlagou/lintel/internal/config"
	"github.com/MHChlagou/lintel/internal/hook"
	"github.com/MHChlagou/lintel/internal/installer"
)

func cmdInstall() *cobra.Command {
	var force bool
	var all bool
	c := &cobra.Command{
		Use:   "install [scanner]",
		Short: "Install git hooks, or fetch a pinned scanner into ~/.lintel/bin",
		Long: `Without arguments, installs lintel-managed git hooks into .git/hooks.
With a scanner name (e.g. 'lintel install gitleaks'), downloads the pinned
release from the embedded pin database, verifies its sha256, and places
the verified binary at ~/.lintel/bin/<scanner>. Use --all to install every
scanner declared in the loaded lintel.yaml (skipping gofmt, which ships
with the Go toolchain).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch {
			case all:
				return installAllScanners(cmd, out)
			case len(args) == 0:
				return installHooks(out, force)
			case len(args) == 1:
				return installOneScanner(cmd, out, args[0])
			default:
				return fmt.Errorf("install takes at most one scanner name (got %d args)", len(args))
			}
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite pre-existing non-lintel hooks")
	c.Flags().BoolVar(&all, "all", false, "install every scanner declared in lintel.yaml")
	return c
}

func installHooks(out io.Writer, force bool) error {
	installed, skipped, err := hook.Install(resolveRepoRoot(), force)
	if err != nil {
		return err
	}
	for _, n := range installed {
		fpf(out, "✓ installed hook: %s\n", n)
	}
	for _, n := range skipped {
		fpf(out, "! skipped: existing non-lintel hook %s (use --force to overwrite)\n", n)
	}
	return nil
}

func installOneScanner(cmd *cobra.Command, out io.Writer, name string) error {
	spec, err := config.Load(resolveRepoRoot(), flags.configPath)
	if err != nil {
		return err
	}
	bin, ok := spec.Binaries[name]
	if !ok {
		return fmt.Errorf("scanner %q is not declared in lintel.yaml (declared: %v)", name, sortedKeys(spec.Binaries))
	}
	if name == "gofmt" {
		return fmt.Errorf("gofmt ships with the Go toolchain; install Go instead of running `lintel install gofmt`")
	}
	reg, err := installer.Load()
	if err != nil {
		return err
	}
	res, err := installer.Install(reg, installer.Options{
		Scanner:  name,
		Version:  bin.Version,
		Progress: cmd.OutOrStderr(),
	})
	if err != nil {
		return err
	}
	fpf(out, "✓ installed %s@%s → %s\n", res.Scanner, res.Version, res.InstalledAt)
	fpf(out, "  archive sha256: %s\n", res.ArchiveSHA256)
	fpf(out, "  binary sha256:  %s\n", res.BinarySHA256)
	return nil
}

func installAllScanners(cmd *cobra.Command, out io.Writer) error {
	spec, err := config.Load(resolveRepoRoot(), flags.configPath)
	if err != nil {
		return err
	}
	reg, err := installer.Load()
	if err != nil {
		return err
	}
	names := sortedKeys(spec.Binaries)
	var failed []string
	for _, name := range names {
		if name == "gofmt" {
			fpf(out, "↳ skipping gofmt (ships with Go toolchain)\n")
			continue
		}
		bin := spec.Binaries[name]
		res, err := installer.Install(reg, installer.Options{
			Scanner:  name,
			Version:  bin.Version,
			Progress: cmd.OutOrStderr(),
		})
		if err != nil {
			fpf(out, "✖ %s: %v\n", name, err)
			failed = append(failed, name)
			continue
		}
		fpf(out, "✓ installed %s@%s → %s\n", res.Scanner, res.Version, res.InstalledAt)
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d scanner(s) failed to install: %v", len(failed), failed)
	}
	return nil
}

func sortedKeys(m map[string]config.Binary) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func cmdUninstall() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove lintel-installed git hooks (foreign hooks are preserved)",
		RunE: func(cmd *cobra.Command, args []string) error {
			removed, err := hook.Uninstall(resolveRepoRoot())
			if err != nil {
				return err
			}
			for _, n := range removed {
				fpf(cmd.OutOrStdout(), "✓ removed hook: %s\n", n)
			}
			return nil
		},
	}
}
