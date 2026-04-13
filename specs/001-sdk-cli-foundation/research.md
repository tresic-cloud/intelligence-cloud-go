# Phase 0 Research: Intelligence Cloud Go SDK & CLI

**Date**: 2026-04-13
**Spec**: [spec.md](./spec.md) · **Plan**: [plan.md](./plan.md)

This document resolves every `NEEDS CLARIFICATION` and records the architectural decisions that underpin the plan. Each entry follows **Decision → Rationale → Alternatives considered**.

---

## R1 · OpenAPI code-generator

**Decision**: Use `github.com/oapi-codegen/oapi-codegen/v2` in "types + low-level client" mode. Emit only request/response types and a minimal client with per-operation functions into `internal/generated/`. Hand-write the public surface (`Client`, resource files) on top.

**Rationale**:
- `oapi-codegen` is the reference community generator for Go, actively maintained, and produces idiomatic Go code (no reflection hacks, no init-time registrations).
- The plain "types + stubs" mode produces code that's easy to wrap; "ChiServer" / "EchoServer" modes are not relevant (we're a client).
- Output lives behind an `internal/` boundary so we can change generators later without breaking consumers.
- Aligns with the implicit preference in IDB-1354 which names `oapi-codegen` explicitly as the smoke-test compile target for the consolidated spec.

**Alternatives considered**:
- `OpenAPITools/openapi-generator` — Java-based, slower, less idiomatic Go output, heavier toolchain burden in CI.
- `ogen-go/ogen` — modern, fast, schema-first, but its public-API shape is still evolving and its generated surface bleeds through more than oapi-codegen's.
- Hand-written client — rejected because the canonical OpenAPI has ~21 operations today and will grow as IDB-1354 completes; re-transcription is the exact anti-pattern the spec forbids (FR-003, SC-007).

## R2 · CLI framework

**Decision**: `github.com/spf13/cobra` for command tree and flag parsing; `github.com/spf13/viper` for configuration loading (YAML files, env-var bindings, precedence resolution).

**Rationale**:
- De-facto standard for production Go CLIs (`kubectl`, `gh`, `hugo`, `docker`, `pulumi`).
- Cobra's built-in help generator produces the long-form usage expected by FR-015.
- Viper handles the flag > env > file precedence demanded by FR-012 without hand-rolled config layering.

**Alternatives considered**:
- `urfave/cli/v3` — lighter, but lacks the hierarchical help story Cobra gives for deeply-nested resource verbs.
- `alecthomas/kong` — elegant struct-tag design, smaller ecosystem.
- Stdlib `flag` — rejected; insufficient for the ≥50 subcommands anticipated once the CLI mirrors the full SDK surface.

## R3 · OS keychain integration

**Decision**: `github.com/zalando/go-keyring` for cross-platform secret storage.

**Rationale**:
- Single dependency, no cgo on macOS and Windows; Linux uses the Secret Service D-Bus API (libsecret).
- Minimal, stable API: `Set(service, user, password)` / `Get(service, user)` / `Delete(service, user)`.
- Gracefully returns a typed `ErrNotFound` we can fall back on for the file-based credential store (FR-012a).

**Alternatives considered**:
- `99designs/keyring` — more backends (including plain file, pass, encrypted file), but heavier, multiple indirect dependencies, and several historically stale. Its broader backend matrix is not justified given our need for "OS keychain or `0600` file" — both of which we implement directly.
- `github.com/keybase/go-keychain` — macOS-only.
- Shelling out to `security` / `keyring` / `cmdkey` — fragile on Windows and not fully scriptable.

## R4 · Retry & rate-limit implementation

**Decision**: Implement retry as a custom `http.RoundTripper` layer in `internal/transport/transport.go` that wraps the user-supplied (or default) transport. Policy: exponential backoff starting at 100 ms, doubling, capped at 30 s, with full jitter. Retry on `5xx`, `429`, and transport errors. Honour `Retry-After` (both integer-seconds and HTTP-date forms). Abort on context cancellation or deadline. Never retry non-429 `4xx`. Idempotency is not a gating factor for retry — the backend is responsible for handling duplicate requests safely (the SDK surfaces the `Idempotency-Key` header pass-through via a request option for callers who need exactly-once guarantees).

**Rationale**:
- A RoundTripper is the canonical Go idiom for cross-cutting HTTP concerns; it composes cleanly with user-supplied transports (including OTel's `otelhttp.NewTransport`).
- Full jitter (as opposed to "equal jitter" or no jitter) is proven in the AWS Architecture Blog's retry study to minimise client thundering-herd; it is cheap.
- Spec clarification Q1 (FR-009a) already approved this shape.

**Alternatives considered**:
- `github.com/hashicorp/go-retryablehttp` — full-featured but pulls in opinionated logger and clones the request body; our needs are narrower and we want to control body re-reads carefully.
- Retry at the SDK method layer (wrapping each generated call) — rejected; bloats hand-written code surface (~21+ methods would each carry identical retry scaffolding) and is harder to test in one place.

## R5 · OpenTelemetry integration pattern

**Decision**: SDK holds a `trace.TracerProvider` reference (default: `trace.NewNoopTracerProvider()`) injected via `WithTracerProvider(tp)` option. The transport layer starts a span named `intelligencecloud.{operationID}` around every outbound request, applies the W3C trace-context propagator to inject `traceparent`/`tracestate` headers, and records attributes per OpenTelemetry's semantic conventions for HTTP client spans (1.24): `http.request.method`, `url.template`, `url.full` (with secrets redacted), `http.response.status_code`, `network.protocol.name`, plus SDK-specific `intelligencecloud.request_id` and `intelligencecloud.retry_attempt`. When the tracer provider is a no-op, the span start/end calls compile to cheap no-ops (benchmarked).

**Rationale**:
- The `trace` API is stable (1.0+); consumers can wire any exporter (Datadog, Honeycomb, Jaeger, OTLP) without the SDK caring.
- Semantic-convention compliance means Datadog (Tresic's APM backend) auto-classifies our spans as HTTP client calls without custom processors.
- Spec clarification Q3 (FR-008a) already approved this shape.

**Alternatives considered**:
- Embed `otelhttp.NewTransport` directly — simpler but opinionated about span naming (`HTTP {method}`) and doesn't know operation IDs. We wrap instead of delegate.
- Datadog-native tracer (`gopkg.in/DataDog/dd-trace-go.v1`) — locks out other tracers and forces a runtime dep on `dd-trace-go`'s large module tree. Rejected per Q3 discussion.
- No tracing in v1 — rejected; observability is a constitution principle, and retro-fitting tracing into a released SDK is a breaking change to transport internals.

## R6 · Observability adaptation (Constitution IV)

**Decision**: The backend-oriented Prometheus `InstrumentedService` / `InstrumentedRepository` decorator pattern from the Tresic observability standard does **not** apply verbatim to a client-side SDK. The SDK's observability contract is:
- **Tracing**: OTel client spans per R5 (equivalent to the `ErrorContext`/`InfoContext` requirement — trace-correlated).
- **Logging**: `slog.Logger` hook with `traceId` attribute auto-injected from span context.
- **Metrics**: deferred to consumer instrumentation via the OTel `metric` API or user-supplied middleware — the SDK does not emit Prometheus metrics directly because it has no service boundary of its own. Consumers who want Prometheus counters wrap the SDK transport with their existing service-level metrics.
- **Business error classification**: typed error hierarchy (per R7) fulfils the "specific error types" requirement.

**Rationale**:
- A single SDK instance may be called from many services with different service names; forcing a service-level Prometheus gauge would be wrong.
- OTel spans carry enough information for downstream OTel-to-Prom converters to recover all the dimensions the Prom-decorator pattern provides.
- Consumer ownership of metrics is the Datadog-recommended model for client libraries.

**Compliance note**: This adaptation is documented here and reflected in the Constitution Check table. If the Tresic observability standard is later extended to prescribe client-SDK metrics, revisit and add an OTel-metrics layer to the transport.

## R7 · Typed error hierarchy

**Decision**: Define a public error hierarchy in `errors.go`:

```text
APIError (interface)
├── *AuthenticationError       // HTTP 401
├── *AuthorizationError        // HTTP 403
├── *ValidationError           // HTTP 400, 422 — carries field-level details
├── *NotFoundError             // HTTP 404
├── *ConflictError             // HTTP 409
├── *RateLimitError            // HTTP 429 — carries Retry-After
├── *ServerError               // HTTP 5xx
└── *UnexpectedError           // everything else / schema-drift
```

All concrete types implement `APIError` (which embeds `error`) and satisfy `errors.Is` / `errors.As`. Each carries: `StatusCode int`, `Code string` (backend error code), `Message string`, `RequestID string`, `OperationID string`, `RawBody []byte` (bounded). Sentinel values (`ErrAuthentication`, `ErrAuthorization`, …) enable `errors.Is(err, intelligencecloud.ErrNotFound)` patterns.

**Rationale**:
- Spec FR-004 and SC-008 require classification by kind via the public error API; sentinel errors + type assertions give both patterns.
- `RequestID` is load-bearing for support triage (mentioned in FR-004).
- Bounded `RawBody` lets tools print a debug payload without risking unbounded memory on huge error responses.

**Alternatives considered**:
- Single `Error` struct with a `Kind` enum — simpler but doesn't compose with `errors.Is` in the ergonomic way Go consumers expect.
- Codes-only (`error` with a `Code()` method) — insufficient for compile-time exhaustiveness checks that Go switch-on-type gives.

## R8 · Pagination model

**Decision**: Generic iterator `Iterator[T]` with `Next(ctx) bool` / `Value() T` / `Err() error` / `PageInfo() PageInfo` API. Each resource's `List*` method returns an `*Iterator[T]`. Internally the iterator fetches pages on demand using the backend's page-token contract (determined from OpenAPI `parameters` — typically `page_token` / `page_size`; exact field names resolved at codegen time). Callers can cap results via `WithMaxItems(n)` option on the list call.

**Rationale**:
- Generic iterators are idiomatic post-Go 1.18 and map cleanly to Go 1.23's `range-over-func` if we later want to expose that convenience.
- Matches the pattern used in Google Cloud Go, Stripe Go, and DigitalOcean Godo.
- Satisfies FR-007 (no caller-side token reassembly) and the "no unbounded memory" constraint.

**Alternatives considered**:
- Return `([]T, nextToken, error)` tuples — leaks pagination to the caller; FR-007 explicitly forbids this.
- Channel-based streaming — lifecycle and cancellation semantics are harder; iterators compose better with `context.Context` deadlines.

## R9 · Canonical OpenAPI sourcing

**Decision**: Maintain a pinned copy of the canonical `openapi.yaml` in this repo at `testdata/openapi.yaml`, updated by a Makefile target `make pull-openapi` that runs `scripts/pull-openapi.sh`. The script fetches the spec from the `intelligence-cloud` repository at a pinned commit SHA (recorded in `testdata/openapi.commit`). Codegen reads from the local pinned copy. CI includes a "drift" job that, on pushes to `main`, fetches the latest `intelligence-cloud/main` openapi.yaml and opens a PR if it differs — never auto-merges.

**Rationale**:
- Builds are reproducible without network access — critical for release signing and CI reliability.
- SDK releases are explicitly pinned to a backend contract commit, which makes contract drift a review-gated event.
- Satisfies SC-007's "within one normal release cycle" target without coupling our build to the intelligence-cloud repo's availability.

**Alternatives considered**:
- Git submodule of `intelligence-cloud` — heavyweight (500 MB+ of unrelated code), slow clones, breaks private-repo access for external consumers.
- Release-artefact download (the intelligence-cloud repo publishes `openapi.yaml` as a release asset) — cleaner but requires IDB-1354 to also set up that release pipeline; out of scope for v0.1.
- Fetch at build time — breaks reproducibility, couples build to network.

## R10 · Experimental-API policy (Constitution IX adaptation)

**Decision**: Two-tier stability:
1. **Stable**: any exported symbol in `go doc` output without the `// Experimental:` prefix. Breaking changes only on major version bumps.
2. **Experimental**: symbols prefixed with `// Experimental:` in their GoDoc and placed behind the `experimental` build tag (`//go:build experimental`). Not compiled into default builds; consumers opt in with `go build -tags=experimental`. No semver compatibility commitment. Moved to Stable by removing the tag + comment when the design settles.

**Rationale**:
- The build-tag mechanism doubles as a compile-time feature flag, which is the closest analogue to the runtime feature-flag architecture the constitution mandates for backend services.
- Pre-1.0 releases (`v0.x`) additionally provide whole-module "experimental" semantics; post-1.0, only tagged symbols are experimental.
- Matches the Go ecosystem convention (`x/exp`, `slices.Experimental`, etc.) and is auditable via `git grep "Experimental:"`.

**Alternatives considered**:
- Runtime feature flags (e.g. LaunchDarkly client) inside the SDK — adds network dependency, inappropriate for a client library.
- Separate `intelligence-cloud-go/experimental` submodule — fragments the API surface and breaks `go get` of the main module.

## R11 · Compliance considerations (Constitution XI)

**Decision**: Record the following explicitly in `docs/architecture.md` and CI-check where automation is possible:

| Concern | Control | Verification |
|---|---|---|
| **Bearer-token at rest (CLI)** | OS keychain; `0600` file fallback | Unit test verifies file perms; integration test verifies keychain round-trip |
| **Bearer-token in transit** | HTTPS-only; `crypto/tls` default | Client constructor rejects `http://` URLs that are not `localhost` / `127.0.0.1` |
| **Secrets in logs** | `slog.Handler` wrapper redacts `Authorization` and any attribute whose key matches `/token|secret|key|password/i` | Unit test with captured log output asserts redaction |
| **Secrets in errors** | Typed errors carry `RawBody` capped at 4 KiB and strip request headers | Unit test |
| **Secrets in OTel spans** | Span attributes limited to the allow-list in R5; `url.full` has query params with names matching secret regex redacted | Unit test |
| **Audit trail (SOC2)** | Request ID (`X-Request-Id`) attached to every span, log line, and error | Integration test against `httptest.Server` that reflects the header |
| **HIPAA-adjacent data** | SDK does not persist any response body; callers are responsible for PHI handling downstream | Documented in `docs/architecture.md` |
| **Data residency** | SDK passes through whatever base URL the caller configures; no cross-region rewriting | N/A |

**Rationale**: These are the compliance commitments implied by the Tresic constitution + SOC2 auditor expectations. Capturing them here makes them reviewable now rather than discovered at audit time.

## R12 · Release toolchain

**Decision**: `goreleaser` driven by `.goreleaser.yaml`. Tag push (`vX.Y.Z`) on `main` triggers `.github/workflows/release.yaml`, which builds binaries for `{linux,darwin,windows} x {amd64,arm64}`, generates checksums, signs with `cosign` keyless (GitHub OIDC), and publishes a GitHub Release with `go.mod`-discoverable tag so `go get github.com/tresic-cloud/intelligence-cloud-go@vX.Y.Z` works immediately.

**Rationale**:
- Standard in the Go ecosystem (used by `goreleaser` itself, `terraform-provider-*`, `k6`, `docker/buildx`, etc.).
- `cosign` keyless signing satisfies supply-chain compliance (SLSA L2+) with no long-lived secrets.
- Semver tags on `main` are the `go get` discovery mechanism; nothing additional needed.

**Alternatives considered**:
- Hand-rolled `Makefile` + `go build -o` across GOOS/GOARCH matrix — works but re-implements what goreleaser already handles (checksums, archives, release notes, Homebrew tap).
- `gh release create` + GitHub Actions matrix — same objection.

## Summary of decisions

| ID | Decision | Affects |
|---|---|---|
| R1 | `oapi-codegen/v2` (types + low-level stubs) | FR-003, SC-001, SC-007 |
| R2 | `cobra` + `viper` | FR-010, FR-012, FR-013, FR-015 |
| R3 | `zalando/go-keyring` | FR-012a |
| R4 | RoundTripper-based retry, full jitter, `Retry-After` aware | FR-009a |
| R5 | OTel client spans via wrapped RoundTripper, semconv 1.24 | FR-008a |
| R6 | Client-SDK observability adaptation of Constitution IV | Constitution IV |
| R7 | Typed error hierarchy with sentinels + `errors.Is`/`errors.As` | FR-004, SC-008 |
| R8 | Generic `Iterator[T]` pagination | FR-007 |
| R9 | Pinned `testdata/openapi.yaml` with drift-check PR automation | SC-001, SC-007, FR-003 |
| R10 | `// Experimental:` + `experimental` build-tag for pre-stable APIs | Constitution IX |
| R11 | Explicit compliance controls recorded in architecture doc | Constitution XI |
| R12 | `goreleaser` + `cosign` keyless signing | FR-017, FR-018, SC-005 |

All previously-flagged `NEEDS CLARIFICATION` items are resolved. No deferred items block Phase 1.
