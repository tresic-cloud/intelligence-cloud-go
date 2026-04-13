---
title: "ADR-010: Goreleaser plus Cosign Keyless Signing"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-010
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Releasing]]"
tags:
  - decision/accepted
  - release
  - ci
  - security
  - compliance/soc2
aliases:
  - Release Toolchain ADR
  - Goreleaser ADR
  - Cosign ADR
  - R12
related:
  - "[[Releasing]]"
  - "[[Compliance Controls]]"
  - "[[Security Posture]]"
  - "[[Roadmap]]"
---

# ADR-010: Goreleaser plus Cosign Keyless Signing

## Status

Accepted

## Context

The specification requires that CLI binaries be published for Linux, macOS, and Windows on common CPU architectures, that every release carry an immutable version tag and changelog, and that the Go module be installable via standard tooling. The project's compliance posture also requires supply-chain security measures.

## Decision

Use `goreleaser` driven by `.goreleaser.yaml`. A tag push (`vX.Y.Z`) on `main` triggers `.github/workflows/release.yaml`, which:

1. Builds binaries for `{linux, darwin, windows} x {amd64, arm64}`.
2. Generates checksums.
3. Signs with `cosign` keyless (GitHub OIDC) -- no long-lived signing keys to manage.
4. Publishes a GitHub Release with a `go.mod`-discoverable tag so `go get ...@vX.Y.Z` works immediately.

## Consequences

### Positive

- **Standard in Go ecosystem**: used by goreleaser itself, Terraform providers, k6, and Docker BuildX.
- **SLSA L2+ compliance**: cosign keyless signing with GitHub OIDC satisfies supply-chain provenance requirements with no long-lived secrets.
- **Automatic changelog**: goreleaser generates release notes from conventional commits.
- **Cross-platform**: six binary targets from a single workflow.

### Negative

- **CI dependency**: the release workflow depends on goreleaser and cosign being available in CI. Both are well-maintained and available as GitHub Actions.
- **Keyless trust model**: consumers must trust the Sigstore transparency log and GitHub's OIDC provider. This is the standard model for open-source Go projects.

## Alternatives Considered

### Hand-rolled Makefile + go build

Works, but re-implements what goreleaser already handles: checksums, archives, release notes, and Homebrew tap generation.

### gh release create + GitHub Actions matrix

Similar objection. More manual steps and no built-in signing integration.

## References

- [[Releasing]]
- [[Compliance Controls]]
- [goreleaser](https://goreleaser.com/)
- [cosign keyless signing](https://docs.sigstore.dev/signing/quickstart/)
