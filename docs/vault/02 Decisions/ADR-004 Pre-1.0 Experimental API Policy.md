---
title: "ADR-004: Pre-1.0 Experimental API Policy"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-004
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[SDK Surface]]"
tags:
  - decision/accepted
  - sdk
  - semver
  - stability
aliases:
  - Experimental API ADR
  - Stability Policy ADR
  - R10
related:
  - "[[SDK Surface]]"
  - "[[Roadmap]]"
  - "[[Constitution Compliance]]"
---

# ADR-004: Pre-1.0 Experimental API Policy

## Status

Accepted

## Context

The project constitution requires that new features be behind feature flags for controlled rollouts. For a backend service, this means runtime feature flags (e.g. LaunchDarkly). For a client-side Go library, runtime flags add a network dependency and inappropriate complexity. A compile-time mechanism is needed that provides the same intent: controlled exposure of unstable functionality.

Additionally, pre-1.0 releases (`v0.x`) can break compatibility per semver, but once the module reaches v1, consumers need a way to opt into experimental features without affecting their stable code paths.

## Decision

Implement a two-tier stability policy:

1. **Stable**: any exported symbol without the `// Experimental:` GoDoc prefix. Breaking changes only on major version bumps.
2. **Experimental**: symbols prefixed with `// Experimental:` in their GoDoc and placed behind the `experimental` build tag (`//go:build experimental`). Not compiled into default builds; consumers opt in with `go build -tags=experimental`. No semver compatibility commitment. Moved to Stable by removing the tag and comment when the design settles.

Pre-1.0 releases (`v0.x`) additionally provide whole-module experimental semantics.

## Consequences

### Positive

- **Compile-time feature flag**: the build tag is the closest Go analogue to runtime feature flags, and it is auditable via `git grep "Experimental:"`.
- **Ecosystem precedent**: matches the Go convention (`x/exp`, `slices.Experimental`, etc.).
- **No runtime cost**: experimental symbols do not exist in the default binary.

### Negative

- **Consumer friction**: opting into experimental features requires a build tag, which is slightly more ceremony than a runtime flag.
- **Potential confusion**: consumers must understand that `v0.x` is experimental as a whole, while post-v1 experimental features are per-symbol.

## Alternatives Considered

### Runtime feature flags (LaunchDarkly)

Adds a network dependency. Inappropriate for a client library that should not phone home.

### Separate `intelligence-cloud-go/experimental` submodule

Fragments the API surface and breaks the single `go get` import model.

## References

- [[SDK Surface]]
- [[Constitution Compliance]]
