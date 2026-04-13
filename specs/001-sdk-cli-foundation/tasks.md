# Phase 2 Tasks: Intelligence Cloud Go SDK & CLI Foundation

**Date**: 2026-04-13
**Branch**: `001-sdk-cli-foundation`
**Spec**: [spec.md](./spec.md) · **Plan**: [plan.md](./plan.md) · **Research**: [research.md](./research.md) · **Data model**: [data-model.md](./data-model.md)
**Contracts**: [sdk-api.md](./contracts/sdk-api.md), [cli-schema.md](./contracts/cli-schema.md), [telemetry.md](./contracts/telemetry.md)
**Tracking**: [IDB-1353](https://tresic.atlassian.net/browse/IDB-1353) · Upstream: [IDB-1354](https://tresic.atlassian.net/browse/IDB-1354)

## Summary

**Total tasks**: 125, organised into 5 streams. Every implementation task is preceded by a failing-test task (Constitution II — TDD mandatory). Every task is intended to become one GitHub issue.

| Stream | Scope | Tasks | Priority mix |
|---|---|---|---|
| A | SDK core (client, options, errors, retry, pagination, telemetry, auth) | 28 | 25×P1 · 3×P2 |
| B | OpenAPI codegen pipeline + resource wrappers | 23 | 21×P1 · 2×P2 |
| C | HTTP transport (retry, OTel, auth inject, redaction, 401 re-fetch) | 16 | 13×P1 · 2×P2 · 1×P3 |
| D | `icctl` CLI (commands, profile store, output, destructive preview) | 32 | 30×P2 · 2×P3 |
| E | Release, CI, docs, semver automation, supply-chain signing | 26 | 5×P1 · 12×P2 · 9×P3 |

**Priority map to User Stories** (per spec.md §User Scenarios):
- **P1** = User Story 1 (developer-facing SDK) — the MVP
- **P2** = User Story 2 (operator-facing CLI) — ships after P1
- **P3** = User Story 3 (release polish, docs, discoverability) — continuous

## How to use this file

- Each task has a stable ID like `[A-12]`. IDs are used as cross-references in dependencies.
- `Depends on` lists direct blockers. A task with `Depends on: —` can start immediately.
- `Kind` is one of `test` | `impl` | `bench` | `docs` | `refactor` | `script` | `config`.
- **TDD discipline**: every `impl` task has a preceding `test` task (or tests are bundled into the same task and must be written first). CI coverage gate enforces ≥80% (constitution II + FR-020).
- The **Parallelisation Plan** at the bottom groups tasks into waves that can be dispatched to parallel agents.

---

## Stream A · SDK core

Hand-written ergonomic layer in the root `intelligencecloud` package plus the `auth` sub-package. Foundation that Streams B, C, D depend on.

### [A-1] Test: `auth.Token` struct and `CredentialProvider` interface

**File(s)**: `auth/provider_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing tests for the `Token` struct fields and `CredentialProvider` interface contract.

**Acceptance**:
- `Token` exposes `AccessToken string`, `ExpiresAt time.Time` (sdk-api.md §3).
- A mock satisfying `CredentialProvider` interface with `Token(ctx) (Token, error)` compiles.
- Zero-value `ExpiresAt` = "never expires" (data-model.md §1.2).

**Related**: IDB-1353

### [A-2] Implement: `auth.Token` struct and `CredentialProvider` interface

**File(s)**: `auth/provider.go`
**Depends on**: `[A-1]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Define `Token` and `CredentialProvider` per sdk-api.md §3.

**Acceptance**: FR-002; all [A-1] tests green.

**Related**: IDB-1353

### [A-3] Test: `auth.StaticToken` provider

**File(s)**: `auth/static_test.go`
**Depends on**: `[A-2]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing tests for `StaticToken`.

**Acceptance**:
- Always returns `Token{AccessToken: s, ExpiresAt: zero}`, nil error.
- Concurrent-safe.
- Empty string accepted (no panic).

**Related**: IDB-1353

### [A-4] Implement: `auth.StaticToken`

**File(s)**: `auth/static.go`
**Depends on**: `[A-3]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `func StaticToken(s string) CredentialProvider`. FR-002.

**Related**: IDB-1353

### [A-5] Test: `auth.RefreshFunc` with expiry-aware caching

**File(s)**: `auth/refresh_test.go`
**Depends on**: `[A-2]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests covering invocation, 30-s-margin caching, error forwarding, zero-expiry handling, concurrent safety (single-flight).

**Acceptance**: cites data-model.md §1.2, FR-002.

**Related**: IDB-1353

### [A-6] Implement: `auth.RefreshFunc` adapter

**File(s)**: `auth/refresh.go`
**Depends on**: `[A-5]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Thread-safe expiry-aware token cache with 30-s safety margin.

**Acceptance**: FR-002; all [A-5] tests green.

**Related**: IDB-1353

### [A-7] Test: Sentinel errors and `APIError` interface

**File(s)**: `errors_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for seven sentinels (`ErrAuthentication`, `ErrAuthorization`, `ErrValidation`, `ErrNotFound`, `ErrConflict`, `ErrRateLimit`, `ErrServer`) and the `APIError` interface (sdk-api.md §4).

**Related**: IDB-1353

### [A-8] Test: Concrete error types with `errors.Is` / `errors.As` support

**File(s)**: `errors_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for all eight concrete error types, field population, sentinel routing via `Unwrap`, and 4-KiB `UnexpectedError.CauseBody` cap (R7, data-model.md §1.5).

**Acceptance**: FR-004, SC-008.

**Related**: IDB-1353

### [A-9] Test: `ConfigurationError` (non-`APIError`)

**File(s)**: `errors_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing tests that `ConfigurationError` implements `error` but NOT `APIError`.

**Related**: IDB-1353

### [A-10] Implement: Error hierarchy (sentinels + concrete types + `ConfigurationError`)

**File(s)**: `errors.go`
**Depends on**: `[A-7]`, `[A-8]`, `[A-9]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Full error hierarchy per sdk-api.md §4 and data-model.md §1.5.

**Acceptance**: FR-004, SC-008; all [A-7..9] tests green.

**Related**: IDB-1353

### [A-11] Test: `RetryPolicy`, `JitterStrategy`, defaults, validation

**File(s)**: `retry_test.go`
**Depends on**: `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for `DefaultRetryPolicy()`, `NoRetry()`, and `Validate()` rejecting `MaxAttempts<1`, `BaseDelay<=0`, `MaxDelay<BaseDelay`.

**Acceptance**: cites sdk-api.md §6, data-model.md §1.3 + §5.

**Related**: IDB-1353

### [A-12] Implement: `RetryPolicy`, `JitterStrategy`, factories, validation

**File(s)**: `retry.go`
**Depends on**: `[A-11]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Policy struct + validation only — NO transport wiring (Stream C owns that). FR-009a (policy struct).

**Acceptance**: `Validate()` returns `*ConfigurationError`; all [A-11] tests green.

**Related**: IDB-1353

### [A-13] Test: `CallOption`, `ListOption`, helper constructors

**File(s)**: `options_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for `WithIdempotencyKey`, `WithRequestTimeout`, `WithExtraHeader` (rejects `Authorization`), `WithPageSize(>0)`, `WithMaxItems(>=0)`, `WithFilter`.

**Related**: IDB-1353

### [A-14] Implement: options + shared config structs

**File(s)**: `options.go`
**Depends on**: `[A-13]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: `CallOption` + `ListOption` interfaces and all helpers per sdk-api.md §2.1/2.2.

**Related**: IDB-1353

### [A-15] Test: `Iterator[T]` lifecycle

**File(s)**: `pagination_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for `Next`/`Value`/`Err`/`Close` invariants (data-model.md §1.4).

**Related**: IDB-1353

### [A-16] Test: `Iterator[T]` multi-page fetching + `PageInfo`

**File(s)**: `pagination_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for on-demand page fetching, `PageInfo` fields, `WithMaxItems` cap.

**Acceptance**: FR-007.

**Related**: IDB-1353

### [A-17] Test: `Iterator[T]` error and cancellation paths

**File(s)**: `pagination_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing tests for fetcher-error surfacing, context cancellation, post-error idempotence.

**Acceptance**: FR-006.

**Related**: IDB-1353

### [A-18] Implement: generic `Iterator[T]` with `PageInfo`

**File(s)**: `pagination.go`
**Depends on**: `[A-15]`, `[A-16]`, `[A-17]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Generic iterator accepting injected page-fetcher; memory bounded to one page. FR-007.

**Related**: IDB-1353

### [A-19] Test: `NewClient` valid constructions

**File(s)**: `client_test.go`
**Depends on**: `[A-2]`, `[A-10]`, `[A-12]`, `[A-14]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for HTTPS URLs, localhost/127.0.0.1 HTTP, all `WithXxx` option applications, resource-service field initialisation.

**Related**: IDB-1353

### [A-20] Test: `NewClient` validation failures

**File(s)**: `client_test.go`
**Depends on**: `[A-2]`, `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for nil provider, non-HTTPS non-localhost, invalid `RetryPolicy`. All return `*ConfigurationError`, which does NOT satisfy `APIError`.

**Related**: IDB-1353

### [A-21] Implement: `Client` struct + `NewClient` with functional options

**File(s)**: `client.go`
**Depends on**: `[A-19]`, `[A-20]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Full `Client` per data-model.md §1.1. Resource-service fields as nil stubs until Streams B/C wire them.

**Acceptance**: FR-001, FR-005, FR-008.

**Related**: IDB-1353

### [A-22] Test: telemetry helpers — span naming + attribute builders

**File(s)**: `telemetry_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for `intelligencecloud.{operationID}` span name, attribute key/value helpers, `url.full` redaction regex, `RedactedHeaders` default set (telemetry.md).

**Related**: IDB-1353

### [A-23] Test: telemetry log-redacting handler

**File(s)**: `telemetry_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests that the handler strips `token|secret|key|password` attrs, never emits `Authorization`, injects `trace_id`/`span_id` from active span (telemetry.md §Logging).

**Related**: IDB-1353

### [A-24] Implement: telemetry helpers (span name, attrs, URL/header redaction, log handler)

**File(s)**: `telemetry.go`
**Depends on**: `[A-22]`, `[A-23]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Reusable helpers for Stream C transport. FR-008a (helper layer only).

**Related**: IDB-1353

### [A-25] Test: default User-Agent format

**File(s)**: `client_test.go`
**Depends on**: `[A-21]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Failing test for `intelligence-cloud-go/vX.Y.Z (os/arch)` pattern; `WithUserAgent` override.

**Related**: IDB-1353

### [A-26] Implement: version-stamped default User-Agent

**File(s)**: `client.go`
**Depends on**: `[A-25]`, `[E-24]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Derive from the `internal/version` package populated by ldflags (see [E-24]).

**Related**: IDB-1353

### [A-27] Docs: root and `auth` package-level GoDoc

**File(s)**: `doc.go`, `auth/doc.go`
**Depends on**: `[A-6]`, `[A-21]`
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Package-overview comments with usage sketch; passes `go vet`. FR-019, SC-002.

**Related**: IDB-1353

### [A-28] Test: `FieldError` round-trip on `ValidationError`

**File(s)**: `errors_test.go`
**Depends on**: `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Dedicated test pinning `FieldError` shape and `errors.As` recovery of field-level detail.

**Related**: IDB-1353

### Stream A · Cross-stream dependencies outgoing
- `errors.go` exports consumed by transport error mapping — Stream C depends on [A-10].
- `auth` package consumed by transport auth injection — Stream C depends on [A-2], [A-4], [A-6].
- `RetryPolicy` consumed by transport retry loop — Stream C depends on [A-12].
- `CallOption`/`ListOption` consumed by resource wrappers + transport — Streams B, C depend on [A-14].
- `Iterator[T]` consumed by resource wrappers' `List*` methods — Stream B depends on [A-18].
- Telemetry helpers consumed by transport span emission + log handler — Stream C depends on [A-24].
- `Client` consumed by Stream B resource-service wiring and Stream D CLI — [A-21] is a hinge dependency.

---

## Stream B · OpenAPI codegen pipeline + resource wrappers

Generated layer (never hand-edited) plus thin, hand-written resource-service wrappers that expose idiomatic Go methods to consumers.

### [B-1] Test: contract-drift test — every `operationId` has a generated function

**File(s)**: `internal/generated/drift_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing test parsing `testdata/openapi.yaml`, extracting every `operationId` (currently 21: `login`, `getMe`, `createAvatarUploadURL`, `confirmAvatar`, `deleteAvatar`, `getNotificationPreferences`, `patchNotificationPreferences`, `getDailyRecapPreferences`, `patchDailyRecapPreferences`, `createReseller`, `listResellers`, `getReseller`, `updateReseller`, `patchReseller`, `deactivateReseller`, `reactivateReseller`, `listResellerAuditLogs`, `listLocationConnectors`, `listCompanyConnectors`, `patchCompanyUser`, `listProducts`), and asserting each has a generated function in `client.gen.go` plus a type in `types.gen.go`.

**Acceptance**: FR-003, SC-001, SC-007, R9.

**Related**: IDB-1353, IDB-1354

### [B-2] Test: codegen idempotence shell check

**File(s)**: `scripts/codegen_test.sh`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Script that runs `scripts/codegen.sh` then fails if `git diff --exit-code internal/generated/` shows changes.

**Acceptance**: R9, SC-007.

**Related**: IDB-1353, IDB-1354

### [B-3] Test: compilation sentinel for generated package

**File(s)**: `internal/generated/compile_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing test that instantiates a representative generated request + response type and asserts every `operationId` has a corresponding exported function.

**Related**: IDB-1353

### [B-4] Config: pin initial OpenAPI snapshot + commit SHA

**File(s)**: `testdata/openapi.yaml`, `testdata/openapi.commit`
**Depends on**: —
**Kind**: `config`
**Priority**: `P1`
**Effort**: `S`

**Goal**: One-time bootstrap copy from `../intelligence-cloud/backend/docs/api/openapi.yaml` + SHA pin. R9.

**Related**: IDB-1353, IDB-1354

### [B-5] Config: `oapi-codegen.yaml`

**File(s)**: `oapi-codegen.yaml`
**Depends on**: `[B-4]`
**Kind**: `config`
**Priority**: `P1`
**Effort**: `S`

**Goal**: oapi-codegen v2 config: `package: generated`, output `internal/generated/`, types + client only (no server). Disable `embedded-spec`. R1.

**Related**: IDB-1353

### [B-6] Script: `scripts/codegen.sh` + `make codegen`

**File(s)**: `scripts/codegen.sh`, `Makefile`
**Depends on**: `[B-4]`, `[B-5]`
**Kind**: `script`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Automate oapi-codegen invocation; `chmod +x`; `set -euo pipefail`; install hint if oapi-codegen missing. Turns [B-1], [B-2], [B-3] green.

**Related**: IDB-1353

### [B-7] Script: `scripts/pull-openapi.sh` + `make pull-openapi`

**File(s)**: `scripts/pull-openapi.sh`, `Makefile`
**Depends on**: `[B-4]`
**Kind**: `script`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Fetch latest canonical `openapi.yaml` from `intelligence-cloud` (local `IC_REPO_PATH` for dev, `gh api` fallback for CI), validate YAML, update SHA pin, print summary line.

**Acceptance**: R9, SC-007.

**Related**: IDB-1353, IDB-1354

### [B-8] Docs: `internal/generated/doc.go`

**File(s)**: `internal/generated/doc.go`
**Depends on**: `[B-6]`
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Package doc stating generated-do-not-edit, re-gen via `make codegen`, pinned spec path.

**Related**: IDB-1353

### [B-9] Test: `MeService` wrapper

**File(s)**: `me_test.go`
**Depends on**: `[B-6]`, `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: `httptest.Server`-backed failing tests for all 8 `/api/v1/me/*` operations, status-to-typed-error mapping (401→`*AuthenticationError`, 403→`*AuthorizationError`, 404→`*NotFoundError`, 500→`*ServerError`), ctx propagation.

**Acceptance**: FR-003, FR-004, FR-006, SC-008.

**Related**: IDB-1353

### [B-10] Implement: `MeService`

**File(s)**: `me.go`
**Depends on**: `[B-9]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: 8 methods; each delegates to generated client; uses shared `mapError` helper.

**Related**: IDB-1353

### [B-11] Test: `ResellerService` wrapper (CRUD + deactivate/reactivate + paginated list)

**File(s)**: `resellers_test.go`
**Depends on**: `[B-6]`, `[A-10]`, `[A-18]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: `httptest.Server` tests for all 7 reseller ops; `Iterator[Reseller]` pagination via flat `next_cursor`/`has_more`; error mapping for 400/401/403/404/409/422; `DeactivateResellerRequest.Reason` required.

**Acceptance**: FR-003, FR-007, FR-004.

**Related**: IDB-1353

### [B-12] Implement: `ResellerService`

**File(s)**: `resellers.go`
**Depends on**: `[B-11]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: `List/Get/Create/Update/Patch/Deactivate/Reactivate` with `*Iterator[Reseller]`.

**Related**: IDB-1353

### [B-13] Test: `AuthService` (`Login` with OAuth error envelope)

**File(s)**: `auth_svc_test.go`
**Depends on**: `[B-6]`, `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Tests for `POST /api/v1/auth/login` with both standard and OAuth-style error envelopes (`error`+`error_description`); verifies NO `Authorization` header sent (endpoint has `security: []`).

**Related**: IDB-1353

### [B-14] Implement: `AuthService.Login`

**File(s)**: `auth_svc.go`
**Depends on**: `[B-13]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `Login(ctx, req) (*LoginResponse, error)` bypassing credential provider; dual-envelope error mapping.

**Related**: IDB-1353

### [B-15] Test: `AuditLogService` paginated list with filters

**File(s)**: `auditlogs_test.go`
**Depends on**: `[B-6]`, `[A-10]`, `[A-18]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Tests verifying `action`, `start_date`, `end_date`, `actor_id`, `search`, `cursor`, `limit` query params; cursor pagination.

**Related**: IDB-1353

### [B-16] Implement: `AuditLogService`

**File(s)**: `auditlogs.go`
**Depends on**: `[B-15]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `List(ctx, resellerID, ...ListOption) *Iterator[AuditLogEntry]` with audit-log-specific option helpers (`WithAction`, `WithDateRange`, `WithActorID`, `WithSearch`).

**Related**: IDB-1353

### [B-17] Test: `ConnectorService` (location + company scopes)

**File(s)**: `connectors_test.go`
**Depends on**: `[B-6]`, `[A-10]`, `[A-18]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Tests for both endpoints; nested pagination (`pagination.next_cursor`/`.has_more`); company-scoped response's `errors` summary array exposed alongside data.

**Related**: IDB-1353

### [B-18] Implement: `ConnectorService`

**File(s)**: `connectors.go`
**Depends on**: `[B-17]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `ListByLocation` (returns iterator) + `ListByCompany` (returns `*CompanyConnectorList` with `Iterator` + `ErrorSummary`).

**Related**: IDB-1353

### [B-19] Test: `ProductService.List` (non-paginated)

**File(s)**: `products_test.go`
**Depends on**: `[B-6]`, `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `GET /api/v1/products` returns `[]Product`; 401/403 error mapping.

**Related**: IDB-1353

### [B-20] Implement: `ProductService`

**File(s)**: `products.go`
**Depends on**: `[B-19]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `List(ctx) ([]Product, error)` — non-paginated.

**Related**: IDB-1353

### [B-21] Test: `UserService.PatchCompanyUser`

**File(s)**: `users_test.go`
**Depends on**: `[B-6]`, `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `PATCH /api/v1/companies/{companyId}/users/{userId}`; 400/401/403/404 mapping.

**Related**: IDB-1353

### [B-22] Implement: `UserService`

**File(s)**: `users.go`
**Depends on**: `[B-21]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `PatchCompanyUser(ctx, companyID, userID, req) (*CompanyUser, error)`.

**Related**: IDB-1353

### [B-23] Scaffold: `CompanyService`, `LocationService`, `VerticalService`

**File(s)**: `companies.go`, `locations.go`, `verticals.go`
**Depends on**: `[B-6]`, `[A-21]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Empty service types with back-pointer to `*Client` + GoDoc stating "methods added as canonical OpenAPI grows (IDB-1354)". No `TODO`/`FIXME`. Files compile. Wired as `Client.Companies`/`Locations`/`Verticals` fields.

**Acceptance**: FR-003 (coverage bounded by spec contents).

**Related**: IDB-1353, IDB-1354

### Stream B · Cross-stream dependencies outgoing
- Generated `client.gen.go` defines the per-operation HTTP functions that resource wrappers call and that Stream C's transport wraps.
- Resource wrappers use Stream A's error hierarchy ([A-10]) and `Iterator[T]` ([A-18]) via a shared `mapError` helper.
- [B-2] codegen idempotence check wired into Stream E CI.
- [B-7] pull-openapi script wired into Stream E's scheduled drift workflow.

---

## Stream C · HTTP transport layer

Cross-cutting `http.RoundTripper` that composes auth injection, retry, OTel spans, trace-context propagation, slog logging, redaction, 401 re-fetch-and-retry-once, HTTP-status-to-typed-error mapping, User-Agent injection, and `X-Request-Id` capture.

### [C-1] Test: redaction helpers (URL query params, headers)

**File(s)**: `internal/transport/redact_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing tests: `RedactURL` strips values of keys matching `token|secret|key|password` regex; `RedactHeaders` keeps only allow-list; `Authorization: Bearer secret123` + `?token=secret456` produces output with neither (telemetry.md §Prohibited content).

**Related**: IDB-1353

### [C-2] Implement: redaction helpers

**File(s)**: `internal/transport/redact.go`
**Depends on**: `[C-1]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `RedactURL`, `RedactHeaders`, `RedactedHeaders` default set.

**Related**: IDB-1353

### [C-3] Test: retry decision function

**File(s)**: `internal/transport/retry_test.go`
**Depends on**: `[A-12]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests covering: retryable {429,500,502,503,504}; non-retryable 4xx (excl. 429 handled separately from 401); transport-error retryable; `Retry-After` integer-seconds + HTTP-date per RFC 7231 §7.1.3; `MaxDelay` clamping; full-jitter formula; ctx-cancel aborts regardless; `MaxAttempts` cap; `Enabled=false` short-circuit.

**Related**: IDB-1353

### [C-4] Implement: retry decision function

**File(s)**: `internal/transport/retry.go`
**Depends on**: `[C-3]`, `[A-12]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `M`

**Goal**: `shouldRetry(ctx, policy, attempt, resp, err) (bool, time.Duration)` + `parseRetryAfter(header, now) (time.Duration, bool)`.

**Related**: IDB-1353

### [C-5] Test: HTTP-status-to-typed-error mapping

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests for every status→error mapping (401→Auth, 403→Authz, 400/422→Validation w/ parsed `FieldErrors`, 404→NotFound, 409→Conflict, 429→RateLimit w/ `RetryAfter`, 5xx→Server, else→Unexpected w/ ≤4 KiB `CauseBody`); `RequestID` extracted from `X-Request-Id` response header.

**Acceptance**: FR-004, SC-008.

**Related**: IDB-1353

### [C-6] Test: auth injection + 401 re-fetch-and-retry-once flow

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[A-2]`, `[A-10]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Failing tests that `Authorization: Bearer <token>` is injected; on 401 the cache is invalidated and provider re-consulted, request retried EXACTLY ONCE (no infinite loop); provider-error → `*AuthenticationError`; 401 handled outside the general retry budget (data-model.md §4.1).

**Related**: IDB-1353

### [C-7] Test: OTel span emission + W3C trace-context propagation

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[A-24]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: `tracetest.InMemoryExporter`-backed failing tests: span name `intelligencecloud.{operationID}`, kind `SpanKindClient`, all attrs per telemetry.md table, status mapping, `retry.attempt_failed` + `retry.giving_up` events, `traceparent`/`tracestate` injection, `intelligencecloud.request_id` from response header.

**Related**: IDB-1353

### [C-8] Test: structured slog logging with trace correlation + redaction

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[A-24]`, `[C-2]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Capture-backed slog tests per telemetry.md §Logging: DEBUG request-start/response-received, WARN retry-sleep, ERROR give-up; `trace_id`/`span_id` injection; bearer token + `Authorization` value NEVER present; no-logger = silent.

**Related**: IDB-1353

### [C-9] Test: User-Agent injection + request body replay across retries

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Failing tests that every request carries configured User-Agent; non-seekable body is buffered before first attempt so retries replay identical bytes; `bytes.Reader` body retries cleanly; nil body fine.

**Related**: IDB-1353

### [C-10] Test: ctx cancellation + deadline override retry

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[A-12]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Cancellation during backoff sleep → `context.Canceled` (wrapped so `errors.Is` works); short deadline mid-sequence → `context.DeadlineExceeded`; NO goroutine leaks.

**Acceptance**: FR-009a.

**Related**: IDB-1353

### [C-11] Implement: Transport `RoundTripper` (auth + retry + OTel + slog + redaction + error mapping + 401 re-fetch + UA + RequestID)

**File(s)**: `internal/transport/transport.go`
**Depends on**: `[C-2]`, `[C-4]`, `[C-5]`, `[C-6]`, `[C-7]`, `[C-8]`, `[C-9]`, `[C-10]`, `[A-2]`, `[A-10]`, `[A-12]`, `[A-24]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `L`

**Goal**: Single `http.RoundTripper` wrapping base transport, composing all concerns. Per-attempt flow: UA → provider.Token → inject `Authorization` → propagate trace-context → base.RoundTrip → capture X-Request-Id → retry decision → on final, map to typed error. 401 re-fetch bounded to 1. Body buffering for retry safety. Context deadline wins over retry budget. FR-008.

**Related**: IDB-1353

### [C-12] Integration test: end-to-end round-trip via in-memory OTel exporter

**File(s)**: `internal/transport/integration_test.go`
**Depends on**: `[C-11]`
**Kind**: `test`
**Priority**: `P1`
**Effort**: `M`

**Goal**: 5 scenarios: (1) happy path; (2) 503×2 then 200; (3) retries exhausted; (4) 401 re-fetch success; (5) redaction enforcement (secret/token/api_key never leaks into span/event/log). Constitution III.

**Related**: IDB-1353

### [C-13] Bench: zero-cost no-tracer guarantee

**File(s)**: `internal/transport/benchmark_test.go`
**Depends on**: `[C-11]`
**Kind**: `bench`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `BenchmarkClient_NoTracer` (< 100 ns/op extra overhead, 0 allocs per `b.ReportAllocs()`) vs `BenchmarkClient_WithTracer` baseline. Per R5 + telemetry.md §Zero-cost no-op contract.

**Related**: IDB-1353

### [C-14] Test: `ValidationError.FieldErrors` parsing on 400/422

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[A-10]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Failing tests for JSON error bodies with multi-field `errors[]` arrays; raw/empty bodies fall back to empty `FieldErrors`.

**Related**: IDB-1353

### [C-15] Test: concurrent safety (`-race`)

**File(s)**: `internal/transport/transport_test.go`
**Depends on**: `[C-11]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: 100 goroutines sharing one transport + mixed-status responses; no data races; token cache not corrupted under concurrent 401 storms.

**Related**: IDB-1353

### [C-16] Refactor: extract operation-context plumbing

**File(s)**: `internal/transport/context.go`, `internal/transport/context_test.go`
**Depends on**: `[C-11]`
**Kind**: `refactor`
**Priority**: `P3`
**Effort**: `S`

**Goal**: `WithOperationContext(ctx, OperationContext) / OperationContextFromContext(ctx)` inside `internal/transport`; generated client layer annotates requests so transport can name spans. May promote to P1 + reorder if Stream B needs it earlier.

**Related**: IDB-1353

### Stream C · Cross-stream dependencies incoming/outgoing
- Requires [A-2], [A-10], [A-12], [A-24].
- Consumed by Stream B resource wrappers (via shared `*http.Client` that Stream A [A-21] installs the transport on) and by Stream D CLI integration tests.

---

## Stream D · `icctl` CLI

All CLI code under `cmd/icctl/`. Subprocess-based integration tests spawn `icctl` against `httptest.Server`.

### [D-1] Test: `SecretStore` interface + keychain-first-file-fallback

**File(s)**: `cmd/icctl/profile/secret_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Mock-backend failing tests: keychain success bypasses file; keychain `ErrNotFound`/`ErrUnsupportedPlatform`/D-Bus-error triggers file fallback; `Delete` removes from both; keychain delete failure logged as warning but not fatal (data-model.md §2.1). Credential JSON-encoded as `{"access_token","expires_at"}`.

**Related**: IDB-1353

### [D-2] Implement: `SecretStore` with keychain + file backends

**File(s)**: `cmd/icctl/profile/secret.go`, `cmd/icctl/profile/keychain.go`, `cmd/icctl/profile/file.go`
**Depends on**: `[D-1]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: `zalando/go-keyring` wrapper (`service=icctl`, `account=<profile>`); file fallback `$XDG_CONFIG_HOME/icctl/credentials.yaml` `0600`. FR-012a.

**Related**: IDB-1353

### [D-3] Test: `Profile` store + `config.yaml` perms + name validation

**File(s)**: `cmd/icctl/profile/store_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: `t.TempDir`-based failing tests: `Save` writes `0600`; round-trip of all non-secret fields; `Name` regex `^[a-z0-9][a-z0-9-]*$`; `SetDefault` updates `current_profile`; `List` without secrets; `Remove`; no token material EVER in config.yaml.

**Related**: IDB-1353

### [D-4] Implement: `Profile` store

**File(s)**: `cmd/icctl/profile/store.go`
**Depends on**: `[D-3]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: YAML (`gopkg.in/yaml.v3`), parent dir `0700`, file `0600`, default `current_profile = "default"`.

**Related**: IDB-1353

### [D-5] Test: output formatters (JSON, table, error)

**File(s)**: `cmd/icctl/output/formatter_test.go`, `cmd/icctl/output/json_test.go`, `cmd/icctl/output/table_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Failing tests: single-entity JSON; list JSON `{items, page_info}`; JSON error → stderr matching documented schema; table w/ tablewriter; UTF-8 → ASCII fallback; ✓/✗ booleans; factory selects by `--output`.

**Related**: IDB-1353

### [D-6] Implement: formatters + error-to-exit-code mapper

**File(s)**: `cmd/icctl/output/formatter.go`, `cmd/icctl/output/json.go`, `cmd/icctl/output/table.go`, `cmd/icctl/output/error.go`
**Depends on**: `[D-5]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Error output → stderr (never stdout); exit-code mapping via `errors.Is`/`errors.As`. FR-014.

**Related**: IDB-1353

### [D-7] Test: `DestructivePreview` rendering (table + JSON)

**File(s)**: `cmd/icctl/confirm/preview_test.go`
**Depends on**: —
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Failing tests: table preview → stderr w/ env+url/op/resource-type+id/irreversible/attrs; JSON plan → stdout BEFORE prompt; NO API call. FR-015a.

**Related**: IDB-1353

### [D-8] Test: confirmation prompt (TTY, `--yes`, env, non-TTY refusal)

**File(s)**: `cmd/icctl/confirm/prompt_test.go`
**Depends on**: `[D-7]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: TTY `y`/`N`; `--yes` bypass; `ICCTL_ASSUME_YES=1|true|yes`; non-TTY without bypass → exit 2, no API call. FR-015a, cli-schema.md.

**Related**: IDB-1353

### [D-9] Implement: `DestructivePreview` + prompt

**File(s)**: `cmd/icctl/confirm/preview.go`, `cmd/icctl/confirm/prompt.go`
**Depends on**: `[D-7]`, `[D-8]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: TTY detection via `golang.org/x/term.IsTerminal`; prompt → stderr so stdout stays parseable; short-circuit on bypass.

**Related**: IDB-1353

### [D-10] Test: root cmd global flags + 4-layer precedence

**File(s)**: `cmd/icctl/cmd/root_test.go`
**Depends on**: `[D-1]`, `[D-3]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `L`

**Goal**: All 9 global flags per cli-schema.md §Global flags; flag > env > profile > default precedence; unknown flag → exit 2.

**Related**: IDB-1353

### [D-11] Test: `version` subcommand

**File(s)**: `cmd/icctl/cmd/version_test.go`
**Depends on**: `[E-24]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Failing tests for semver + commit + go runtime + os/arch in both table and JSON forms; exit 0.

**Related**: IDB-1353

### [D-12] Implement: root cmd + `version` + main

**File(s)**: `cmd/icctl/main.go`, `cmd/icctl/cmd/root.go`, `cmd/icctl/cmd/version.go`
**Depends on**: `[D-10]`, `[D-11]`, `[D-2]`, `[D-4]`, `[E-24]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `L`

**Goal**: Cobra root w/ viper bindings; `PersistentPreRunE` builds `State` (data-model.md §2.2); version consumes `internal/version` (ldflags-injected).

**Related**: IDB-1353

### [D-13] Test: `tui/client.go` — build SDK `*Client` from CLI State

**File(s)**: `cmd/icctl/tui/client_test.go`
**Depends on**: `[A-21]`, `[A-4]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Failing tests that `BuildClient(state)` wires `EffectiveBaseURL`/`EffectiveToken`; timeout as ctx wrapper; request-id via `CallOption`; `--log-level`→`slog.Logger`; empty token → exit 3.

**Related**: IDB-1353

### [D-14] Implement: `tui/client.go`

**File(s)**: `cmd/icctl/tui/client.go`
**Depends on**: `[D-13]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Uses `auth.StaticToken` for v0.1; applies SDK options.

**Related**: IDB-1353

### [D-15] Test: exit-code mapper (all codes 0–7, 124, 130)

**File(s)**: `cmd/icctl/output/error_test.go`
**Depends on**: `[A-10]`, `[C-11]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Dedicated tests for each documented exit code (cli-schema.md §Exit codes).

**Related**: IDB-1353

### [D-16] Implement: exit-code mapper

**File(s)**: `cmd/icctl/output/error.go`
**Depends on**: `[D-15]`, `[D-6]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `S`

**Goal**: `errors.Is`/`errors.As`-driven classification; non-TTY destructive-refusal → exit 2.

**Related**: IDB-1353

### [D-17] Test: `profile` subcommand tree (list/show/add/set-default/set-token/remove/test)

**File(s)**: `cmd/icctl/cmd/profile_test.go`
**Depends on**: `[D-4]`, `[D-2]`, `[D-12]`, `[D-14]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `L`

**Goal**: Subprocess tests per cli-schema.md `profile` subtree; `--token-stdin`; never reveals tokens; `profile test` calls `Me.Get`; invalid name → exit 2; nonexistent → exit 3; always-`0600` after writes.

**Related**: IDB-1353

### [D-18] Implement: `profile` subcommand tree

**File(s)**: `cmd/icctl/cmd/profile.go`
**Depends on**: `[D-17]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `L`

**Related**: IDB-1353

### [D-19] Test: `resellers` subcommand (reference resource)

**File(s)**: `cmd/icctl/cmd/resellers_test.go`
**Depends on**: `[D-12]`, `[D-6]`, `[D-9]`, `[D-14]`, `[B-12]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `L`

**Goal**: Subprocess tests for all verbs; both output formats; `--page-size`/`--max-items`; `--from-file` + flag-field creation/update; `delete` TTY-prompt/ --yes/non-TTY-refused; `delete --output json` emits plan BEFORE prompt; `deactivate` second destructive-verb validation; `reactivate` NOT destructive; `audit-logs --since/--until`.

**Related**: IDB-1353

### [D-20] Implement: `resellers` subcommand

**File(s)**: `cmd/icctl/cmd/resellers.go`
**Depends on**: `[D-19]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `L`

**Related**: IDB-1353

### [D-21] Test + impl: `me` subcommand

**File(s)**: `cmd/icctl/cmd/me_test.go`, `cmd/icctl/cmd/me.go`
**Depends on**: `[D-12]`, `[D-6]`, `[D-14]`, `[B-10]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: `get`, `update-notification-prefs`, `upload-avatar` (multipart); missing arg → exit 2.

**Related**: IDB-1353

### [D-22] Test + impl: `companies` subcommand

**File(s)**: `cmd/icctl/cmd/companies_test.go`, `cmd/icctl/cmd/companies.go`
**Depends on**: `[D-12]`, `[D-6]`, `[D-9]`, `[D-14]`, `[B-23]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: list/get/create/update/delete (destructive)/connectors. NB: depends on future operations from IDB-1354; until then, most verbs are scaffolded.

**Related**: IDB-1353, IDB-1354

### [D-23] Test + impl: `locations` subcommand

**File(s)**: `cmd/icctl/cmd/locations_test.go`, `cmd/icctl/cmd/locations.go`
**Depends on**: `[D-12]`, `[D-6]`, `[D-9]`, `[D-14]`, `[B-23]`, `[B-18]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `M`

**Goal**: list/get/create/update/delete/connectors.

**Related**: IDB-1353, IDB-1354

### [D-24] Test + impl: remaining resource subcommands (`products`, `verticals`, `connectors`, `users`, `audit-logs`)

**File(s)**: `cmd/icctl/cmd/{products,verticals,connectors,users,auditlogs}{,_test}.go`
**Depends on**: `[D-12]`, `[D-6]`, `[D-9]`, `[D-14]`, `[B-16]`, `[B-18]`, `[B-20]`, `[B-22]`, `[B-23]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `L`

**Goal**: Per-resource list/get + (where defined) create/update/delete/search; destructive verbs go through shared `confirm/` package; `audit-logs search --since/--until`.

**Related**: IDB-1353, IDB-1354

### [D-25] Test: shell completion (bash, zsh, fish, powershell)

**File(s)**: `cmd/icctl/cmd/completion_test.go`
**Depends on**: `[D-12]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Output non-empty + shell-specific markers; `bash -n` / `zsh -n` / `fish --no-execute` parse-check where available.

**Related**: IDB-1353

### [D-26] Implement: `completion` subcommand

**File(s)**: `cmd/icctl/cmd/completion.go`
**Depends on**: `[D-25]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Cobra `Gen*Completion` wrappers; `--help` example per shell.

**Related**: IDB-1353

### [D-27] Integration test: `--help` contract enforcement (whole tree)

**File(s)**: `cmd/icctl/cmd/help_test.go`
**Depends on**: `[D-18]`, `[D-20]`, `[D-21]`, `[D-22]`, `[D-23]`, `[D-24]`, `[D-26]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Walks tree; every leaf has `Use`, `Short`, `Long` ≥1 sentence, `Example`, flags table. FR-015.

**Related**: IDB-1353

### [D-28] Test: token-never-logged invariant

**File(s)**: `cmd/icctl/cmd/security_test.go`
**Depends on**: `[D-12]`, `[D-14]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Subprocess with `--log-level debug --token <sentinel>` against mock server; assert sentinel ABSENT from captured stderr (even in JSON error output).

**Related**: IDB-1353

### [D-29] Integration test: quickstart Part B reproduction

**File(s)**: `cmd/icctl/cmd/quickstart_integration_test.go`
**Depends on**: `[D-18]`, `[D-20]`, `[D-9]`, `[D-14]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `L`

**Goal**: `//go:build integration`; reproduces quickstart.md Part B end-to-end against `httptest.Server`; exit codes match cli-schema.md. Keeps SC-002 promise honest.

**Related**: IDB-1353

### [D-30] Test: file-permissions audit

**File(s)**: `cmd/icctl/profile/permissions_test.go`
**Depends on**: `[D-2]`, `[D-4]`
**Kind**: `test`
**Priority**: `P2`
**Effort**: `S`

**Goal**: After every write: config.yaml + credentials.yaml = `0600`; parent dir = `0700`; perms survive rewrite.

**Related**: IDB-1353

### [D-31] Impl: SIGINT handling

**File(s)**: `cmd/icctl/cmd/signal_test.go`, `cmd/icctl/main.go`
**Depends on**: `[D-12]`
**Kind**: `impl`
**Priority**: `P2`
**Effort**: `S`

**Goal**: `signal.NotifyContext`; SIGINT → exit 130, in-flight requests cancelled.

**Related**: IDB-1353

### [D-32] Docs: GoDoc on all exported symbols under `cmd/icctl/`

**File(s)**: `cmd/icctl/{profile,output,confirm,tui}/*.go`
**Depends on**: `[D-2]`, `[D-4]`, `[D-6]`, `[D-9]`, `[D-14]`
**Kind**: `docs`
**Priority**: `P3`
**Effort**: `S`

**Goal**: `go vet` clean; `revive`'s `exported` rule passes.

**Related**: IDB-1353

### Stream D · Cross-stream dependencies incoming
- [A-21] client + options + auth adapters
- [B-10..B-23] resource services
- [C-11] transport (so CLI can `errors.Is` against typed errors)

---

## Stream E · Release, CI, documentation, semver automation

Release engineering, CI gates, supply-chain controls, user-facing docs, ADRs, and **auto-incrementing semver** via release-please + ldflags-injected build-time version.

### [E-1] Config: `go.mod` + module path + minimum Go version

**File(s)**: `go.mod`, `go.sum`
**Depends on**: —
**Kind**: `config`
**Priority**: `P1`
**Effort**: `S`

**Goal**: `module github.com/tresic-cloud/intelligence-cloud-go`; `go 1.22`; `toolchain go1.23.x`. FR-016, SC-005.

**Related**: IDB-1353

### [E-2] Config: `.golangci.yaml`

**File(s)**: `.golangci.yaml`
**Depends on**: `[E-1]`
**Kind**: `config`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Enables `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `misspell`, `revive` (with `exported` rule ON for GoDoc enforcement), `gosec`. FR-019, FR-022, constitution VI/VII.

**Related**: IDB-1353

### [E-3] CI workflow — gated pipeline (lint → security → static → unit → coverage → build-matrix)

**File(s)**: `.github/workflows/ci.yaml`
**Depends on**: `[E-1]`, `[E-2]`
**Kind**: `config`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Triggers on PR + push-to-main. Pipeline in strict order, each gate blocking the next:

1. **lint**: `golangci-lint run ./...`
2. **security**: `govulncheck ./...`
3. **static**: `go vet ./...`, `go build ./...` (typecheck), `staticcheck ./...`
4. **unit tests**: `go test -race -coverprofile=coverage.out ./...`; Go matrix `[1.22.x, 1.23.x]`
5. **coverage gate**: parse `coverage.out`; fail if <80% total lines
6. **codegen-drift**: run `scripts/codegen.sh` + `scripts/codegen_test.sh`; fail on diff
7. **example-build**: `go build ./examples/...`
8. **cross-platform build sanity**: `GOOS={linux,darwin,windows} GOARCH={amd64,arm64} go build ./cmd/icctl/` (6 combinations) — catches platform-specific breakage before release
9. **integration** (optional job): `go test -tags=integration ./...` gated behind `secrets.IC_STAGING_TOKEN` (not a required check)

All non-integration jobs are required status checks for merging. Caching enabled. Constitution II/III/VI, FR-020/022, SC-003.

**Related**: IDB-1353

### [E-4] Config: `Makefile` with developer targets

**File(s)**: `Makefile`
**Depends on**: `[E-1]`, `[E-2]`
**Kind**: `config`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Targets `build`, `test`, `lint`, `vuln`, `static`, `codegen`, `pull-openapi`, `coverage`, `bench`, `release-snapshot`, `docs`, `clean`, `help` (default). Each phony; `help` prints descriptions.

**Related**: IDB-1353

### [E-5] Config: Dependabot (weekly Go modules + Actions)

**File(s)**: `.github/dependabot.yaml`
**Depends on**: —
**Kind**: `config`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Weekly `gomod` + `github-actions` ecosystems targeting `main`; labels `dependencies`; open-PR cap 10.

**Related**: IDB-1353

### [E-6] Docs: ADR-0001 · oapi-codegen

**File(s)**: `docs/adr/0001-oapi-codegen.md`
**Depends on**: —
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Extract R1 verbatim + alternatives. Constitution V.

**Related**: IDB-1353

### [E-7] Docs: ADR-0002 · Cobra + Viper

**File(s)**: `docs/adr/0002-cobra-viper-cli.md`
**Depends on**: —
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Extract R2.

**Related**: IDB-1353

### [E-8] Docs: ADR-0003 · OS keychain

**File(s)**: `docs/adr/0003-os-keychain-storage.md`
**Depends on**: —
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Extract R3 + compliance rationale (constitution XI).

**Related**: IDB-1353

### [E-9] Docs: ADR-0004 · Pre-1.0 experimental-API policy

**File(s)**: `docs/adr/0004-pre-1.0-experimental-policy.md`
**Depends on**: —
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Extract R10.

**Related**: IDB-1353

### [E-10] Config: GitHub community files (CODEOWNERS, PR/issue templates)

**File(s)**: `.github/CODEOWNERS`, `.github/pull_request_template.md`, `.github/ISSUE_TEMPLATE/bug.md`, `.github/ISSUE_TEMPLATE/feature.md`
**Depends on**: —
**Kind**: `config`
**Priority**: `P3`
**Effort**: `S`

**Goal**: CODEOWNERS `@tresic-cloud/platform` (placeholder — user confirmation required). Issue templates in YAML front-matter.

**Review gate**: team-slug confirmation.

**Related**: IDB-1353

### [E-11] Docs: `SECURITY.md` disclosure policy

**File(s)**: `SECURITY.md`
**Depends on**: —
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Supported versions (latest `v0.x`); private reporting via GitHub Security Advisories (confirm with user); response SLAs; link to cosign verification (R12).

**Review gate**: reporting-channel confirmation.

**Related**: IDB-1353

### [E-12] Docs: `README.md`

**File(s)**: `README.md`
**Depends on**: `[E-1]`, `[E-3]`
**Kind**: `docs`
**Priority**: `P1`
**Effort**: `M`

**Goal**: Badges (Go ver, CI, pkg.go.dev, license); `go get` install line; SDK + CLI quickstarts adapted from quickstart.md; links to architecture + cli-reference + GoDoc; prereqs; contributing section. FR-019, SC-002.

**Related**: IDB-1353

### [E-13] Docs: `docs/quickstart.md` + `docs/architecture.md`

**File(s)**: `docs/quickstart.md`, `docs/architecture.md`
**Depends on**: `[E-6]`, `[E-7]`, `[E-8]`, `[E-9]`
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Maintained copy of spec quickstart; architecture doc includes R11 compliance controls matrix (constitution XI) + error model + observability contract summary + stability policy with ADR links.

**Decision recorded**: `docs/quickstart.md` is a maintained copy (NOT a symlink) — symlinks break on GitHub Markdown viewer and some CI runners.

**Related**: IDB-1353

### [E-14] Docs: `CONTRIBUTING.md`

**File(s)**: `CONTRIBUTING.md`
**Depends on**: `[E-4]`, `[E-23]`
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Dev setup; test/lint/cover/bench commands; **conventional-commits** spec (see [E-23] release-please); constitution Git-Branch-Naming section; release checklist; review expectations.

**Related**: IDB-1353

### [E-15] Docs: `CHANGELOG.md` (Keep-a-Changelog, managed by release-please)

**File(s)**: `CHANGELOG.md`
**Depends on**: `[E-23]`
**Kind**: `docs`
**Priority**: `P3`
**Effort**: `S`

**Goal**: Seed file (release-please will maintain). Note at top: "Changelog is auto-generated from conventional commits by release-please. Do not edit by hand between releases."

**Related**: IDB-1353

### [E-16] Docs: `LICENSE` (REVIEW GATE)

**File(s)**: `LICENSE`
**Depends on**: —
**Kind**: `docs`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Canonical OpenAPI declares "Proprietary" but repo path (`github.com/tresic-cloud/…`) is structured for public Go-module consumption. Options: (a) Proprietary EULA, (b) Apache 2.0, (c) MIT, (d) BSL. Record compliance rationale (constitution XI).

**⚠️ BLOCKS release tasks [E-17], [E-18], [E-22], [E-23], [E-26] until user confirms license.**

**Related**: IDB-1353

### [E-17] Config: `.goreleaser.yaml` (cross-platform builds + ldflags version injection)

**File(s)**: `.goreleaser.yaml`
**Depends on**: `[E-1]`, `[E-16]`, `[E-24]`
**Kind**: `config`
**Priority**: `P3`
**Effort**: `M`

**Goal**: Build `cmd/icctl` for `{linux,darwin,windows} × {amd64,arm64}`. Archives: `tar.gz` (Unix), `zip` (Win). SHA-256 checksums. Name: `icctl_{version}_{os}_{arch}.{ext}`. **Ldflags inject `internal/version.Version`, `internal/version.Commit`, `internal/version.Date`** from git tag / SHA / build time. Snapshot builds via `make release-snapshot`. Pre-hook: `go mod tidy`. Brew tap section present but commented. FR-017, SC-005, R12.

**Related**: IDB-1353

### [E-18] CI: `.github/workflows/release.yaml` (goreleaser + cosign keyless)

**File(s)**: `.github/workflows/release.yaml`
**Depends on**: `[E-3]`, `[E-17]`, `[E-23]`
**Kind**: `config`
**Priority**: `P3`
**Effort**: `M`

**Goal**: Triggered by **release-please** creating a tag (or direct `v*` tag push as fallback). Runs goreleaser; signs each artefact with `cosign sign-blob` via GitHub OIDC keyless; emits SLSA L2+ provenance; creates GitHub Release. `id-token: write` permission. Documented verify command: `cosign verify-blob --certificate-identity-regexp '^https://github.com/tresic-cloud/intelligence-cloud-go/'`. Confirms `go get github.com/tresic-cloud/intelligence-cloud-go@vX.Y.Z` resolves. R12, constitution XI, FR-016, FR-017, FR-018, SC-005.

**Related**: IDB-1353

### [E-19] CI: scheduled OpenAPI drift-detection workflow

**File(s)**: `.github/workflows/openapi-drift.yaml`
**Depends on**: `[E-3]`, `[B-7]`
**Kind**: `config`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Daily cron. Fetches `intelligence-cloud/main/backend/docs/api/openapi.yaml`; diff vs `testdata/openapi.yaml`; opens PR on drift labelled `openapi-drift automated`; updates existing PR instead of duplicating. **Never auto-merges.** R9, SC-007.

**Related**: IDB-1353, IDB-1354

### [E-20] CI: markdownlint + broken-link + example-compile (docs quality)

**File(s)**: `.github/workflows/ci.yaml` (extend), `.markdownlint.yaml`
**Depends on**: `[E-3]`, `[E-12]`, `[E-13]`
**Kind**: `config`
**Priority**: `P3`
**Effort**: `S`

**Goal**: markdownlint-cli2 on all `.md`; link-check internal links; example-compile is the SC-002 guard (quickstart breaks → CI goes red).

**Related**: IDB-1353

### [E-21] Script: `make docs` — generate CLI reference from cobra

**File(s)**: `Makefile` (extend), `docs/cli-reference.md`
**Depends on**: `[E-4]`, `[D-26]`
**Kind**: `script`
**Priority**: `P3`
**Effort**: `S`

**Goal**: `go run cmd/icctl/main.go --generate-docs docs/` → cobra `doc.GenMarkdownTree`. CI drift-check (same pattern as codegen-drift).

**Related**: IDB-1353

### [E-22] CI: post-release verification job

**File(s)**: `.github/workflows/release.yaml` (extend)
**Depends on**: `[E-18]`
**Kind**: `test`
**Priority**: `P3`
**Effort**: `M`

**Goal**: Triggered by `release` event. In clean container: (1) `go get ...@<tag>` succeeds; (2) all 6 artefacts present with matching SHA-256; (3) every artefact has `.sig`; (4) `cosign verify-blob --certificate-identity-regexp '...'` succeeds. Post failure as an issue. R12, SC-005, FR-016/017.

**Related**: IDB-1353

### [E-23] CI: **release-please** — conventional-commit-driven auto-semver

**File(s)**: `.github/workflows/release-please.yaml`, `release-please-config.json`, `.release-please-manifest.json`
**Depends on**: `[E-1]`, `[E-16]`
**Kind**: `config`
**Priority**: `P2`
**Effort**: `M`

**Goal**: Google's `release-please-action` runs on every push to `main` and:

1. Parses conventional commits since last tag (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `perf:`, `test:`, `build:`, `ci:`, `style:`, `BREAKING CHANGE:` footer).
2. Bumps semver: `feat` → MINOR; `fix` → PATCH; `BREAKING CHANGE` or `!` suffix → MAJOR.
3. Opens/updates a "Release PR" with generated `CHANGELOG.md` update + version bump in `.release-please-manifest.json`.
4. **Merging the Release PR** tags `vX.Y.Z` on `main` → triggers [E-18] release workflow.

**Config**:
- `release-type: go` (handles Go module versioning)
- `package-name: intelligence-cloud-go`
- `bump-minor-pre-major: true` (so `v0.x` breaking changes still MINOR-bump pre-1.0, aligning with R10)
- `include-v-in-tag: true`
- `changelog-sections` mapped to Keep-a-Changelog categories

**Acceptance**:
- Conventional-commit compliance documented in `CONTRIBUTING.md` ([E-14])
- PR title lint action (`amannn/action-semantic-pull-request`) optionally enforces conventional commits on PRs — or the release-please `include-commits` scope controls which commits count.
- Dry-run verified on a throwaway branch before first real tag.

**Satisfies user request**: auto-increment version number.

**Related**: IDB-1353

### [E-24] Impl: `internal/version` package — ldflags-injected build-time version

**File(s)**: `internal/version/version.go`, `internal/version/version_test.go`
**Depends on**: `[E-1]`
**Kind**: `impl`
**Priority**: `P1`
**Effort**: `S`

**Goal**: Internal (not public-API) package exposing:

```go
package version

var (
    Version = "dev"       // set by ldflags at build time
    Commit  = "unknown"   // set by ldflags: git rev-parse HEAD
    Date    = "unknown"   // set by ldflags: RFC 3339 UTC build time
)

// String returns "vX.Y.Z (abc1234, 2026-04-13T10:00:00Z)"
func String() string { ... }
```

**Ldflags** (invoked from `.goreleaser.yaml` + `Makefile` `build`):

```
-ldflags "-s -w \
  -X github.com/tresic-cloud/intelligence-cloud-go/internal/version.Version={{.Tag}} \
  -X github.com/tresic-cloud/intelligence-cloud-go/internal/version.Commit={{.FullCommit}} \
  -X github.com/tresic-cloud/intelligence-cloud-go/internal/version.Date={{.CommitDate}}"
```

**Test**: verify fallback defaults when no ldflags (`go run`) produce `"dev"`, `"unknown"`, `"unknown"`; verify `String()` format.

**Consumers**:
- [A-26] default User-Agent
- [D-12] `icctl version` subcommand
- [D-11] version-subcommand test

**Satisfies user request**: version embedded in artifact at build time.

**Related**: IDB-1353

### [E-25] Config: PR conventional-commit lint action

**File(s)**: `.github/workflows/ci.yaml` (extend)
**Depends on**: `[E-3]`, `[E-23]`
**Kind**: `config`
**Priority**: `P3`
**Effort**: `S`

**Goal**: `amannn/action-semantic-pull-request@v5` runs on PRs, enforcing conventional-commit PR titles (which become squash-commit messages on main). Types: `feat|fix|docs|chore|refactor|perf|test|build|ci|style`. Scopes optional.

**Related**: IDB-1353

### [E-26] CI: cross-platform build sanity in main CI (extracted to named reusable workflow)

**File(s)**: `.github/workflows/build-matrix.yaml`
**Depends on**: `[E-3]`, `[E-17]`
**Kind**: `config`
**Priority**: `P2`
**Effort**: `S`

**Goal**: Reusable workflow invoked by `ci.yaml` step 8. Matrix: `{linux,darwin,windows} × {amd64,arm64}`. `go build -ldflags "..." ./cmd/icctl/`. Ensures a PR that breaks Windows or arm64 is caught before merge (not at release time).

**Satisfies user request**: "build for all go platforms" enforced on every PR (not only at release).

**Related**: IDB-1353

### Stream E · Cross-stream dependencies incoming
- [B-6] `scripts/codegen.sh` for the CI codegen-drift gate in [E-3]
- [B-7] `scripts/pull-openapi.sh` + testdata for [E-19] drift workflow
- [A..D tests] must exist for coverage-gate to be meaningful in [E-3]
- [D-26] `completion` subcommand for [E-21] `make docs`
- [D-12] cobra tree for [E-21] reference generation
- [A-21] SDK compiles so example-build and cross-platform-build gates pass

### Stream E · Review gates requiring user input
1. **LICENSE choice** ([E-16]) — BLOCKS release. Default proposal: **Apache 2.0** (matches other Go SDKs and is the likely intent of a public-github-path client library; overrides canonical OpenAPI "Proprietary" designation because the SDK code is a distinct artefact).
2. **CODEOWNERS team slug** ([E-10]) — confirm `@tresic-cloud/platform` exists or replace.
3. **Security reporting channel** ([E-11]) — default proposal: GitHub Security Advisories.
4. **Milestone name** — confirmed `v0.1.0` (from your prior input).

---

## Cross-stream dependency graph (summary)

```
A (SDK core, foundational)
    │
    ├──▶ B (codegen + resource wrappers): uses A.errors, A.Client, A.Iterator, A.Options
    │
    ├──▶ C (transport): uses A.errors, A.auth, A.RetryPolicy, A.telemetry-helpers
    │         │
    │         └──▶ B (resource wrappers gain error-mapping via transport layer on Client.httpClient)
    │
    └──▶ D (CLI): uses A.Client, A.auth adapters; routes errors via transport's typed classification

E (release, CI, docs, semver automation)
    │
    ├──▶ Gates A, B, C, D (CI lint+vuln+static+unit+coverage from day one)
    ├──▶ Consumes B.codegen scripts, D.completion, D.cobra-tree
    └──▶ [E-24] internal/version populates A.UserAgent + D.version
```

## Parallelisation plan — agent-team execution

Use the `superpowers:dispatching-parallel-agents` skill + Plan/general-purpose agents. Waves below can each be done by multiple agents in parallel. Inside a wave, cross-dependencies are absent.

### Wave 0 — Foundation (can run immediately, all parallel)

Dispatch **4 parallel agents**:
- Agent W0-1: `[E-1] go.mod`, `[E-2] golangci`, `[E-24] internal/version` (plus test)
- Agent W0-2: `[A-1/A-2] auth.Token+CredentialProvider`, `[A-3/A-4] StaticToken`, `[A-5/A-6] RefreshFunc`
- Agent W0-3: `[A-7..A-10] errors hierarchy`, `[A-11/A-12] RetryPolicy`, `[A-13/A-14] options`
- Agent W0-4: `[A-15..A-18] pagination Iterator`, `[A-22..A-24] telemetry helpers`, `[C-1/C-2] redaction`

### Wave 1 — Codegen + Client + Transport (depends on Wave 0)

Dispatch **3 parallel agents**:
- Agent W1-1: `[B-4] pin openapi`, `[B-5] oapi-codegen config`, `[B-1/B-2/B-3] codegen tests`, `[B-6] codegen script`, `[B-7] pull-openapi script`, `[B-8] generated doc.go`
- Agent W1-2: `[A-19/A-20/A-21] Client + NewClient`, `[A-25/A-26] UserAgent`, `[A-27/A-28] docs + FieldError test`
- Agent W1-3: `[C-3..C-16] transport` (16 tasks — sized for one dedicated agent given tight coupling within the RoundTripper)

### Wave 2 — Resource wrappers (depends on Wave 1 W1-1 + W1-2)

Dispatch **4 parallel agents** (resource wrappers are all independent of each other):
- Agent W2-1: `[B-9/B-10] Me`, `[B-11/B-12] Resellers`
- Agent W2-2: `[B-13/B-14] Auth`, `[B-15/B-16] AuditLogs`, `[B-17/B-18] Connectors`
- Agent W2-3: `[B-19/B-20] Products`, `[B-21/B-22] Users`, `[B-23] scaffold Co/Loc/Vert`
- Agent W2-4 (parallel with W2-1..3): kick off `[E-3] CI workflow`, `[E-4] Makefile`, `[E-5] Dependabot`, `[E-12] README`, ADRs `[E-6..E-9]`

### Wave 3 — CLI core (depends on W1-2 Client + all W2 wrappers)

Dispatch **3 parallel agents**:
- Agent W3-1: `[D-1..D-9]` — profile store, SecretStore, formatters, destructive preview/prompt
- Agent W3-2: `[D-10/D-11/D-12] root cmd + version`, `[D-13/D-14] tui/client`, `[D-15/D-16] exit-code mapper`, `[D-17/D-18] profile subcommand`
- Agent W3-3: start `[D-25/D-26] completion`, `[D-31] SIGINT`, `[D-32] GoDoc pass`

### Wave 4 — CLI resource commands + integration tests

Dispatch **3 parallel agents** (resource subcommands independent):
- Agent W4-1: `[D-19/D-20] resellers`, `[D-21] me`
- Agent W4-2: `[D-22] companies`, `[D-23] locations`, `[D-24] products+verticals+connectors+users+auditlogs`
- Agent W4-3: `[D-27] help-contract test`, `[D-28] token-never-logged`, `[D-29] quickstart integration`, `[D-30] permissions audit`

### Wave 5 — Release engineering (depends on everything compiling + [E-16] LICENSE confirmed)

Dispatch **2 parallel agents**:
- Agent W5-1: `[E-17] goreleaser`, `[E-18] release workflow`, `[E-23] release-please`, `[E-25] PR title lint`, `[E-26] build-matrix`
- Agent W5-2: `[E-13..E-15] docs`, `[E-19] drift workflow`, `[E-20] md-lint`, `[E-21] make docs`, `[E-22] post-release verify`, `[E-10/E-11] governance`

### Wave 6 — First release

Manual (user-initiated):
1. Confirm LICENSE, CODEOWNERS slug, security-reporting channel.
2. Merge the release-please PR → tag `v0.1.0` auto-pushes.
3. `[E-22]` post-release verification confirms.

---

## Parallelism budget per wave

| Wave | Parallel agents | Sequential-within-wave tasks |
|---|---|---|
| 0 | 4 | ~14 |
| 1 | 3 | ~28 |
| 2 | 4 | ~18 |
| 3 | 3 | ~18 |
| 4 | 3 | ~12 |
| 5 | 2 | ~12 |

**Theoretical minimum wall-clock** if every wave waits for all previous to complete: 6 waves × typical Plan-agent task drafting time. Actual implementation time depends on task effort — most tasks are `S` or `M`.

## Milestone mapping

All 125 tasks target **milestone `v0.1.0`** per the user's confirmation. Post-v0.1 planning will split into v0.2 (post-MVP CLI polish), v0.3 (release polish). For now, every GitHub issue created from this file should be tagged with the `v0.1.0` milestone and the `stream:A|B|C|D|E` label.

## Labels proposal (for `/speckit.taskstoissues`)

| Label | Applied to |
|---|---|
| `stream:sdk-core` | Stream A |
| `stream:codegen` | Stream B |
| `stream:transport` | Stream C |
| `stream:cli` | Stream D |
| `stream:release` | Stream E |
| `priority:p1` / `p2` / `p3` | Per task priority |
| `kind:test` / `impl` / `docs` / `config` / `script` / `bench` / `refactor` | Per task kind |
| `jira:IDB-1353` | All tasks |
| `jira:IDB-1354` | Tasks that also reference IDB-1354 (B-1, B-4, B-7, B-19, D-22/23/24 for spec growth) |

Each issue body should include:
- The task block verbatim (acceptance criteria + dependencies)
- A `Related: IDB-1353` / `IDB-1354` trailer
- Link back to `spec.md` and `plan.md` in the feature directory

---

## Readiness

- [x] All spec FRs (FR-001..022) mapped to one or more tasks
- [x] All spec SCs (SC-001..008) have verification tasks
- [x] All constitution principles satisfied or adapted with rationale
- [x] All research decisions (R1..R12) have implementing tasks
- [x] All contract surfaces (sdk-api.md, cli-schema.md, telemetry.md) have enforcement tests
- [x] Semver auto-increment: [E-23] release-please
- [x] Build-time version injection: [E-24] internal/version + [E-17] goreleaser ldflags
- [x] CI gate order: [E-3] lint → security → static → unit → coverage → cross-platform build
- [x] Release on tag: [E-18] goreleaser + cosign keyless

Ready for `/speckit.taskstoissues` once the user confirms: (a) GitHub repo `tresic-cloud/intelligence-cloud-go` exists, (b) LICENSE choice, (c) CODEOWNERS team slug, (d) security reporting channel.
