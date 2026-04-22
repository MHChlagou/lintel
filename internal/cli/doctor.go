package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/aegis-sec/aegis/internal/config"
	"github.com/aegis-sec/aegis/internal/resolve"
)

func cmdDoctor() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Verify scanner binaries, versions, and sha256 hashes",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := resolveRepoRoot()
			spec, err := config.Load(root, flags.configPath)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "platform: %s_%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(out, "strict_versions: %v\n\n", spec.StrictVers)
			r := resolve.New(root, spec.Binaries, spec.StrictVers)
			var failed int
			for name := range spec.Binaries {
				rb, err := r.Resolve(name)
				if err != nil {
					fmt.Fprintf(out, "  ✖ %-14s %v\n", name, err)
					failed++
					continue
				}
				status := "verified"
				if !rb.HashVerified {
					status = "permissive (no hash in spec for this platform)"
				}
				fmt.Fprintf(out, "  ✓ %-14s %s  [%s]\n", name, rb.Path, status)
			}
			if failed > 0 {
				return fmt.Errorf("%d binary problem(s)", failed)
			}
			fmt.Fprintln(out, "\nall binaries resolved")
			return nil
		},
	}
}
