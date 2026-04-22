# Release process

Maintainer runbook for cutting a new Aegis release.

## Cadence

- **Patch releases** (`0.1.x`) as needed for bug fixes, ideally within a week of the fix landing on `main`.
- **Minor releases** (`0.x.0`) roughly monthly once v1.0 is shipped.
- **Major releases** aligned with the roadmap in [`spec.md` §21](reference/spec.md#21-roadmap).

## Release checklist

### Prepare

- [ ] All desired PRs merged to `main`.
- [ ] CI green on `main`.
- [ ] `CHANGELOG.md` has a populated `[Unreleased]` section. Move it to `[X.Y.Z] - YYYY-MM-DD` and update the comparison links at the bottom.
- [ ] If any scanner pin changed, confirm all platforms have hashes.
- [ ] Run `make ci` locally as a belt-and-braces check.

### Tag

```bash
git checkout main
git pull
git tag -a v0.1.0 -m "Aegis 0.1.0"
git push origin v0.1.0
```

Tag names **must** match `vX.Y.Z`. The release workflow's trigger is `tags: ['v*']`.

### Build and sign

The `release.yml` workflow runs on tag push and:

1. Cross-compiles the binary for every supported platform.
2. Computes SHA256 for each artifact.
3. Signs each artifact with Sigstore keyless signing, producing `.sig` and `.sig.crt` alongside the binary.
4. Uploads artifacts + checksums + signatures to the GitHub Release.

Verify the workflow completed and all artifacts are attached before announcing.

### Verify the release

```bash
# Download and verify one platform as a sanity check.
version=0.1.0
curl -fsSLO "https://github.com/aegis-sec/aegis/releases/download/v${version}/aegis-linux-amd64"
curl -fsSLO "https://github.com/aegis-sec/aegis/releases/download/v${version}/aegis-linux-amd64.sha256"
sha256sum -c aegis-linux-amd64.sha256

# Sigstore verification.
cosign verify-blob \
  --certificate aegis-linux-amd64.sig.crt \
  --signature  aegis-linux-amd64.sig \
  --certificate-identity-regexp 'https://github.com/aegis-sec/aegis/.github/workflows/release.yml@.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  aegis-linux-amd64
```

### Publish

- [ ] GitHub Release is published (not draft).
- [ ] Changelog link in Release body points at the right `CHANGELOG.md` anchor.
- [ ] Docs site `mike` deploys the new version and updates the `latest` alias.
- [ ] Announce in Discussions with a short summary of highlights.

## Yanking a bad release

If a release needs to be pulled:

1. Add a **yank notice** to the top of the GitHub Release body. Do not delete the release — downstreams may already have it pinned.
2. Open a tracking issue explaining what was wrong and which version to use instead.
3. Cut a patch release with the fix.
4. Leave the tag in place. Git tag deletion is a supply-chain footgun.

## Docs versioning

Aegis uses [mike](https://github.com/jimporter/mike) for docs versioning. The `docs.yml` workflow calls `mike deploy --push <version> latest` on each tagged release. Users land on `/latest/` by default; `/<version>/` paths remain addressable forever.
