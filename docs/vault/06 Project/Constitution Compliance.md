---
title: Constitution Compliance
type: project
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - project
  - constitution
  - compliance
  - governance
aliases:
  - Constitution Check
  - Governance Compliance
related:
  - "[[Status]]"
  - "[[Roadmap]]"
  - "[[01 Architecture/_MOC|Architecture MOC]]"
  - "[[02 Decisions/_MOC|Decisions MOC]]"
  - "[[04 Operations/_MOC|Operations MOC]]"
---

# Constitution Compliance

The Speckit constitution defines 11 mandatory principles that every project must satisfy. This note maps each principle to concrete evidence in the vault and codebase, recording whether the project satisfies it directly, adapts it with documented rationale, or marks it not applicable.

The constitution itself lives at `.specify/memory/constitution.md`. The formal compliance check was performed during plan authoring and is recorded in `specs/001-sdk-cli-foundation/plan.md` under "Constitution Check". This note transcribes that assessment into vault-native form with wikilinks.

---

## I. Library-First with CLI Interface

**One-liner**: Complex business logic extracted into standalone, independently testable libraries; every library exposed via CLI with text in/out.

**Status**: satisfied

The project is literally a reusable Go library (`package intelligencecloud`) with a companion CLI (`icctl`). The SDK is importable via `go get` and independently testable -- no CLI needed. The CLI exposes the full SDK surface to operators and scripts, accepting text input (flags, JSON stdin) and producing text output (JSON, table).

**Evidence**:
- [[01 Architecture/SDK Surface]] -- the public Go API contract
- [[01 Architecture/CLI Surface]] -- the `icctl` command tree with text in/out protocol
- [[01 Architecture/Overview]] -- module layout showing library + CLI separation

---

## II. Test-First Development (TDD, >=80%)

**One-liner**: Write failing tests first; achieve at least 80% line coverage before considering any task complete.

**Status**: satisfied

Every implementation task in the backlog has a preceding test task. The commit log on `001-sdk-cli-foundation` demonstrates strict red-green-refactor: `test(A-1)` lands before `feat(A-2)`, `test(A-3)` before `feat(A-4)`, and so on through all 57 commits. Coverage on all hand-written SDK packages exceeds 80%, with the root package at 94.6% and `auth` at 100%.

**Evidence**:
- [[Status]] -- coverage table showing 94.6% / 100% / 93.9% / 100% across hand-written packages
- Commit history: `git log --oneline` on the branch shows alternating `test()` / `feat()` pairs
- CI coverage gate (Wave 2, `[E-3]`) will enforce the 80% threshold automatically

---

## III. Integration Testing

**One-liner**: Contract tests for library interfaces, service-to-service tests, external API integration tests.

**Status**: satisfied

Three levels of integration testing are in place or planned:

1. **Contract tests**: the codegen drift test (`[B-1]`) asserts that re-running `oapi-codegen` produces no changes, validating the generated-code contract. The compilation test (`[B-3]`) ensures generated types remain Go-compilable.
2. **Transport tests**: `httptest.Server`-backed tests in `internal/transport` exercise the full RoundTripper pipeline -- auth injection, retry, OTel span emission, 401 re-fetch, and typed error mapping -- against a real HTTP server.
3. **Integration smoke suite**: a 5-scenario integration test suite (`go test -tags=integration`) against the staging environment, gated behind secret-injected credentials in CI (`[D-29]`).

**Evidence**:
- [[01 Architecture/HTTP Transport]] -- transport test strategy
- [[02 Decisions/ADR-011 Hand-curated Generated Code Stop-gap]] -- drift test approach
- [[Status]] -- `internal/transport` at 93.9% coverage from `httptest.Server` tests

---

## IV. Observability

**One-liner**: W3C trace propagation, Prometheus metrics, structured logging, business error tracking.

**Status**: adapted

The constitution's observability principle is framed around backend HTTP handlers (inbound `traceparent`, Prometheus `InstrumentedService` decorators). For an outbound client SDK, the correct adaptation is:

- **(a)** Propagate W3C trace context (`traceparent`, `tracestate`) on outbound requests.
- **(b)** Emit OpenTelemetry client spans per semconv 1.24 with `http.request.method`, `url.full`, `http.response.status_code`, and `server.address` attributes.
- **(c)** Integrate with `log/slog` for structured logging with automatic PII redaction.
- **(d)** Zero-cost when no `TracerProvider` is configured (benchmarked at under 100 ns/op additional).

Prometheus `InstrumentedService`/`InstrumentedRepository` decorators do not apply -- there are no services or repositories to decorate.

**Evidence**:
- [[01 Architecture/Telemetry Contract]] -- OTel span attributes, slog integration, no-op guarantee
- [[02 Decisions/ADR-007 OpenTelemetry Client Spans semconv 1.24]] -- semconv choice and rationale
- [[04 Operations/Observability Runbook]] -- interpreting OTel spans and logs

---

## V. Documentation First

**One-liner**: Documentation must exist before implementation begins.

**Status**: satisfied

The project followed a strict documentation-first workflow. The full specification chain -- `spec.md`, `plan.md`, `research.md`, `data-model.md`, three contract documents, and `quickstart.md` -- was authored before a single line of implementation code. This vault was scaffolded as part of Wave 0/1 documentation work, and GoDoc coverage on every exported symbol is enforced by `revive`'s `exported` rule in `.golangci.yaml`.

**Evidence**:
- This vault -- architecture, decisions, guides, operations, reference, and project notes all exist before Wave 2 begins
- `specs/001-sdk-cli-foundation/` -- full spec chain authored pre-implementation
- `.golangci.yaml` -- `revive` with `exported` rule enforces GoDoc on all public symbols
- [[02 Decisions/_MOC|Decisions MOC]] -- 11 ADRs documenting every significant technical choice

---

## VI. Quality Standards

**One-liner**: All code production-ready. No TODO/FIXME/Phase 2. Complete error handling, input validation, security measures.

**Status**: satisfied

The codebase enforces quality through multiple mechanisms:

- **Linting**: `.golangci.yaml` configures `go vet`, `staticcheck`, `revive`, `errcheck`, `gosec`, and `govulncheck`. The no-TODO/FIXME policy is enforced by lint rules.
- **Typed errors**: all error paths return concrete types implementing the `APIError` interface with sentinel wrapping for `errors.Is` matching. No bare `error` strings.
- **Input validation**: `NewClient` validates all options (HTTPS-only for non-localhost, non-nil credential provider, valid retry policy parameters). CLI flags are validated by Cobra.
- **No deferred work**: zero `TODO`, `FIXME`, or `Phase 2` comments in the codebase.

**Evidence**:
- [[05 Reference/Error Hierarchy]] -- the 8 concrete error types
- [[02 Decisions/ADR-008 Typed Error Hierarchy with Sentinels]] -- error design rationale
- [[Status]] -- coverage numbers demonstrating thorough testing

---

## VII. APIs as First-Class Features

**One-liner**: Every API documented as if public-facing. Include examples and error responses. Version appropriately.

**Status**: satisfied

The Go SDK's public API is a product surface. Every exported symbol carries GoDoc documentation. Every resource area will have at least one runnable example in `examples/`. Breaking changes are versioned via semver -- the SDK commits to compatibility from `v1.0.0` onwards, with pre-1.0 experimental APIs clearly marked.

**Evidence**:
- [[01 Architecture/SDK Surface]] -- the public Go API contract with semver commitment
- [[02 Decisions/ADR-004 Pre-1.0 Experimental API Policy]] -- two-tier stability with `experimental` build tag
- [[05 Reference/Operation IDs]] -- the 21 OpenAPI operations mapped to SDK methods

---

## VIII. Scope-Based Authorization

**One-liner**: Every feature associated with permission scopes; every API endpoint has required scopes.

**Status**: N/A

The SDK does not define or enforce authorization scopes -- that is the backend's responsibility, enforced at Azure APIM and the backend service layer. The SDK preserves and surfaces `403 Forbidden` responses as a typed `AuthorizationError` so that callers can react to scope-related failures, but scope definition is not within the SDK's remit.

**Evidence**:
- [[05 Reference/Error Hierarchy]] -- `AuthorizationError` preserves 403 context
- [[02 Decisions/ADR-008 Typed Error Hierarchy with Sentinels]] -- error type covers 403

---

## IX. Feature Flag Architecture

**One-liner**: All new features behind feature flags for controlled rollouts.

**Status**: adapted

The SDK/CLI follows semver; the pre-1.0 (`v0.x`) release series is the feature-flag equivalent. Unstable surfaces may change between minor versions. Experimental APIs are marked with a `// Experimental:` GoDoc prefix and an `experimental` build tag, excluding them from the semver compatibility contract. This gives consumers a clear opt-in mechanism analogous to feature flags.

**Evidence**:
- [[02 Decisions/ADR-004 Pre-1.0 Experimental API Policy]] -- the two-tier stability model
- Pre-1.0 semver policy: breaking changes permitted in `v0.x` minor bumps

---

## X. Backend/Frontend Isolation

**One-liner**: Clear architectural isolation between backend and frontend; no direct database access from frontend.

**Status**: N/A

This principle does not apply. This repository IS the client -- it is neither a backend nor a frontend in the constitutional sense. The backend is an external dependency (the Intelligence Cloud API behind Azure APIM). The SDK communicates exclusively through well-defined HTTP APIs.

**Evidence**:
- [[01 Architecture/Overview]] -- data flow shows SDK as pure HTTP client
- [[01 Architecture/HTTP Transport]] -- all communication via HTTP, no direct data-store access

---

## XI. Built for Compliance

**One-liner**: All decisions account for compliance requirements (SOC2, ISO27001, HiTrust, HIPAA).

**Status**: satisfied

The project addresses compliance through multiple controls:

- **Secret handling**: CLI credentials stored in OS keychain via `go-keyring` with `0600` file fallback. No plaintext secrets in config or logs.
- **Audit trail**: structured `slog` logs and OTel spans carry request IDs (`X-Request-Id`) for correlation. Every API call is traceable.
- **HTTPS enforcement**: `NewClient` rejects non-`localhost` base URLs that do not use HTTPS.
- **PII redaction**: bearer tokens and authorization headers are automatically redacted in logs, error messages, and OTel attributes.
- **No PII persistence**: the SDK/CLI does not persist any PII to disk beyond the credential store.

**Evidence**:
- [[04 Operations/Compliance Controls]] -- full controls matrix with SOC2/HIPAA mapping
- [[04 Operations/Security Posture]] -- threat model, secrets handling, redaction
- [[02 Decisions/ADR-003 OS Keychain Credential Storage]] -- keychain-first credential storage

---

## Summary Table

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Library-First with CLI Interface | satisfied | SDK is library; CLI wraps it |
| II | Test-First Development (TDD, >=80%) | satisfied | 94.6% root, 100% auth |
| III | Integration Testing | satisfied | httptest, drift, staging smoke |
| IV | Observability | adapted | OTel client spans, not Prometheus decorators |
| V | Documentation First | satisfied | Spec chain + vault before code |
| VI | Quality Standards | satisfied | Lint, typed errors, no TODOs |
| VII | APIs as First-Class | satisfied | GoDoc + semver + examples |
| VIII | Scope-Based Authorization | N/A | Scopes are backend concern |
| IX | Feature Flag Architecture | adapted | Pre-1.0 + experimental build tag |
| X | Backend/Frontend Isolation | N/A | This repo IS the client |
| XI | Built for Compliance | satisfied | Keychain, HTTPS, redaction, audit trail |

Two adaptations (IV, IX) and two N/A determinations (VIII, X) are documented with rationale and do not constitute violations.

## See Also

- [[Status]] -- current implementation state
- [[Roadmap]] -- wave plan
- [[01 Architecture/_MOC|Architecture MOC]] -- architecture notes referenced above
- [[02 Decisions/_MOC|Decisions MOC]] -- ADRs providing evidence
- [[04 Operations/_MOC|Operations MOC]] -- operations notes referenced above
