package cli

import (
	"os"
	"path/filepath"
)

// resolveRepoRoot returns the effective repo root: --repo flag, env, or cwd.
func resolveRepoRoot() string {
	if flags.repoRoot != "" {
		p, _ := filepath.Abs(flags.repoRoot)
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
