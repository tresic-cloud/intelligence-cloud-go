---
title: "ADR-011: Hand-curated Generated Code Stop-gap"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-011
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[Overview]]"
  - "[[ADR-001 OpenAPI Codegen with oapi-codegen]]"
tags:
  - decision/accepted
  - sdk
  - codegen
  - openapi
aliases:
  - Codegen Workaround ADR
  - Hand-curated Codegen
related:
  - "[[ADR-001 OpenAPI Codegen with oapi-codegen]]"
  - "[[ADR-005 Pinned OpenAPI with Drift PRs]]"
  - "[[Operation IDs]]"
---

# ADR-011: Hand-curated Generated Code Stop-gap

## Status

Accepted

## Context

The canonical Intelligence Cloud OpenAPI specification is authored in OpenAPI 3.1 format. At the time of this decision, `oapi-codegen` (chosen in [[ADR-001 OpenAPI Codegen with oapi-codegen]]) has incomplete support for certain OpenAPI 3.1 features. Specifically, some schema constructs produce either incorrect Go types or compilation errors in the generated output.

The SDK cannot wait for upstream tooling to mature -- it must ship usable types for all 21 operations in the current spec.

## Decision

For the initial release, hand-curate the generated code as a stop-gap:

1. Run `oapi-codegen` against the pinned OpenAPI spec to produce `internal/generated/types.gen.go` and `internal/generated/client.gen.go`.
2. Manually fix any compilation errors or type mismatches caused by OpenAPI 3.1 gaps.
3. Check the curated output into version control (not `.gitignore`d).
4. Include a drift test (`internal/generated/drift_test.go`) that re-runs codegen and compares the output to the checked-in version. The test logs warnings for known divergences but does not fail the build on expected OpenAPI 3.1 gaps.
5. As `oapi-codegen` improves its 3.1 support, progressively remove manual fixes until the output is fully automated.

## Consequences

### Positive

- **Ships now**: the SDK can release with correct types for all operations without waiting for upstream tooling.
- **Drift visibility**: the drift test ensures the team notices when `oapi-codegen` output changes, whether from upstream fixes or spec changes.
- **Escape hatch**: the `internal/` boundary means this stop-gap is invisible to consumers.

### Negative

- **Maintenance burden**: hand-curated generated code must be re-checked on every OpenAPI update. Mitigated by the drift test flagging changes.
- **Risk of divergence**: manual fixes could introduce bugs that pure codegen would not. Mitigated by compile tests that verify all generated types and functions build correctly.

### Exit Criteria

This ADR is superseded when `oapi-codegen` can process the canonical OpenAPI 3.1 spec without manual intervention. At that point, the generated files return to being fully automated and the manual curation step is removed.

## Alternatives Considered

### Downgrade OpenAPI spec to 3.0

The backend team owns the spec format. Requesting a downgrade would block the SDK on a cross-team negotiation and reduce the spec's expressiveness.

### Switch to a 3.1-native generator (ogen)

Considered, but ogen's public API was still evolving (see [[ADR-001 OpenAPI Codegen with oapi-codegen]]), and switching generators mid-project introduces more risk than hand-curating a known set of issues.

### Write all types by hand

Rejected for the same reasons as in ADR-001: manual transcription of 21+ operations is unsustainable and the specification forbids it.

## References

- [[ADR-001 OpenAPI Codegen with oapi-codegen]]
- [[ADR-005 Pinned OpenAPI with Drift PRs]]
