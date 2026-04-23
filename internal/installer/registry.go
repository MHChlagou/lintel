// Package installer downloads, verifies, and installs pinned scanner
// binaries into ~/.lintel/bin. The pin database (scanners.yaml) is embedded
// into the lintel binary so a release ships one signed artifact with
// everything needed to fetch its declared scanners.
package installer

import (
	_ "embed"
	"fmt"
	"runtime"

	"gopkg.in/yaml.v3"
)

//go:embed scanners.yaml
var scannersYAML []byte

// ArchiveType is how an upstream asset is packaged.
type ArchiveType string

const (
	ArchiveTarGz ArchiveType = "tar.gz"
	ArchiveTarXz ArchiveType = "tar.xz"
	ArchiveZip   ArchiveType = "zip"
	ArchiveRaw   ArchiveType = "raw" // the asset IS the binary
)

// PlatformAsset is one download target for one platform.
//
// Two hashes are pinned because a release asset is verified in two distinct
// places: ArchiveSHA256 is checked against the bytes on the wire at install
// time; BinarySHA256 is checked against the extracted binary on disk at run
// time by the resolver. For archive:raw scanners the two values are equal;
// for tar.gz/tar.xz/zip they differ, and both must be pinned for end-to-end
// verification to hold.
type PlatformAsset struct {
	URL           string `yaml:"url"`
	ArchiveSHA256 string `yaml:"archive_sha256"`
	BinarySHA256  string `yaml:"binary_sha256"`
}

// VersionEntry is one version of one scanner, across platforms.
type VersionEntry struct {
	Archive   ArchiveType              `yaml:"archive"`
	Binary    string                   `yaml:"binary"`
	Platforms map[string]PlatformAsset `yaml:"platforms"`
}

// Scanner is the full history of pinned versions for one scanner.
type Scanner struct {
	Versions map[string]VersionEntry `yaml:"versions"`
}

// Registry is the loaded pin database.
type Registry struct {
	Scanners map[string]Scanner `yaml:"scanners"`
}

// Load parses the embedded scanners.yaml.
func Load() (*Registry, error) {
	var r Registry
	if err := yaml.Unmarshal(scannersYAML, &r); err != nil {
		return nil, fmt.Errorf("parse embedded scanners.yaml: %w", err)
	}
	return &r, nil
}

// Lookup returns the asset for (scanner, version, current platform) or an
// error explaining exactly what was missing. The platform argument is
// normally runtime.GOOS + "_" + runtime.GOARCH; tests may pass custom keys.
func (r *Registry) Lookup(name, version, platform string) (VersionEntry, PlatformAsset, error) {
	sc, ok := r.Scanners[name]
	if !ok {
		return VersionEntry{}, PlatformAsset{}, fmt.Errorf("scanner %q is not in the pin database (known: %v)", name, r.scannerNames())
	}
	ve, ok := sc.Versions[version]
	if !ok {
		return VersionEntry{}, PlatformAsset{}, fmt.Errorf("scanner %s has no pinned entry for version %s (pinned: %v)", name, version, versionList(sc))
	}
	pa, ok := ve.Platforms[platform]
	if !ok {
		return VersionEntry{}, PlatformAsset{}, fmt.Errorf("scanner %s@%s has no asset for platform %s", name, version, platform)
	}
	if isPlaceholderHash(pa.ArchiveSHA256) || isPlaceholderHash(pa.BinarySHA256) {
		return VersionEntry{}, PlatformAsset{}, fmt.Errorf("scanner %s@%s on %s has a placeholder sha256 — the release pipeline has not populated pins yet; refusing to download unverifiable bytes", name, version, platform)
	}
	return ve, pa, nil
}

// CurrentPlatform returns the runtime platform key.
func CurrentPlatform() string {
	return runtime.GOOS + "_" + runtime.GOARCH
}

// LookupHash returns the pinned binary sha256 for (scanner, version,
// platform), or "" when the triplet is unknown or still holds a
// placeholder. Binary (not archive) because this is the hash a Resolver
// will compare against — it hashes the extracted binary on disk, never
// the original archive. Satisfies resolve.PinLookup.
func (r *Registry) LookupHash(scanner, version, platform string) string {
	sc, ok := r.Scanners[scanner]
	if !ok {
		return ""
	}
	ve, ok := sc.Versions[version]
	if !ok {
		return ""
	}
	pa, ok := ve.Platforms[platform]
	if !ok {
		return ""
	}
	if isPlaceholderHash(pa.BinarySHA256) {
		return ""
	}
	return pa.BinarySHA256
}

func (r *Registry) scannerNames() []string {
	out := make([]string, 0, len(r.Scanners))
	for k := range r.Scanners {
		out = append(out, k)
	}
	return out
}

func versionList(s Scanner) []string {
	out := make([]string, 0, len(s.Versions))
	for v := range s.Versions {
		out = append(out, v)
	}
	return out
}

func isPlaceholderHash(h string) bool {
	if len(h) != 64 {
		return true
	}
	for _, c := range h {
		if c != '0' {
			return false
		}
	}
	return true
}
