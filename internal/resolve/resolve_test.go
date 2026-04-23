package resolve

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MHChlagou/lintel/internal/config"
)

// writeFakeBinary creates an executable file with known content and returns its path + hash.
func writeFakeBinary(t *testing.T, dir, name string) (string, string) {
	t.Helper()
	path := filepath.Join(dir, name)
	body := []byte("#!/bin/sh\necho fake " + name + "\n")
	if err := os.WriteFile(path, body, 0o755); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(body)
	return path, hex.EncodeToString(h[:])
}

func TestResolveHashMatch(t *testing.T) {
	dir := t.TempDir()
	path, hash := writeFakeBinary(t, dir, "gitleaks")
	platformKey := runtime.GOOS + "_" + runtime.GOARCH
	bin := config.Binary{
		Command: "gitleaks",
		Path:    path,
		Version: "8.28.0",
		SHA256:  map[string]string{platformKey: hash},
	}
	r := New(dir, map[string]config.Binary{"gitleaks": bin}, true)
	rb, err := r.Resolve("gitleaks")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !rb.HashVerified {
		t.Fatal("expected HashVerified=true")
	}
	if rb.Path != path {
		t.Fatalf("path = %s, want %s", rb.Path, path)
	}
}

func TestResolveHashMismatchRefuses(t *testing.T) {
	dir := t.TempDir()
	path, _ := writeFakeBinary(t, dir, "gitleaks")
	platformKey := runtime.GOOS + "_" + runtime.GOARCH
	bin := config.Binary{
		Command: "gitleaks",
		Path:    path,
		Version: "8.28.0",
		SHA256:  map[string]string{platformKey: "0000000000000000000000000000000000000000000000000000000000000000"},
	}
	r := New(dir, map[string]config.Binary{"gitleaks": bin}, true)
	_, err := r.Resolve("gitleaks")
	if err == nil {
		t.Fatal("expected sha256 mismatch error")
	}
}

func TestResolveStrictRejectsMissingHash(t *testing.T) {
	dir := t.TempDir()
	path, _ := writeFakeBinary(t, dir, "gitleaks")
	bin := config.Binary{Command: "gitleaks", Path: path, Version: "8.28.0", SHA256: map[string]string{}}
	r := New(dir, map[string]config.Binary{"gitleaks": bin}, true)
	_, err := r.Resolve("gitleaks")
	if err == nil {
		t.Fatal("strict mode should reject missing hash")
	}
}

// stubPins implements PinLookup for tests without dragging in the installer.
type stubPins map[string]string

func (s stubPins) LookupHash(scanner, version, platform string) string {
	return s[scanner+"@"+version+"/"+platform]
}

func TestResolve_PinFallback_Used(t *testing.T) {
	// User's lintel.yaml has no sha256 for this platform. The resolver should
	// consult the pin fallback and verify against that.
	dir := t.TempDir()
	path, hash := writeFakeBinary(t, dir, "gitleaks")
	platformKey := runtime.GOOS + "_" + runtime.GOARCH
	bin := config.Binary{
		Command: "gitleaks",
		Path:    path,
		Version: "8.28.0",
		SHA256:  map[string]string{}, // intentionally empty
	}
	r := New(dir, map[string]config.Binary{"gitleaks": bin}, true)
	r.SetPinFallback(stubPins{"gitleaks@8.28.0/" + platformKey: hash})
	rb, err := r.Resolve("gitleaks")
	if err != nil {
		t.Fatalf("expected resolve to succeed via fallback, got %v", err)
	}
	if !rb.HashVerified {
		t.Fatal("expected HashVerified=true after fallback match")
	}
}

func TestResolve_UserPinWinsOverFallback(t *testing.T) {
	// When the user's yaml has an explicit hash, the fallback must NOT be
	// consulted — even if the fallback would say something different.
	dir := t.TempDir()
	path, hash := writeFakeBinary(t, dir, "gitleaks")
	platformKey := runtime.GOOS + "_" + runtime.GOARCH
	bin := config.Binary{
		Command: "gitleaks",
		Path:    path,
		Version: "8.28.0",
		SHA256:  map[string]string{platformKey: hash}, // explicit user pin
	}
	r := New(dir, map[string]config.Binary{"gitleaks": bin}, true)
	// Fallback disagrees — user pin must still win.
	r.SetPinFallback(stubPins{"gitleaks@8.28.0/" + platformKey: "wronghash"})
	rb, err := r.Resolve("gitleaks")
	if err != nil {
		t.Fatalf("expected resolve to succeed via user pin, got %v", err)
	}
	if !rb.HashVerified {
		t.Fatal("expected HashVerified=true using the user-provided pin")
	}
}

func TestResolve_BothMissing_StrictStillRefuses(t *testing.T) {
	// Neither user yaml nor fallback provides a hash. Strict mode must
	// still refuse — the fallback is a convenience, not a loophole.
	dir := t.TempDir()
	path, _ := writeFakeBinary(t, dir, "gitleaks")
	bin := config.Binary{
		Command: "gitleaks",
		Path:    path,
		Version: "99.99.99", // version not in any pin DB
		SHA256:  map[string]string{},
	}
	r := New(dir, map[string]config.Binary{"gitleaks": bin}, true)
	r.SetPinFallback(stubPins{}) // empty fallback
	_, err := r.Resolve("gitleaks")
	if err == nil {
		t.Fatal("expected strict refusal when both user pin and fallback are empty")
	}
}

func TestResolveNotFound(t *testing.T) {
	r := New(t.TempDir(), map[string]config.Binary{
		"ghostscan": {Command: "ghostscan-does-not-exist-xyz", Version: "1.0"},
	}, false)
	_, err := r.Resolve("ghostscan")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}
