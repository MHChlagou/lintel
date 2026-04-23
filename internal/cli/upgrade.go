package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MHChlagou/aegis/internal/version"
)

// The notifier hits the public GitHub Releases API (no auth, 60 req/h per IP).
// It never modifies the installed binary — by design. Self-replacement in a
// security-verifier tool creates a trust-surface larger than the UX win.
const (
	latestReleaseURL = "https://api.github.com/repos/MHChlagou/aegis/releases/latest"
	releaseAssetBase = "https://github.com/MHChlagou/aegis/releases/download"
)

// ghRelease mirrors the subset of the GitHub Releases API we care about.
type ghRelease struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
}

func cmdUpgrade() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Check for a newer aegis release and print the upgrade command",
		Long: `Compares the running aegis version against the latest GitHub release.
If a newer version exists, prints the release notes and a copy-paste
upgrade command for your OS and architecture. This command never
replaces the binary itself — self-updating verifiers are a larger
trust surface than they're worth for a security tool.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := &http.Client{Timeout: 15 * time.Second}
			rel, err := fetchLatestRelease(client, latestReleaseURL)
			if err != nil {
				return err
			}
			return renderUpgradeNotice(cmd.OutOrStdout(), version.Version, rel, runtime.GOOS, runtime.GOARCH)
		},
	}
}

func fetchLatestRelease(client *http.Client, url string) (ghRelease, error) {
	var zero ghRelease
	resp, err := client.Get(url)
	if err != nil {
		return zero, fmt.Errorf("check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return zero, fmt.Errorf("check for updates: GitHub API returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return zero, fmt.Errorf("parse GitHub API response: %w", err)
	}
	if rel.TagName == "" {
		return zero, fmt.Errorf("GitHub API response missing tag_name")
	}
	return rel, nil
}

// renderUpgradeNotice formats the human-readable output. Factored out from
// cmdUpgrade so tests can exercise it with a synthetic release payload and
// a specific OS/arch without hitting the network.
func renderUpgradeNotice(w io.Writer, currentVersion string, rel ghRelease, goos, goarch string) error {
	fpf(w, "current: %s\nlatest:  %s", currentVersion, rel.TagName)
	if date := formatReleaseDate(rel.PublishedAt); date != "" {
		fpf(w, "  (released %s)", date)
	}
	fpf(w, "\n\n")

	cmp := compareSemver(currentVersion, rel.TagName)
	switch {
	case cmp == 0:
		fpf(w, "You are on the latest release.\n")
		return nil
	case cmp > 0:
		fpf(w, "You are running ahead of the latest release (likely a dev build).\n")
		return nil
	}

	if body := strings.TrimSpace(rel.Body); body != "" {
		fpf(w, "Release notes:\n%s\n\n", indentBlock(body, "  "))
	}
	fpf(w, "To upgrade, run:\n\n%s\n", upgradeCommand(rel.TagName, goos, goarch))
	if rel.HTMLURL != "" {
		fpf(w, "\nRelease page: %s\n", rel.HTMLURL)
	}
	return nil
}

// compareSemver returns -1 / 0 / +1 when a < b, a == b, a > b. Follows the
// semver rule that a pre-release version (0.1.0-dev, 0.2.0-rc1) ranks
// strictly below the matching release (0.1.0, 0.2.0). This matters because
// locally-built dev binaries carry "-dev" and should be told to upgrade to
// the tagged release, not greeted with "you're on the latest."
func compareSemver(a, b string) int {
	na, sa := parseSemver(a)
	nb, sb := parseSemver(b)
	for i := 0; i < 3; i++ {
		switch {
		case na[i] < nb[i]:
			return -1
		case na[i] > nb[i]:
			return 1
		}
	}
	// X.Y.Z identical — pre-release loses to release. Beyond that we fall
	// back to lexicographic suffix compare, which is correct for simple
	// forms ("-dev" < "-rc1" < "-rc2") and adequate for our release tags.
	switch {
	case sa != "" && sb == "":
		return -1
	case sa == "" && sb != "":
		return 1
	case sa < sb:
		return -1
	case sa > sb:
		return 1
	}
	return 0
}

// parseSemver returns the (major, minor, patch) triple and any pre-release
// suffix. Leading "v" is tolerated; missing components default to 0.
func parseSemver(s string) ([3]int, string) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	suffix := ""
	for i, c := range s {
		if c == '-' || c == '+' {
			suffix = s[i+1:]
			s = s[:i]
			break
		}
	}
	var out [3]int
	for i, part := range strings.SplitN(s, ".", 3) {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(part)
		out[i] = n
	}
	return out, suffix
}

// upgradeCommand returns the recommended upgrade shell snippet for the
// given platform. Linux/macOS get a curl one-liner; Windows gets a
// PowerShell Invoke-WebRequest. The asset naming mirrors release.yml.
func upgradeCommand(tag, goos, goarch string) string {
	asset := fmt.Sprintf("aegis-%s-%s", goos, goarch)
	if goos == "windows" {
		asset += ".exe"
		return fmt.Sprintf(`  # PowerShell
  Invoke-WebRequest \
    -Uri %s/%s/%s \
    -OutFile $env:USERPROFILE\bin\aegis.exe`, releaseAssetBase, tag, asset)
	}
	return fmt.Sprintf(`  curl -fsSL %s/%s/%s -o /usr/local/bin/aegis
  chmod +x /usr/local/bin/aegis`, releaseAssetBase, tag, asset)
}

func indentBlock(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

func formatReleaseDate(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("2006-01-02")
}
