package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options configures an Install call. Callers normally just set Scanner and
// Version; the rest has working defaults.
type Options struct {
	Scanner      string
	Version      string
	Platform     string // empty = runtime.GOOS + "_" + runtime.GOARCH
	DestDir      string // empty = ~/.lintel/bin
	HTTP         *http.Client
	Progress     io.Writer       // optional; receives "  → downloading …" style lines
	AllowedHosts map[string]bool // optional override of the default allowlist (tests, private mirrors)
}

// Result describes where the binary ended up after a successful install.
type Result struct {
	Scanner       string
	Version       string
	Platform      string
	InstalledAt   string
	ArchiveSHA256 string // hash of the download as received
	BinarySHA256  string // hash of the extracted binary on disk
	URL           string
}

// Install fetches one scanner from its pinned URL, verifies its sha256 against
// the embedded pin DB, and places the binary at <DestDir>/<scanner>.
//
// Errors are deliberately descriptive — this is a user-facing path triggered
// by `lintel install <tool>` or by `lintel doctor`'s suggested remediation.
func Install(reg *Registry, opt Options) (*Result, error) {
	if opt.Scanner == "" || opt.Version == "" {
		return nil, fmt.Errorf("installer: scanner and version are required")
	}
	if opt.Platform == "" {
		opt.Platform = CurrentPlatform()
	}
	if opt.DestDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("locate user home: %w", err)
		}
		opt.DestDir = filepath.Join(home, ".lintel", "bin")
	}
	if opt.HTTP == nil {
		opt.HTTP = &http.Client{Timeout: 5 * time.Minute}
	}

	ve, pa, err := reg.Lookup(opt.Scanner, opt.Version, opt.Platform)
	if err != nil {
		return nil, err
	}
	hosts := opt.AllowedHosts
	if hosts == nil {
		hosts = allowedHosts
	}
	if err := checkHostAllowlistWith(pa.URL, hosts); err != nil {
		return nil, err
	}

	tmpFile, actualSHA, err := downloadToTemp(opt.HTTP, pa.URL, opt.Progress)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(tmpFile) }()

	if !strings.EqualFold(actualSHA, pa.ArchiveSHA256) {
		return nil, fmt.Errorf("archive sha256 mismatch for %s@%s (%s)\n  expected: %s\n  actual:   %s\n  url:      %s\n  refusing to install",
			opt.Scanner, opt.Version, opt.Platform, pa.ArchiveSHA256, actualSHA, pa.URL)
	}

	destPath := filepath.Join(opt.DestDir, ve.Binary)
	if err := ExtractBinary(tmpFile, ve.Archive, ve.Binary, destPath); err != nil {
		return nil, fmt.Errorf("extract %s: %w", opt.Scanner, err)
	}
	binarySHA, err := hashPath(destPath)
	if err != nil {
		return nil, fmt.Errorf("hash extracted %s: %w", destPath, err)
	}
	if !strings.EqualFold(binarySHA, pa.BinarySHA256) {
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("binary sha256 mismatch for %s@%s (%s)\n  expected: %s\n  actual:   %s\n  path:     %s\n  refusing to install — the archive pin and binary pin are out of sync",
			opt.Scanner, opt.Version, opt.Platform, pa.BinarySHA256, binarySHA, destPath)
	}

	return &Result{
		Scanner:       opt.Scanner,
		Version:       opt.Version,
		Platform:      opt.Platform,
		InstalledAt:   destPath,
		ArchiveSHA256: actualSHA,
		BinarySHA256:  binarySHA,
		URL:           pa.URL,
	}, nil
}

func hashPath(path string) (string, error) {
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

func downloadToTemp(client *http.Client, rawURL string, progress io.Writer) (string, string, error) {
	if progress != nil {
		_, _ = fmt.Fprintf(progress, "  → downloading %s\n", rawURL)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("parse url %q: %w", rawURL, err)
	}
	if u.Scheme != "https" {
		return "", "", fmt.Errorf("refusing non-HTTPS download url: %s", rawURL)
	}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("http get %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return "", "", fmt.Errorf("http %s: %s", rawURL, resp.Status)
	}

	tmp, err := os.CreateTemp("", "lintel-install-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp: %w", err)
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		_ = tmp.Close()
		return "", "", fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", "", err
	}
	return tmp.Name(), hex.EncodeToString(h.Sum(nil)), nil
}
