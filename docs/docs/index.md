---
title: Aegis
hide:
  - navigation
  - toc
---

# Aegis

*The shield your commits pass through.*

A single-binary, shift-left security orchestrator that runs as a Git hook and in CI. One declarative spec file, zero runtime dependencies, and the best-in-class OSS scanners you already trust.

[Get started in 5 minutes :material-arrow-right:](getting-started.md){ .md-button .md-button--primary }
[View on GitHub :fontawesome-brands-github:](https://github.com/aegis-sec/aegis){ .md-button }

---

## Why Aegis

Most teams want shift-left security — secrets, SAST, SCA, lint, format — on every commit. The existing landscape forces a choice.

=== "Glue it yourself"

    Husky + lint-staged + *N* tools, each with its own config. Brittle, drifts between repos, breaks when anyone upgrades anything.

=== "Adopt a SaaS"

    Expensive, opaque, and almost always a black box running rules you can't audit.

=== "Aegis"

    One Go binary that coordinates the scanners **you** choose, driven by one YAML file, with a tight CLI and strong supply-chain hygiene built in.

---

## Feature highlights

<div class="grid cards" markdown>

-   :material-package-variant-closed:{ .lg .middle } **Single static binary**

    ---

    Linux, macOS, Windows on amd64 and arm64. No runtime dependencies.
    Download, put on `$PATH`, go.

-   :material-file-document-outline:{ .lg .middle } **Declarative config**

    ---

    One `aegis.yaml` describes stacks, scanners, gates, and ignores.
    Checked in, versioned, reviewed like any other code.

-   :material-shield-lock-outline:{ .lg .middle } **Supply-chain hygiene**

    ---

    Scanner binaries are resolved and verified against pinned SHA256 hashes
    on every invocation. No hash, no run.

-   :material-source-branch:{ .lg .middle } **Git-hook native**

    ---

    `aegis install` manages `pre-commit` and `pre-push` hooks without
    clobbering foreign ones. Uninstall is clean.

-   :material-speedometer:{ .lg .middle } **Fast by default**

    ---

    Parallel execution per scanner with per-check and total timeouts.
    Only staged files are scanned on pre-commit.

-   :material-chart-line:{ .lg .middle } **Deterministic output**

    ---

    Pretty or JSON. Sorted, stable, and CI-friendly. SARIF and JUnit on
    the roadmap for v1.1.

</div>

---

## What's next

- Follow the [5-minute quickstart](getting-started.md).
- Browse the [configuration reference](configuration.md).
- Learn about the [supply-chain model](supply-chain.md) — the bit that makes Aegis different.
- Read the [full specification](reference/spec.md).
