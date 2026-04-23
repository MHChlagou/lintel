package cli

import (
	"fmt"
	"io"
	"runtime"
	"sort"

	"github.com/spf13/cobra"

	"github.com/MHChlagou/lintel/internal/config"
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
			fpf(out, "platform: %s_%s\n", runtime.GOOS, runtime.GOARCH)
			fpf(out, "strict_versions: %v\n\n", spec.StrictVers)
			r := newResolverWithPinFallback(root, spec.Binaries, spec.StrictVers)

			names := sortedBinaryNames(spec.Binaries)
			var failed []string
			for _, name := range names {
				rb, err := r.Resolve(name)
				if err != nil {
					fpf(out, "  ✖ %-14s %v\n", name, err)
					failed = append(failed, name)
					continue
				}
				status := "verified"
				if !rb.HashVerified {
					status = "permissive (no hash in spec for this platform)"
				}
				fpf(out, "  ✓ %-14s %s  [%s]\n", name, rb.Path, status)
			}

			if len(failed) > 0 {
				fpln(out, "")
				renderFixHint(out, failed)
				return fmt.Errorf("%d binary problem(s)", len(failed))
			}
			fpln(out, "\nall binaries resolved")
			return nil
		},
	}
}

// renderFixHint prints a single consolidated remediation hint after a doctor
// run that had failures. The rule is simple: name the one scanner if only
// one failed, otherwise point at --all. gofmt is special-cased because it
// ships with the Go toolchain and cannot be fetched by `lintel install`.
func renderFixHint(w io.Writer, failed []string) {
	installable := make([]string, 0, len(failed))
	gofmtFailed := false
	for _, name := range failed {
		if name == "gofmt" {
			gofmtFailed = true
			continue
		}
		installable = append(installable, name)
	}

	switch len(installable) {
	case 0:
		if gofmtFailed {
			fpln(w, "↳ gofmt ships with the Go toolchain — install Go to fix this row")
		}
	case 1:
		fpf(w, "↳ run: lintel install %s\n", installable[0])
	default:
		fpln(w, "↳ run: lintel install --all")
	}

	if gofmtFailed && len(installable) > 0 {
		fpln(w, "  (gofmt ships with the Go toolchain — install Go separately)")
	}
}

func sortedBinaryNames(m map[string]config.Binary) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
