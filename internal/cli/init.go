package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aegis-sec/aegis/internal/config"
)

func cmdInit() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Create .aegis/aegis.yaml with secure defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := resolveRepoRoot()
			dir := filepath.Join(root, ".aegis")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			target := filepath.Join(dir, "aegis.yaml")
			if _, err := os.Stat(target); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", target)
			}
			name := filepath.Base(root)
			body := strings.Replace(config.DefaultSpecYAML, "{{.ProjectName}}", name, 1)
			if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
				return err
			}
			allowlist := filepath.Join(dir, "allowlist.yaml")
			if _, err := os.Stat(allowlist); os.IsNotExist(err) {
				_ = os.WriteFile(allowlist, []byte(config.DefaultAllowlistYAML), 0o644)
			}
			// gitignore overrides.log if there's a .gitignore
			appendLineIfMissing(filepath.Join(root, ".gitignore"), ".aegis/overrides.log")
			fmt.Fprintf(cmd.OutOrStdout(), "✓ wrote %s\n", target)
			fmt.Fprintln(cmd.OutOrStdout(), "  next: `aegis install` to wire up git hooks")
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite an existing aegis.yaml")
	return c
}

func appendLineIfMissing(path, line string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		// create a new .gitignore only if the repo already has a git dir
		if _, errg := os.Stat(filepath.Join(filepath.Dir(path), ".git")); errg == nil {
			_ = os.WriteFile(path, []byte(line+"\n"), 0o644)
		}
		return
	}
	if strings.Contains(string(raw), line) {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString("\n" + line + "\n")
}
