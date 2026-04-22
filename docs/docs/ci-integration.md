# CI integration

Aegis is the same binary on a developer's laptop and in CI. Running it in CI catches two things the local hook cannot: people who bypassed the hook, and environmental drift between developer machines and the production build.

## GitHub Actions

A minimal job that installs Aegis, restores the configured scanners, and runs `aegis run` on every pull request:

```yaml
name: Aegis

on:
  pull_request:

permissions:
  contents: read

jobs:
  aegis:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0        # aegis needs history for pre-push-style scans

      - name: Install aegis
        run: |
          version="0.1.0"
          curl -fsSL "https://github.com/aegis-sec/aegis/releases/download/v${version}/aegis-linux-amd64" -o /usr/local/bin/aegis
          curl -fsSL "https://github.com/aegis-sec/aegis/releases/download/v${version}/aegis-linux-amd64.sha256" -o /tmp/aegis.sha256
          ( cd /usr/local/bin && sha256sum -c /tmp/aegis.sha256 )
          chmod +x /usr/local/bin/aegis

      - name: Install scanners
        # Install only the ones your aegis.yaml references.
        run: |
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.61.0
          # ... add gitleaks, osv-scanner, etc.

      - name: Verify scanner hashes
        run: aegis doctor

      - name: Run aegis
        run: aegis run --output json | tee aegis-report.json

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: aegis-report
          path: aegis-report.json
```

Pin your release version and verify its checksum — do not `curl | sh` without verification. Once SARIF output lands (v1.1), you can upload findings to GitHub Code Scanning directly.

## GitLab CI

```yaml
aegis:
  image: alpine:3.20
  stage: test
  before_script:
    - apk add --no-cache curl git
    - curl -fsSL https://github.com/aegis-sec/aegis/releases/download/v0.1.0/aegis-linux-amd64 -o /usr/local/bin/aegis
    - chmod +x /usr/local/bin/aegis
  script:
    - aegis doctor
    - aegis run --output json > aegis-report.json
  artifacts:
    when: always
    paths: [aegis-report.json]
```

## CircleCI

```yaml
version: 2.1

jobs:
  aegis:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install aegis
          command: |
            curl -fsSL https://github.com/aegis-sec/aegis/releases/download/v0.1.0/aegis-linux-amd64 -o ~/bin/aegis
            chmod +x ~/bin/aegis
      - run: aegis doctor
      - run: aegis run --output json | tee aegis-report.json
      - store_artifacts:
          path: aegis-report.json

workflows:
  version: 2
  main:
    jobs: [aegis]
```

## What to gate in CI

Three patterns, from strict to permissive:

| Pattern                 | Behavior                                                           |
| ----------------------- | ------------------------------------------------------------------ |
| **Block the PR**        | `aegis run` as a required status check. Default gate fails the job on any error. |
| **Report and warn**     | Use `--output json`, parse the counts in a comment-bot, but do not fail the build. Useful during rollout. |
| **Baseline-guarded**    | Fail only on **new** findings relative to `main`'s baseline.       |

The baseline-guarded pattern is the usual long-term setup for an established repo:

```bash
aegis run --output json > /tmp/pr.json
aegis baseline --check    # exits 1 if baseline would grow
```

## Secrets in CI logs

`aegis run` never prints matched secret values by default; it prints the rule, file, and redacted fingerprint. With `--verbose` that still holds. If you are uncertain, pipe output through a masker that redacts any token-shaped strings.

## Caching

There is no built-in cache for scanner binaries in v1.0. Cache them yourself at the CI layer if you want to avoid re-downloading on every run — the Go/Node language-specific caches work as usual.
