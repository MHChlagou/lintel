package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.0", "v0.2.0", -1},
		{"0.2.0", "v0.2.0", 0}, // tolerant of "v" prefix
		{"v1.0.0", "v0.9.9", 1},
		{"0.1.0-dev", "0.1.0", -1}, // pre-release ranks BELOW release
		{"v0.2.0-dev", "v0.2.0", -1},
		{"v0.2.0", "v0.2.0-dev", 1},
		{"v0.3.0-dev", "v0.2.0", 1},      // higher major.minor.patch wins regardless of suffix
		{"v0.2.0-dev", "v0.2.0-rc1", -1}, // suffix lex order among pre-releases
		{"v1.2.3", "v1.2.3", 0},
		{"v2", "v1.9.9", 1}, // missing components default to 0
		{"", "v0.1.0", -1},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestUpgradeCommand_Linux(t *testing.T) {
	got := upgradeCommand("v0.3.0", "linux", "amd64")
	for _, want := range []string{
		"curl -fsSL",
		"releases/download/v0.3.0/aegis-linux-amd64",
		"-o /usr/local/bin/aegis",
		"chmod +x",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("linux upgrade command missing %q:\n%s", want, got)
		}
	}
}

func TestUpgradeCommand_Windows(t *testing.T) {
	got := upgradeCommand("v0.3.0", "windows", "amd64")
	for _, want := range []string{
		"Invoke-WebRequest",
		"releases/download/v0.3.0/aegis-windows-amd64.exe",
		"$env:USERPROFILE",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("windows upgrade command missing %q:\n%s", want, got)
		}
	}
}

func TestRenderUpgradeNotice_NewerAvailable(t *testing.T) {
	rel := ghRelease{
		TagName:     "v0.3.0",
		PublishedAt: "2026-05-14T12:00:00Z",
		Body:        "### Added\n- new widget",
		HTMLURL:     "https://github.com/MHChlagou/aegis/releases/tag/v0.3.0",
	}
	var buf bytes.Buffer
	if err := renderUpgradeNotice(&buf, "v0.2.0", rel, "linux", "amd64"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"current: v0.2.0",
		"latest:  v0.3.0",
		"released 2026-05-14",
		"new widget",
		"curl -fsSL",
		"aegis-linux-amd64",
		"Release page:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderUpgradeNotice_AlreadyLatest(t *testing.T) {
	rel := ghRelease{TagName: "v0.2.0", PublishedAt: "2026-04-23T10:00:00Z"}
	var buf bytes.Buffer
	if err := renderUpgradeNotice(&buf, "v0.2.0", rel, "linux", "amd64"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "latest release") {
		t.Errorf("expected on-latest message, got:\n%s", out)
	}
	if strings.Contains(out, "curl") {
		t.Errorf("unexpected upgrade command in on-latest output:\n%s", out)
	}
}

func TestRenderUpgradeNotice_AheadOfLatest(t *testing.T) {
	// Local dev build ahead of the released tag — don't instruct a downgrade.
	rel := ghRelease{TagName: "v0.1.0"}
	var buf bytes.Buffer
	if err := renderUpgradeNotice(&buf, "v0.2.0-dev", rel, "linux", "amd64"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ahead of") {
		t.Errorf("expected ahead-of-release message, got:\n%s", out)
	}
}

func TestFetchLatestRelease_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{
			TagName:     "v0.3.0",
			PublishedAt: "2026-05-14T12:00:00Z",
			Body:        "notes",
			HTMLURL:     "https://example.test/release",
		})
	}))
	defer srv.Close()

	rel, err := fetchLatestRelease(srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if rel.TagName != "v0.3.0" {
		t.Errorf("tag=%q, want v0.3.0", rel.TagName)
	}
}

func TestFetchLatestRelease_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchLatestRelease(srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}
