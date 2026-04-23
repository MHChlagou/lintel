<!--
Thanks for contributing to Lintel! Please fill out the sections below.
Your PR title must follow Conventional Commits - it is enforced by CI.
Example: feat(runner): parallelize scanners per stack
-->

## What does this change?

<!-- A short, focused description of the change. Link the related issue: -->
Closes #

## Why is it needed?

<!-- What problem does it solve? What is the user-visible effect? -->

## How was it tested?

- [ ] `make ci` passes locally
- [ ] Unit tests added or updated (please describe)
- [ ] Manual verification steps (if relevant):

## Checklist

- [ ] PR title follows Conventional Commits
- [ ] Commits are signed off (`git commit -s`)
- [ ] Documentation updated (`docs/`, `README.md`, `CHANGELOG.md` if user-visible)
- [ ] For new scanner adapters: SHA256 hashes pinned in `internal/config/defaults_spec.go`
- [ ] For changes to exit codes or CLI surface: `spec.md` updated
- [ ] No secrets, keys, or personal data in the diff

## Screenshots or terminal output (optional)

<!-- If the change affects user-facing output, paste a before/after here. -->
