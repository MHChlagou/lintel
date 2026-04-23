package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MHChlagou/lintel/internal/config"
	"github.com/MHChlagou/lintel/internal/installer"
	"github.com/MHChlagou/lintel/internal/resolve"
)

// Writer errors on stdout/stderr are non-actionable, so these helpers swallow
// them to keep call sites readable. Used across the cli package.
func fpf(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
func fpln(w io.Writer, args ...any)               { _, _ = fmt.Fprintln(w, args...) }

// newResolverWithPinFallback builds a resolver and wires the embedded pin
// DB as a fallback so user yaml files without sha256 entries still hit
// verified hashes for scanners shipped with lintel. A failed Load here is
// non-fatal — we keep a resolver without fallback and let the normal
// missing-hash path handle it — because the embedded pin DB is a
// correctness enhancement, not a precondition for running.
func newResolverWithPinFallback(repoRoot string, binaries map[string]config.Binary, strict bool) *resolve.Resolver {
	r := resolve.New(repoRoot, binaries, strict)
	if reg, err := installer.Load(); err == nil {
		r.SetPinFallback(reg)
	}
	return r
}

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
