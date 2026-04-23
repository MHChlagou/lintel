# Contributing to Lintel

Thanks for your interest in making Lintel better. This document describes how
to set up your environment, propose changes, and ship them.

By participating, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

---

## Ways to contribute

- **File a bug.** Open an issue using the **Bug report** template. Reproduction
  steps, OS, Go version, and `lintel version` output are the minimum.
- **Request a feature.** Use the **Feature request** template. If it fits our
  published roadmap (see [`spec.md`](docs/docs/reference/spec.md) §21), we'll label it
  `roadmap`; otherwise expect a design discussion first.
- **Report a vulnerability.** Do **not** open a public issue - follow
  [`SECURITY.md`](SECURITY.md).
- **Submit a pull request.** See below.

---

## Development setup

You need Go `1.25+` and `make`. Optional but helpful: `golangci-lint`,
`govulncheck`.

```bash
git clone https://github.com/MHChlagou/lintel.git
cd lintel
make ci        # fmt + vet + lint + test + build - the same gates CI runs
```

Useful targets (see [`Makefile`](Makefile) for the full list):

| target        | purpose                                                   |
| ------------- | --------------------------------------------------------- |
| `make build`  | Build the `lintel` binary into `bin/`                      |
| `make test`   | `go test -race ./...`                                     |
| `make lint`   | `gofmt -l .` + `golangci-lint run`                        |
| `make smoke`  | End-to-end smoke test (init → run) against the built binary |
| `make ci`     | Everything CI runs, locally                                |

---

## Pull request process

1. **Open an issue first** for anything non-trivial. This saves you from
   building something we can't merge. Bug fixes and docs fixes can skip
   this.
2. **Branch from `main`**: `git checkout -b feat/<short-topic>`.
3. **Keep PRs focused.** One logical change per PR. If you discover a separate
   bug, fix it in a follow-up PR.
4. **Write a test.** New features need unit tests. Bug fixes need a regression
   test that fails without the fix.
5. **Use Conventional Commits** for the PR title - the `pr-checks.yml`
   workflow enforces this. Allowed types:

   ```
   feat  fix  docs  style  refactor  perf  test  build  ci  chore  revert
   ```

   Examples: `feat(runner): parallelize scanners per stack`, `fix(gate): ignore WARN when severity is low`.
6. **Sign off your commits** (`git commit -s`) to agree to the
   [Developer Certificate of Origin](https://developercertificate.org/). A
   signed-off-by line is required.
7. **Keep `make ci` green** locally before pushing. CI will block the merge
   otherwise.
8. **Respond to review within ~1 week.** We'll close stale PRs after 30 days
   of inactivity; a gentle nudge is always welcome to reopen.

### We merge by **squash + conventional title**

The squashed commit message becomes a line in `CHANGELOG.md` at release time,
so the PR title matters. Keep it short (≤ 70 chars) and declarative.

---

## Code style

- **Go**: `gofmt` + `golangci-lint` ([`.golangci.yml`](.golangci.yml)). Export
  docs for public symbols; error messages start with a lowercase verb.
- **YAML**: 2-space indent, no trailing whitespace.
- **Markdown**: 80-char soft wrap, fenced code blocks with a language tag.
- **Commit messages**: Conventional Commits, imperative mood, body wrapped at 72.

---

## Adding a new scanner adapter

Scanner adapters live in `internal/checker/`. A minimal adapter must:

1. Implement the `Checker` interface from `internal/checker/checker.go`.
2. Register itself in `internal/checker/registry.go`.
3. Be accompanied by a unit test that feeds canned scanner output through the
   `Normalize` path and asserts a stable `[]Finding` slice.
4. Have an entry in `internal/config/defaults_spec.go` under the appropriate
   stack, with **pinned SHA256s** for the upstream binary releases on each
   supported platform. Do not wire an adapter without hashes - Lintel refuses to
   execute unverified binaries.
5. Have a docs page under `docs/docs/scanners/`.

---

## Security

This is a security tool. A bug in Lintel can mask real vulnerabilities in
downstream repositories. Be especially careful with:

- **Supply-chain paths** (`internal/resolve`, `internal/checker/*/resolve`): do
  not loosen SHA256 checks, do not accept relative binary paths, do not let the
  `strict_versions` bypass flow silently.
- **Override / audit logging** (`internal/cli/run.go`, `.lintel/overrides.log`):
  every bypass must be recorded with a mandatory reason.
- **Exit codes**: changes must be reflected in `spec.md` §11.5.

These files are listed in [`CODEOWNERS`](.github/CODEOWNERS) - review from a
code owner is mandatory before merging.

---

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).
