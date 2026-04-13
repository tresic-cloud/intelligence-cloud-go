---
title: Decisions MOC
type: moc
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - moc
  - decisions
aliases:
  - ADR Index
  - Decision Log
related:
  - "[[Home]]"
  - "[[01 Architecture/_MOC|Architecture MOC]]"
---

# Decisions

Every significant technical choice in the Intelligence Cloud Go SDK is recorded as an Architectural Decision Record (ADR). These records capture the context, decision, consequences, and alternatives so that future contributors understand not just *what* was chosen but *why*.

## Accepted Decisions

| ADR | Decision | Primary concern |
|---|---|---|
| [[ADR-001 OpenAPI Codegen with oapi-codegen]] | Use `oapi-codegen/v2` for types + client stubs | Code generation |
| [[ADR-002 Cobra and Viper for CLI]] | Use Cobra + Viper for the CLI framework | CLI framework |
| [[ADR-003 OS Keychain Credential Storage]] | Store CLI secrets in OS keychain via `go-keyring` | Credential storage |
| [[ADR-004 Pre-1.0 Experimental API Policy]] | Two-tier stability with `experimental` build tag | API stability |
| [[ADR-005 Pinned OpenAPI with Drift PRs]] | Pin canonical OpenAPI locally; detect drift via CI PRs | Contract sourcing |
| [[ADR-006 RoundTripper-based Retry with Full Jitter]] | Retry as a composable RoundTripper with full jitter | Retry strategy |
| [[ADR-007 OpenTelemetry Client Spans semconv 1.24]] | Emit OTel client spans using semconv 1.24 | Observability |
| [[ADR-008 Typed Error Hierarchy with Sentinels]] | Eight concrete error types with sentinel wrapping | Error handling |
| [[ADR-009 Generic Iterator T for Pagination]] | Generic `Iterator[T]` with lazy page fetching | Pagination |
| [[ADR-010 Goreleaser plus Cosign Keyless Signing]] | goreleaser + cosign keyless for releases | Release toolchain |
| [[ADR-011 Hand-curated Generated Code Stop-gap]] | Hand-curate generated code until OpenAPI 3.1 tooling matures | Codegen workaround |

## Creating a New ADR

1. Copy the template from `_templates/ADR.md`.
2. Assign the next sequential number.
3. Fill in all sections (Context, Decision, Consequences, Alternatives).
4. Update this MOC.

## Related MOCs

- [[01 Architecture/_MOC|Architecture MOC]] -- the architecture these decisions support
- [[05 Reference/_MOC|Reference MOC]] -- detailed reference for patterns chosen by these ADRs
