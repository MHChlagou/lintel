// Package detect identifies project stacks from manifest files.
package detect

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Stacks identified by Aegis; kept as strings for flexibility with config.
const (
	Npm      = "npm"
	Yarn     = "yarn"
	Pnpm     = "pnpm"
	Maven    = "maven"
	Gradle   = "gradle"
	Pip      = "pip"
	Poetry   = "poetry"
	Go       = "go"
	Cargo    = "cargo"
	Composer = "composer"
	Bundler  = "bundler"
	Mix      = "mix"
)

type ProjectContext struct {
	RepoRoot    string
	Stacks      []string
	StagedFiles []string
}

// Detect runs the three-step resolution: explicit list → manifest scan → extension fallback.
func Detect(repoRoot string, explicit []string, excludeGlobs []string, stagedFiles []string) (*ProjectContext, error) {
	if len(explicit) > 0 {
		return &ProjectContext{RepoRoot: repoRoot, Stacks: dedupe(explicit), StagedFiles: stagedFiles}, nil
	}
	stacks, err := scanManifests(repoRoot, excludeGlobs)
	if err != nil {
		return nil, err
	}
	if len(stacks) == 0 {
		stacks = extensionFallback(stagedFiles)
	}
	return &ProjectContext{RepoRoot: repoRoot, Stacks: stacks, StagedFiles: stagedFiles}, nil
}

func scanManifests(root string, excludeGlobs []string) ([]string, error) {
	seen := map[string]bool{}
	hasPkgJSON := false
	var pkgJSONDir string
	var lockFiles = map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "vendor" || base == "dist" || base == "build" || base == ".aegis" {
				return filepath.SkipDir
			}
			if matchAny(rel, excludeGlobs) {
				return filepath.SkipDir
			}
			return nil
		}
		if matchAny(rel, excludeGlobs) {
			return nil
		}
		name := d.Name()
		switch name {
		case "package.json":
			hasPkgJSON = true
			if pkgJSONDir == "" {
				pkgJSONDir = filepath.Dir(path)
			}
		case "package-lock.json", "yarn.lock", "pnpm-lock.yaml":
			lockFiles[name] = true
		case "pom.xml":
			seen[Maven] = true
		case "build.gradle", "build.gradle.kts":
			seen[Gradle] = true
		case "requirements.txt", "Pipfile":
			seen[Pip] = true
		case "pyproject.toml":
			// Distinguishing poetry vs pip requires reading the file; default to pip.
			seen[Pip] = true
		case "go.mod":
			seen[Go] = true
		case "Cargo.toml":
			seen[Cargo] = true
		case "composer.json":
			seen[Composer] = true
		case "Gemfile":
			seen[Bundler] = true
		case "mix.exs":
			seen[Mix] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if hasPkgJSON {
		switch {
		case lockFiles["pnpm-lock.yaml"]:
			seen[Pnpm] = true
		case lockFiles["yarn.lock"]:
			seen[Yarn] = true
		default:
			seen[Npm] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func extensionFallback(files []string) []string {
	counts := map[string]int{}
	for _, f := range files {
		switch strings.ToLower(filepath.Ext(f)) {
		case ".js", ".jsx", ".ts", ".tsx":
			counts[Npm]++
		case ".py":
			counts[Pip]++
		case ".go":
			counts[Go]++
		case ".rs":
			counts[Cargo]++
		case ".rb":
			counts[Bundler]++
		case ".php":
			counts[Composer]++
		case ".java":
			counts[Maven]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	type kv struct {
		k string
		v int
	}
	ranked := make([]kv, 0, len(counts))
	for k, v := range counts {
		ranked = append(ranked, kv{k, v})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].v > ranked[j].v })
	return []string{ranked[0].k}
}

func matchAny(path string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := doublestar.PathMatch(p, path); ok {
			return true
		}
	}
	return false
}

func dedupe(xs []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
