---
title: "ADR-005: Pinned OpenAPI with Drift PRs"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-005
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Overview]]"
tags:
  - decision/accepted
  - sdk
  - openapi
  - ci
aliases:
  - OpenAPI Sourcing ADR
  - Drift Detection ADR
  - R9
related:
  - "[[ADR-001 OpenAPI Codegen with oapi-codegen]]"
  - "[[ADR-011 Hand-curated Generated Code Stop-gap]]"
  - "[[Operation IDs]]"
  - "[[Releasing]]"
---

# ADR-005: Pinned OpenAPI with Drift PRs

## Status

Accepted

## Context

The SDK generates types and client stubs from the canonical OpenAPI specification maintained in the `intelligence-cloud` repository. The SDK's build must be reproducible (critical for release signing and CI reliability), but it must also stay in sync with the evolving backend contract. These two goals are in tension: fetching the spec at build time ensures freshness but breaks reproducibility; pinning a local copy ensures reproducibility but can drift.

## Decision

Maintain a pinned copy of the canonical `openapi.yaml` in this repository at `testdata/openapi.yaml`, updated by a Makefile target (`make pull-openapi`) that runs `scripts/pull-openapi.sh`. The script fetches the spec from the `intelligence-cloud` repository at a pinned commit SHA recorded in `testdata/openapi.commit`. Code generation reads from the local pinned copy.

CI includes a "drift" job that, on pushes to `main`, fetches the latest `intelligence-cloud/main` openapi.yaml and opens a pull request if it differs from the pinned copy. This drift PR is never auto-merged -- it requires human review.

## Consequences

### Positive

- **Reproducible builds**: builds work without network access and always use the exact spec version the SDK was tested against.
- **Explicit contract pinning**: each SDK release is tied to a specific backend contract commit, making contract drift a review-gated event.
- **Timely updates**: the drift-detection CI job ensures the team is notified when the backend contract changes, keeping the SDK within one release cycle of the latest contract.

### Negative

- **Manual update step**: someone must merge the drift PR and re-run codegen. This is intentional friction to prevent untested contract changes from landing silently.
- **Stale spec risk**: if drift PRs are ignored for too long, the SDK falls behind. Mitigated by CI visibility and team process.

## Alternatives Considered

### Git submodule of intelligence-cloud

Heavyweight (500+ MB of unrelated code), slow clones, and breaks private-repo access for external consumers.

### Release-artefact download

Cleaner, but requires the intelligence-cloud repo to publish `openapi.yaml` as a release asset. That pipeline does not yet exist and is out of scope for v0.1.

### Fetch at build time

Breaks reproducibility and couples the build to network availability and the upstream repo's uptime.

## References

- [[ADR-001 OpenAPI Codegen with oapi-codegen]]
- [[Releasing]]
- [[Operation IDs]]
