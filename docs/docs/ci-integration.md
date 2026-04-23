# CI integration

Lintel is the same binary on a developer's laptop and in CI. Running it in CI catches two things the local hook cannot: people who bypassed the hook, and environmental drift between developer machines and the production build.

## GitHub Actions

A minimal job that installs Lintel, restores the configured scanners, and runs `lintel run` on every pull request:

```yaml
name: Lintel

on:
  pull_request:

permissions:
  contents: read

jobs:
  lintel:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0        # lintel needs history for pre-push-style scans

      - name: Install lintel
        run: |
          version="0.1.0"
          curl -fsSL "https://github.com/MHChlagou/lintel/releases/download/v${version}/lintel-linux-amd64" -o /usr/local/bin/lintel
          curl -fsSL "https://github.com/MHChlagou/lintel/releases/download/v${version}/lintel-linux-amd64.sha256" -o /tmp/lintel.sha256
          ( cd /usr/local/bin && sha256sum -c /tmp/lintel.sha256 )
          chmod +x /usr/local/bin/lintel

      - name: Install scanners
        # Install only the ones your lintel.yaml references.
        run: |
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.61.0
          # ... add gitleaks, osv-scanner, etc.

      - name: Verify scanner hashes
        run: lintel doctor

      - name: Run lintel
        run: lintel run --output json | tee lintel-report.json

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: lintel-report
          path: lintel-report.json
```

Pin your release version and verify its checksum - do not `curl | sh` without verification. Once SARIF output lands (v1.1), you can upload findings to GitHub Code Scanning directly.

## GitLab CI

```yaml
lintel:
  image: alpine:3.20
  stage: test
  before_script:
    - apk add --no-cache curl git
    - curl -fsSL https://github.com/MHChlagou/lintel/releases/download/v0.1.0/lintel-linux-amd64 -o /usr/local/bin/lintel
    - chmod +x /usr/local/bin/lintel
  script:
    - lintel doctor
    - lintel run --output json > lintel-report.json
  artifacts:
    when: always
    paths: [lintel-report.json]
```

## CircleCI

```yaml
version: 2.1

jobs:
  lintel:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install lintel
          command: |
            curl -fsSL https://github.com/MHChlagou/lintel/releases/download/v0.1.0/lintel-linux-amd64 -o ~/bin/lintel
            chmod +x ~/bin/lintel
      - run: lintel doctor
      - run: lintel run --output json | tee lintel-report.json
      - store_artifacts:
          path: lintel-report.json

workflows:
  version: 2
  main:
    jobs: [lintel]
```

## What to gate in CI

Three patterns, from strict to permissive:

| Pattern                 | Behavior                                                           |
| ----------------------- | ------------------------------------------------------------------ |
| **Block the PR**        | `lintel run` as a required status check. Default gate fails the job on any error. |
| **Report and warn**     | Use `--output json`, parse the counts in a comment-bot, but do not fail the build. Useful during rollout. |
| **Baseline-guarded**    | Fail only on **new** findings relative to `main`'s baseline.       |

The baseline-guarded pattern is the usual long-term setup for an established repo:

```bash
lintel run --output json > /tmp/pr.json
lintel baseline --check    # exits 1 if baseline would grow
```

## Secrets in CI logs

`lintel run` never prints matched secret values by default; it prints the rule, file, and redacted fingerprint. With `--verbose` that still holds. If you are uncertain, pipe output through a masker that redacts any token-shaped strings.

## Caching

There is no built-in cache for scanner binaries in v1.0. Cache them yourself at the CI layer if you want to avoid re-downloading on every run - the Go/Node language-specific caches work as usual.
