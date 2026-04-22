package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/aegis-sec/aegis/internal/filter"
)

// cmdIgnore adds a rule+path pair to .aegis/allowlist.yaml.
func cmdIgnore() *cobra.Command {
	var path, reason, expires string
	c := &cobra.Command{
		Use:   "ignore <rule>",
		Short: "Add a rule to .aegis/allowlist.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reason == "" {
				return fmt.Errorf("--reason is required")
			}
			root := resolveRepoRoot()
			al, err := filter.LoadAllowlist(root)
			if err != nil {
				return err
			}
			al.Entries = append(al.Entries, filter.AllowEntry{
				Rule: args[0], Path: path, Reason: reason, Expires: expires,
			})
			raw, _ := yaml.Marshal(al)
			p := filepath.Join(root, ".aegis", "allowlist.yaml")
			if err := os.WriteFile(p, raw, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ added allowlist entry for rule=%s\n", args[0])
			return nil
		},
	}
	c.Flags().StringVar(&path, "path", "", "glob to scope the ignore")
	c.Flags().StringVar(&reason, "reason", "", "required justification")
	c.Flags().StringVar(&expires, "expires", "", "optional YYYY-MM-DD expiry")
	return c
}

// cmdFmt is a shortcut for `aegis run --check format`.
func cmdFmt() *cobra.Command {
	return &cobra.Command{
		Use:   "fmt",
		Short: "Shortcut for `aegis run --check format`",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Root().SetArgs([]string{"run", "--check", "format"})
			return cmd.Root().Execute()
		},
	}
}

// cmdExplain is a stub that delegates to the scanner's own documentation URL.
func cmdExplain() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <rule>",
		Short: "Print documentation pointer for a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "rule %s: consult the scanner's own registry\n", args[0])
			return nil
		},
	}
}
