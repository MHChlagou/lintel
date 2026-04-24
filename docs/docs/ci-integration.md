# CI integration

Lintel is the same binary on a developer's laptop and in CI. Running it in CI catches two things the local hook cannot: people who bypassed the hook, and environmental drift between developer machines and the production build.

## GitHub Actions

### Recommended: the `lintel-action`

The [**`MHChlagou/lintel-action`**](https://github.com/MHChlagou/lintel-action) composite action installs lintel, fetches the scanner binaries declared in your `lintel.yaml`, runs the gate, and writes SARIF ready to hand off to GitHub Advanced Security.

```yaml
name: security
on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  security-events: write   # required for upload-sarif

jobs:
  lintel:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0         # pre-push hook scans history

      - uses: MHChlagou/lintel-action@v1
        with:
          version: v0.2.4        # pin a lintel release for reproducible CI

      - uses: github/codeql-action/upload-sarif@v3
        if: always()             # always upload, even when lintel blocked
        with:
          sarif_file: lintel.sarif
```

Findings then appear as **inline annotations on the PR** and in **Security → Code scanning**. The action caches scanner binaries in `~/.lintel/bin` keyed on lintel version + `lintel.yaml` hash, so only the first run pays the download cost.

Useful inputs:

| Input | Default | Purpose |
| --- | --- | --- |
| `version` | `latest` | Lintel release tag. Pin a version for reproducible CI. |
| `hook` | `pre-push` | `pre-commit` \| `pre-push` \| `commit-msg` |
| `output` | `sarif` | `pretty` \| `json` \| `sarif` |
| `fail-on-findings` | `true` | Set `false` to only annotate, never fail the job |

Full input / output reference: [the action's README](https://github.com/MHChlagou/lintel-action#inputs).

### Alternative: install the CLI by hand

If you'd rather not depend on a third-party action, the install script works in CI exactly the same way it works on your laptop:

```yaml
name: Lintel
on:
  pull_request:

permissions:
  contents: read
  security-events: write

jobs:
  lintel:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }

      - name: Install lintel
        run: |
          curl -fsSL https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.sh \
            | sh -s -- --version v0.2.4

      - name: Install scanners from lintel.yaml
        run: lintel install --all

      - name: Run lintel
        run: lintel run --hook pre-push --output sarif > lintel.sarif

      - uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: lintel.sarif
```

The install script verifies the SHA256 against the release sidecar and, when `cosign` is on `PATH`, the Sigstore bundle. Pin a version (`--version vX.Y.Z`) for reproducible CI; `latest` is fine for experimentation.

## GitLab CI

```yaml
lintel:
  image: alpine:3.20
  stage: test
  before_script:
    - apk add --no-cache curl git
    - curl -fsSL https://github.com/MHChlagou/lintel/releases/download/v0.2.4/lintel-linux-amd64 -o /usr/local/bin/lintel
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
            curl -fsSL https://github.com/MHChlagou/lintel/releases/download/v0.2.4/lintel-linux-amd64 -o ~/bin/lintel
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
