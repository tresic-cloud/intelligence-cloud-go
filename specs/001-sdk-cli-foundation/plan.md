# Implementation Plan: Intelligence Cloud Go SDK & CLI Foundation

**Branch**: `001-sdk-cli-foundation` | **Date**: 2026-04-13 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-sdk-cli-foundation/spec.md`
**Tracking**: [IDB-1353](https://tresic.atlassian.net/browse/IDB-1353) · Depends on [IDB-1354](https://tresic.atlassian.net/browse/IDB-1354)

## Summary

Deliver an idiomatic Go SDK and companion CLI (`icctl`) for the Intelligence Cloud HTTP API. The SDK is generated from the consolidated OpenAPI contract at `backend/docs/api/openapi.yaml` in the `intelligence-cloud` repository (via `oapi-codegen`), wrapped in a hand-written ergonomic layer that adds credential-provider auth, exponential-backoff retry, pagination, OpenTelemetry span emission, and typed errors. The CLI is built on `cobra` + `viper`, stores bearer tokens in the OS-native keychain (with a `0600` file fallback), emits both JSON and human-readable table output, and guards destructive verbs behind a Pulumi-style preview + confirmation. Both artefacts ship together via `goreleaser` under `github.com/tresic-cloud/intelligence-cloud-go`, with cross-platform binaries and semver-tagged Go-module releases.

## Technical Context

**Language/Version**: Go 1.23 (current stable target); minimum supported Go 1.22 (one prior minor, per spec Assumptions)
**Primary Dependencies**:
- `github.com/oapi-codegen/oapi-codegen/v2` (build-time) — types + low-level method stubs from the canonical OpenAPI 3.1 document
- `github.com/oapi-codegen/runtime` (runtime) — decoding helpers for generated code
- `go.opentelemetry.io/otel` and `go.opentelemetry.io/otel/trace` (API-only; SDK core takes no hard dep on a tracer or exporter)
- `log/slog` (stdlib) — structured logging hook
- `github.com/spf13/cobra` + `github.com/spf13/viper` — CLI command framework and config
- `github.com/zalando/go-keyring` — cross-platform OS-keychain wrapper (Keychain / Credential Manager / libsecret)
- `github.com/olekukonko/tablewriter` — human-readable table output
- `github.com/stretchr/testify` — test assertions and mocks (test-only)
- `net/http/httptest` (stdlib) — transport-level integration tests
- `github.com/goreleaser/goreleaser` (CI-time) — cross-platform release orchestration

**Storage**:
- SDK: stateless — no disk or database access.
- CLI: non-secret profile metadata in plaintext YAML under `$XDG_CONFIG_HOME/icctl/config.yaml` (`0600`); secrets in OS keychain (`service=icctl`, `account=<profile>`); file fallback `$XDG_CONFIG_HOME/icctl/credentials.yaml` with `0600` perms when no keychain is reachable.

**Testing**:
- Unit: `go test` + `testify/assert` + `testify/require`. HTTP transport tested against `httptest.Server`.
- Contract: oapi-codegen-generated types pinned; diff check in CI that re-running codegen produces no changes.
- Integration (smoke): `go test -tags=integration` against the staging environment, gated behind a secret-injected credential in CI.
- Coverage: ≥80% line coverage on SDK packages (constitution II + spec FR-020).

**Target Platform**:
- SDK: any platform supported by Go 1.22+ (Linux, macOS, Windows, FreeBSD, WASM — best-effort).
- CLI binary: Linux amd64/arm64, macOS amd64/arm64, Windows amd64/arm64 — released via goreleaser.

**Project Type**: Go library (`github.com/tresic-cloud/intelligence-cloud-go`) + CLI (`cmd/icctl`) in a single Go module.

**Performance Goals**:
- SDK per-call overhead (excluding backend latency and retry sleeps): p99 ≤ 5 ms.
- OTel span emission: zero runtime cost when no tracer provider is configured (verified by benchmark `BenchmarkClient_NoTracer`).
- CLI cold-start for a help subcommand: ≤ 200 ms on a reference laptop (M-series and Linux x86_64).
- Retry decisions: O(1) per attempt; no goroutine leaks across retries or cancellations.

**Constraints**:
- HTTPS-only for any non-`localhost` base URL (rejected at client construction).
- No unbounded memory on pagination — iterators stream page-by-page, never load all pages into memory.
- Bearer tokens never appear in logs, error messages, or OTel attributes.
- Non-TTY destructive verb invocations refuse unless `--yes`/`ICCTL_ASSUME_YES=1` is explicit.
- SDK has zero mandatory runtime dependencies on a tracer, exporter, or log sink.

**Scale/Scope**:
- ~21 operations at v0.1 (current contents of `backend/docs/api/openapi.yaml`), growing as IDB-1354 consolidates more per-feature contracts.
- Hand-written code surface target: ≤ 3 000 LOC (ergonomic layer + CLI + tests). Generated code is unbounded but not counted against the hand-maintained budget.
- Anticipated resource areas at v0.1: `me`, `resellers`, `companies`, `locations`, `products`, `verticals`, `connectors`, `audit-logs`, `users`, `auth`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Applies? | Status | How Satisfied / Justification |
|---|-----------|----------|--------|-------------------------------|
| I | Library-First with CLI Interface | **Yes** | PASS | Feature is literally a reusable library + CLI. Library (`intelligencecloud` package) is independently testable and documented; CLI (`icctl`) exposes the full SDK surface with text in/out. |
| II | Test-First Development (TDD, ≥80%) | **Yes** | PASS | Every task (see `/speckit.tasks` output) will produce `*_test.go` files before implementation. Coverage gate enforced in CI via `go test -coverprofile` with ≥80% threshold. Constitution reference to `*.test.ts` is adapted to Go convention `*_test.go`. |
| III | Integration Testing | **Yes** | PASS | Contract tests: generated-code-drift check. Service-to-service tests: `httptest.Server`-backed transport tests. External API: smoke-test suite against staging (FR-021). |
| IV | Observability | **Yes — adapted** | PASS | Constitution is framed around backend HTTP handlers (W3C `traceparent` inbound, Prometheus `InstrumentedService` decorators). For an outbound client SDK the correct adaptation is: **(a)** propagate W3C trace context on outbound requests; **(b)** emit OTel client spans (FR-008a); **(c)** structured `slog` logger. Prometheus decorator pattern does not apply — no services/repositories to decorate. Adaptation documented in `research.md`. |
| V | Documentation First | **Yes** | PASS | `README.md`, `docs/quickstart.md`, GoDoc on every exported symbol, this plan + spec + research.md + data-model.md + contracts + quickstart all authored before implementation begins. |
| VI | Quality Standards | **Yes** | PASS | No `TODO`/`FIXME`/`Phase 2` permitted. All error paths typed (FR-004). Input validation on every SDK builder and CLI flag. `go vet`, `staticcheck`, and `govulncheck` enforced in CI (FR-022). |
| VII | APIs as First-Class Features | **Yes** | PASS | Public Go API is a product surface — every exported symbol carries GoDoc, every resource has at least one runnable example (FR-019, SC-005). Breaking changes versioned via semver (FR-009). |
| VIII | Scope-Based Authorization | **Partial — N/A for SDK enforcement** | PASS with justification | The SDK does not define or enforce authorization scopes; that is the backend's responsibility and is enforced at Azure APIM and the backend. The SDK MUST, however, preserve and surface `403 Forbidden` responses as a typed `AuthorizationError` (FR-004) and MUST never drop scope-related error context from backend responses. Documented in research.md. |
| IX | Feature Flag Architecture | **Adapted** | PASS | SDK/CLI releases follow semver; pre-1.0 (`v0.x`) is the feature-flag equivalent — unstable surfaces may change. Experimental APIs are marked with `// Experimental:` GoDoc prefix and an `experimental` build tag, and are excluded from the semver compat contract. Documented in research.md. |
| X | Backend/Frontend Isolation | **N/A** | PASS | No backend/frontend split here — this repo IS the client; the backend is an external dependency. |
| XI | Built for Compliance | **Yes** | PASS | Secret handling: OS keychain + `0600` fallback; no plaintext secrets in config or logs. Audit trail: structured logs and OTel spans carry request IDs. HTTPS-enforced. No PII persisted by SDK/CLI. Compliance rationale recorded in research.md. |

**Pre-Phase-0 gate**: PASS. Two adaptations (IV observability pattern and IX pre-1.0 stability) are documented with rationale and do not constitute violations — no entry in Complexity Tracking required.

**Post-Phase-1 re-check (2026-04-13)**: PASS. The Phase 1 artefacts (`data-model.md`, `contracts/sdk-api.md`, `contracts/cli-schema.md`, `contracts/telemetry.md`, `quickstart.md`) introduce no new dependencies, patterns, or complexity beyond what was already justified. Specifically:
- II (TDD) — contracts enumerate testable surfaces (public Go API, CLI command tree, telemetry attributes) each with verification criteria the task layer will turn into failing tests first.
- IV (Observability) — `contracts/telemetry.md` locks the adapted observability contract into the reviewable spec surface.
- VII (APIs as first-class) — `contracts/sdk-api.md` is the semver-binding public-surface contract.
- XI (Compliance) — `contracts/telemetry.md` "Prohibited content" + `data-model.md` "Validation rules" together make every R11 control testable.

No Complexity Tracking entries added.

## Project Structure

### Documentation (this feature)

```text
specs/001-sdk-cli-foundation/
├── plan.md              # This file (/speckit.plan output)
├── spec.md              # Feature spec (/speckit.specify + /speckit.clarify)
├── research.md          # Phase 0 output — decisions + rationale
├── data-model.md        # Phase 1 output — entities and types
├── quickstart.md        # Phase 1 output — copy-paste first-call walkthrough
├── contracts/
│   ├── sdk-api.md       # Public Go API surface contract
│   ├── cli-schema.md    # CLI command / flag / output contract
│   └── telemetry.md     # OTel span attribute contract
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT produced here)
```

### Source Code (repository root)

```text
intelligence-cloud-go/                # Go module: github.com/tresic-cloud/intelligence-cloud-go
├── client.go                         # package intelligencecloud — top-level Client
├── options.go                        # ClientOption functional options
├── errors.go                         # typed error hierarchy
├── retry.go                          # retry policy (FR-009a)
├── pagination.go                     # Iterator[T] for list endpoints (FR-007)
├── telemetry.go                      # OTel span emission (FR-008a)
├── resellers.go                      # hand-written ergonomic wrappers
├── companies.go
├── locations.go
├── products.go
├── verticals.go
├── connectors.go
├── auditlogs.go
├── users.go
├── me.go
├── auth/
│   ├── provider.go                   # CredentialProvider interface
│   ├── static.go                     # StaticToken provider
│   └── refresh.go                    # RefreshFunc adapter
├── internal/
│   ├── generated/                    # oapi-codegen output — do not edit by hand
│   │   ├── types.gen.go
│   │   └── client.gen.go
│   ├── transport/                    # HTTP round-tripper (auth inject, retry, OTel)
│   │   └── transport.go
│   └── testutil/                     # shared test helpers
├── cmd/
│   └── icctl/
│       ├── main.go
│       ├── cmd/                      # cobra commands — one file per resource
│       │   ├── root.go
│       │   ├── resellers.go
│       │   ├── companies.go
│       │   └── ...
│       ├── profile/                  # keychain + XDG config store (FR-012a)
│       │   ├── store.go
│       │   ├── keychain.go
│       │   └── file.go
│       ├── output/                   # JSON + table formatters (FR-011)
│       │   ├── formatter.go
│       │   ├── json.go
│       │   └── table.go
│       └── confirm/                  # destructive-verb preview + prompt (FR-015a)
│           └── preview.go
├── examples/                         # runnable code samples — one per resource area
│   ├── me/
│   ├── resellers/
│   └── ...
├── docs/
│   ├── quickstart.md
│   ├── architecture.md
│   └── adr/                          # Architectural Decision Records
├── scripts/
│   ├── codegen.sh                    # re-runs oapi-codegen from vendored OpenAPI
│   └── pull-openapi.sh               # fetches canonical openapi.yaml from intelligence-cloud
├── testdata/
│   └── openapi.yaml                  # pinned copy of canonical spec for reproducible builds
├── .goreleaser.yaml
├── .golangci.yaml
├── .github/workflows/
│   ├── ci.yaml                       # test + lint + vuln scan + codegen-drift
│   └── release.yaml                  # goreleaser on tag push
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE
└── CHANGELOG.md
```

**Structure Decision**: Single Go module rooted at the repository, with the SDK as the root package (`intelligencecloud`) and the CLI as a `cmd/icctl` sub-binary. Generated code is confined to `internal/generated/` and never edited by hand. Resource files at the module root are thin hand-written wrappers over the generated layer, giving consumers a stable, idiomatic API regardless of codegen-tool or OpenAPI churn. This layout matches how mature Go SDKs (Stripe, Twilio, DigitalOcean) are structured and makes it trivial to release the CLI as an independent binary while keeping the SDK importable as a single `go get` dependency.

## Complexity Tracking

*No unjustified complexity. Two constitution adaptations (IV observability, IX pre-1.0 stability) are covered inline in the Constitution Check table and documented in `research.md`.*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| *(none)*  | —          | —                                    |
