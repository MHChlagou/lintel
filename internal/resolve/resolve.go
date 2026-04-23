// Package resolve locates scanner binaries and verifies them against sha256 hashes.
package resolve

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/MHChlagou/lintel/internal/config"
)

// ResolvedBinary is the output of resolution: where the binary lives, what
// hash it has right now, and whether that matches the spec.
type ResolvedBinary struct {
	Name         string
	Path         string
	Command      string
	ExpectedHash string // empty if platform key missing
	ActualHash   string
	HashVerified bool
	Version      string
	InstallHint  string
}

// PinLookup is the narrow interface the resolver uses to fall back to a
// shipped pin database (e.g. internal/installer.Registry) when the user's
// lintel.yaml has no sha256 entry for a given scanner/platform. Returning
// "" means "no pinned hash available"; callers then apply their usual
// strict/permissive handling.
type PinLookup interface {
	LookupHash(scanner, version, platform string) string
}

type Resolver struct {
	repoRoot string
	binaries map[string]config.Binary
	strict   bool
	pins     PinLookup

	mu    sync.Mutex
	cache map[string]*ResolvedBinary
}

func New(repoRoot string, binaries map[string]config.Binary, strict bool) *Resolver {
	return &Resolver{
		repoRoot: repoRoot,
		binaries: binaries,
		strict:   strict,
		cache:    map[string]*ResolvedBinary{},
	}
}

// SetPinFallback installs a secondary source for expected hashes. It is
// only consulted when the user's lintel.yaml leaves a (scanner, platform)
// hash empty — any explicit user pin takes precedence.
func (r *Resolver) SetPinFallback(p PinLookup) {
	r.pins = p
}

// Resolve locates and verifies the named binary. Results are memoized for the
// life of the Resolver (one process == one lintel run).
func (r *Resolver) Resolve(name string) (*ResolvedBinary, error) {
	r.mu.Lock()
	if rb, ok := r.cache[name]; ok {
		r.mu.Unlock()
		return rb, nil
	}
	r.mu.Unlock()

	spec, ok := r.binaries[name]
	if !ok {
		return nil, fmt.Errorf("binary %q not declared in spec.binaries", name)
	}
	path, err := r.locate(name, spec)
	if err != nil {
		return nil, err
	}
	actual, err := hashFile(path)
	if err != nil {
		return nil, fmt.Errorf("hash %s: %w", path, err)
	}
	platformKey := runtime.GOOS + "_" + runtime.GOARCH
	expected := spec.SHA256[platformKey]
	if expected == "" && r.pins != nil {
		expected = r.pins.LookupHash(name, spec.Version, platformKey)
	}
	rb := &ResolvedBinary{
		Name:         name,
		Path:         path,
		Command:      ifEmpty(spec.Command, name),
		ExpectedHash: expected,
		ActualHash:   actual,
		Version:      spec.Version,
		InstallHint:  spec.InstallHint,
	}
	switch {
	case expected == "":
		if r.strict {
			return nil, fmt.Errorf("binary %s: no sha256 for platform %s; strict_versions=true refuses to run", name, platformKey)
		}
		// permissive: proceed with HashVerified=false
	case !strings.EqualFold(expected, actual):
		return nil, fmt.Errorf("binary %s sha256 mismatch\n  expected: %s\n  actual:   %s\n  path:     %s\n  refusing to execute", name, expected, actual, path)
	default:
		rb.HashVerified = true
	}
	r.mu.Lock()
	r.cache[name] = rb
	r.mu.Unlock()
	return rb, nil
}

// locate implements the resolution order from spec §6.1.
func (r *Resolver) locate(name string, spec config.Binary) (string, error) {
	var tried []string
	// 1. Explicit config path.
	if spec.Path != "" {
		p := expandHome(spec.Path)
		tried = append(tried, p)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	// 2. $LINTEL_BIN_DIR.
	if dir := os.Getenv("LINTEL_BIN_DIR"); dir != "" {
		p := filepath.Join(expandHome(dir), name)
		tried = append(tried, p)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	// 3. ~/.lintel/bin/<name>.
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".lintel", "bin", name)
		tried = append(tried, p)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	// 4. $PATH lookup.
	cmd := ifEmpty(spec.Command, name)
	if p, err := exec.LookPath(cmd); err == nil {
		return p, nil
	}
	tried = append(tried, "$PATH")
	hint := spec.InstallHint
	if hint == "" {
		hint = "(no install_hint in spec)"
	}
	return "", fmt.Errorf("required binary not found: %s (v%s)\n  searched: %s\n  install:  %s\n  hint:     drop the binary on $PATH or at ~/.lintel/bin/%s, then run `lintel doctor`",
		name, ifEmpty(spec.Version, "?"), strings.Join(tried, ", "), hint, name)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
