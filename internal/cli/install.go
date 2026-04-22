package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aegis-sec/aegis/internal/hook"
)

func cmdInstall() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "install",
		Short: "Install git hooks that dispatch to aegis run",
		RunE: func(cmd *cobra.Command, args []string) error {
			installed, skipped, err := hook.Install(resolveRepoRoot(), force)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, n := range installed {
				fmt.Fprintf(out, "✓ installed hook: %s\n", n)
			}
			for _, n := range skipped {
				fmt.Fprintf(out, "! skipped: existing non-aegis hook %s (use --force to overwrite)\n", n)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite pre-existing non-aegis hooks")
	return c
}

func cmdUninstall() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove aegis-installed git hooks (foreign hooks are preserved)",
		RunE: func(cmd *cobra.Command, args []string) error {
			removed, err := hook.Uninstall(resolveRepoRoot())
			if err != nil {
				return err
			}
			for _, n := range removed {
				fmt.Fprintf(cmd.OutOrStdout(), "✓ removed hook: %s\n", n)
			}
			return nil
		},
	}
}
