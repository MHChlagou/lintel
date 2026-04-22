# Adding a scanner

Adding a new scanner adapter is a focused, well-bounded contribution — typically ~200 lines of Go, a testdata blob, a config entry, and a docs page. This walkthrough shows the exact steps.

## 1. Decide the fit

Before writing code, answer:

- Which `check` does this scanner serve? One of `secrets`, `malicious_code`, `dependencies`, `lint`, `format`. If none fits, open a design discussion first.
- Which `stack`(s) does it serve? `go`, `npm`, `python`, `shell`, or stack-agnostic.
- Does an existing adapter already cover these combinations? Overlapping scanners can coexist (users pick one via `checks.<check>.scanners.<stack>`) but we prefer curation over proliferation.

## 2. Implement the `Checker` interface

```go
// internal/checker/myscanner.go
package checker

type myScanner struct{ base }

func (m *myScanner) Name() string { return "myscanner" }

func (m *myScanner) Supports(stack string) bool { return stack == "npm" }

func (m *myScanner) Run(ctx context.Context, files []string, opts RunOpts) ([]finding.Finding, error) {
    cmd := exec.CommandContext(ctx, m.Path(), append([]string{"--format=json"}, files...)...)
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("myscanner: %w", err)
    }
    return m.normalize(out)
}

func (m *myScanner) normalize(raw []byte) ([]finding.Finding, error) {
    // parse JSON → []finding.Finding; compute fingerprints
}
```

Three rules:

1. **`normalize` must be deterministic.** Same input → same output, same order.
2. **Fingerprints must be stable** across cosmetic code movement. Include `scanner/rule/file/line/<normalized-message>` but not timestamps or absolute paths.
3. **Severity mapping must be documented** in `docs/docs/scanners/myscanner.md`.

## 3. Register the adapter

```go
// internal/checker/registry.go
func init() {
    register(&myScanner{})
}
```

The registry is the single source of truth for "what scanners does Aegis know about."

## 4. Pin SHA256 hashes

```go
// internal/config/defaults_spec.go
var scannerDefaults = map[string]ScannerPin{
    // …
    "myscanner": {
        Version: "1.2.3",
        SHA256: map[string]string{
            "linux/amd64":  "…",
            "linux/arm64":  "…",
            "darwin/amd64": "…",
            "darwin/arm64": "…",
            "windows/amd64": "…",
        },
    },
}
```

Get these hashes from the upstream release — verify the upstream's own signature first. A PR that adds a scanner without valid per-platform pins will be requested to complete them before merge.

## 5. Write a normalization test

```go
// internal/checker/myscanner_test.go
func TestMyScannerNormalize(t *testing.T) {
    raw, _ := os.ReadFile("../../testdata/myscanner-sample.json")
    got, err := (&myScanner{}).normalize(raw)
    if err != nil { t.Fatal(err) }
    want := []finding.Finding{ /* … */ }
    if diff := cmp.Diff(want, got); diff != "" {
        t.Fatalf("diff: %s", diff)
    }
}
```

Place the sample blob under `testdata/myscanner-sample.json`. Realistic fixtures are better than minimal ones — include edge cases (multi-line messages, missing optional fields, unusual paths).

## 6. Add a docs page

Create `docs/docs/scanners/myscanner.md` following the pattern of the existing pages — upstream link, invocation, severity mapping, configuration, and common pitfalls.

Add the page to `docs/mkdocs.yml` under `nav:`.

## 7. Update `README.md` and `CHANGELOG.md`

- Add the scanner to the list in `README.md`.
- Add an `### Added` entry in `CHANGELOG.md` under `[Unreleased]`.

## 8. Run the full CI gate locally

```bash
make ci
```

This runs fmt, vet, lint, tests, and the build matrix.

## 9. Open the PR

Follow the [PR template](https://github.com/aegis-sec/aegis/blob/main/.github/PULL_REQUEST_TEMPLATE.md). Expect a careful review from CODEOWNERS — especially of the hash pins and the normalize function.
