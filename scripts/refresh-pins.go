//go:build ignore

// Command refresh-pins downloads every scanner asset declared in
// internal/installer/scanners.yaml, computes both its archive_sha256 and
// (after extraction) its binary_sha256, and rewrites the file in place.
//
// This is a release-engineering tool, not part of the shipped lintel binary
// (the //go:build ignore tag excludes it from the main module's build).
// Run it before cutting a release:
//
//	go run ./scripts/refresh-pins.go
//
// Flags:
//
//	-n       dry run: download, hash, and extract, but do not rewrite the file
//	-yaml    path to scanners.yaml (default: internal/installer/scanners.yaml)
//
// The rewrite is a minimal textual substitution — only the two sha256
// values change per entry, and the file's indentation, blank lines,
// quoting, and header comments are preserved byte-for-byte outside those
// edits. Substitutions preserve length (both hashes are 64-char hex), so
// splicing order inside a single entry doesn't matter.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/MHChlagou/lintel/internal/installer"
)

// pinEntry matches a url + archive_sha256 + binary_sha256 triplet. The
// capture groups isolate the URL and the exact byte ranges of the two hash
// values so they can be spliced in place.
var pinEntry = regexp.MustCompile(
	`(?s)url:\s*"([^"]+)"` +
		`\s+archive_sha256:\s*"([0-9a-fA-F]+)"` +
		`\s+binary_sha256:\s*"([0-9a-fA-F]+)"`)

// archiveLine and binaryName match the `archive:` and `binary:` declarations
// of the enclosing version block, scanned backward from a pin entry.
var archiveLine = regexp.MustCompile(`archive:\s+([a-z.]+)`)
var binaryName = regexp.MustCompile(`binary:\s+([A-Za-z0-9_-]+)`)

func main() {
	var (
		dryRun   bool
		yamlPath string
	)
	flag.BoolVar(&dryRun, "n", false, "dry run: download and hash but do not rewrite the file")
	flag.StringVar(&yamlPath, "yaml", "internal/installer/scanners.yaml", "path to scanners.yaml")
	flag.Parse()

	raw, err := os.ReadFile(yamlPath)
	must(err)

	client := &http.Client{Timeout: 5 * time.Minute}
	matches := pinEntry.FindAllSubmatchIndex(raw, -1)
	fmt.Printf("found %d pin entries\n\n", len(matches))

	newRaw := append([]byte(nil), raw...)
	var failed []string
	var updated int

	for _, m := range matches {
		url := string(raw[m[2]:m[3]])
		archiveSpan := [2]int{m[4], m[5]}
		binarySpan := [2]int{m[6], m[7]}
		currentArchive := string(raw[archiveSpan[0]:archiveSpan[1]])
		currentBinary := string(raw[binarySpan[0]:binarySpan[1]])

		archiveType, binName := contextFor(raw, m[0])
		fmt.Printf("→ %s\n  archive=%s binary=%s\n", url, archiveType, binName)

		archiveHash, binaryHash, err := fetchAndExtract(client, url, archiveType, binName)
		if err != nil {
			fmt.Printf("  ✖ %v\n\n", err)
			failed = append(failed, fmt.Sprintf("%s: %v", url, err))
			continue
		}

		if currentArchive == archiveHash && currentBinary == binaryHash {
			fmt.Printf("  = already current\n\n")
			continue
		}
		if currentArchive != archiveHash {
			fmt.Printf("  ✓ archive_sha256 → %s\n    was:            %s\n", archiveHash, currentArchive)
			newRaw = splice(newRaw, archiveSpan[0], archiveSpan[1], archiveHash)
			updated++
		}
		if currentBinary != binaryHash {
			fmt.Printf("  ✓ binary_sha256  → %s\n    was:           %s\n", binaryHash, currentBinary)
			newRaw = splice(newRaw, binarySpan[0], binarySpan[1], binaryHash)
			updated++
		}
		fmt.Println()
	}

	fmt.Printf("visited %d entr(ies); %d hash value(s) updated; %d failed\n", len(matches), updated, len(failed))

	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "\nfailures:")
		for _, f := range failed {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("(dry run — file not written)")
		return
	}
	if updated == 0 {
		return
	}
	must(os.WriteFile(yamlPath, newRaw, 0o644))
	fmt.Printf("wrote %s\n", yamlPath)
}

// contextFor finds the archive type and binary name from the enclosing
// version block by scanning backward from the pin entry and taking the
// NEAREST `archive:` and `binary:` declarations — not the first ones in
// the file, which is what a plain FindSubmatch would return. A 4 KB
// window safely covers the largest version block (8 platforms) without
// leaking into the previous scanner's block.
func contextFor(raw []byte, matchStart int) (installer.ArchiveType, string) {
	const span = 4096
	start := matchStart - span
	if start < 0 {
		start = 0
	}
	window := raw[start:matchStart]

	archive := installer.ArchiveRaw
	if all := archiveLine.FindAllSubmatch(window, -1); len(all) > 0 {
		archive = installer.ArchiveType(string(all[len(all)-1][1]))
	}
	name := ""
	if all := binaryName.FindAllSubmatch(window, -1); len(all) > 0 {
		name = string(all[len(all)-1][1])
	}
	return archive, name
}

func fetchAndExtract(client *http.Client, url string, archive installer.ArchiveType, binName string) (string, string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return "", "", fmt.Errorf("HTTP %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "refresh-pins-*")
	if err != nil {
		return "", "", err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		_ = tmp.Close()
		return "", "", err
	}
	if err := tmp.Close(); err != nil {
		return "", "", err
	}
	archiveHash := hex.EncodeToString(h.Sum(nil))

	dir, err := os.MkdirTemp("", "refresh-pins-extract-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	dest := filepath.Join(dir, binName)
	if err := installer.ExtractBinary(tmpName, archive, binName, dest); err != nil {
		return "", "", fmt.Errorf("extract: %w", err)
	}

	f, err := os.Open(dest)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = f.Close() }()
	bh := sha256.New()
	if _, err := io.Copy(bh, f); err != nil {
		return "", "", err
	}
	return archiveHash, hex.EncodeToString(bh.Sum(nil)), nil
}

func splice(buf []byte, start, end int, replacement string) []byte {
	out := make([]byte, 0, len(buf))
	out = append(out, buf[:start]...)
	out = append(out, replacement...)
	out = append(out, buf[end:]...)
	return out
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
