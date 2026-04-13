---
title: Overview
type: architecture
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - architecture
  - sdk
  - cli
  - overview
aliases:
  - Architecture Overview
  - System Overview
related:
  - "[[SDK Surface]]"
  - "[[CLI Surface]]"
  - "[[HTTP Transport]]"
  - "[[Telemetry Contract]]"
---

# Architecture Overview

The `intelligence-cloud-go` repository delivers two artefacts from a single Go module:

1. **SDK** (`package intelligencecloud`) -- an importable Go library for calling the Intelligence Cloud HTTP API with typed requests, responses, and errors.
2. **CLI** (`icctl`) -- a command-line binary that exposes the full SDK surface to operators and scripts.

Both artefacts live at `github.com/tresic-cloud/intelligence-cloud-go`.

## Module Layout

```
intelligencecloud (root package)
  client.go          Client, NewClient, ClientOption
  options.go         functional options (WithHTTPClient, WithRetryPolicy, ...)
  errors.go          typed error hierarchy + sentinels
  retry.go           RetryPolicy, JitterStrategy, DefaultRetryPolicy
  pagination.go      Iterator[T], PageInfo, ListOption
  telemetry.go       OTel span helpers, slog integration

auth/                CredentialProvider interface + adapters
  provider.go        Token, CredentialProvider
  static.go          StaticToken
  refresh.go         RefreshFunc (expiry-aware caching)

internal/
  generated/         oapi-codegen output (types.gen.go, client.gen.go)
  transport/         RoundTripper composing auth, retry, OTel, redaction
  version/           build-info (semver, commit, Go version)

cmd/icctl/           CLI binary (cobra + viper)
  cmd/               one file per resource subcommand
  profile/           keychain + XDG config store
  output/            JSON + table formatters
  confirm/           destructive-verb preview + prompt
```

## Data Flow

A typical SDK call flows through these layers:

```
Caller code
  |
  v
ResourceService method (e.g. client.Resellers.List)
  |  builds typed request, returns Iterator[T] or (*T, error)
  v
internal/generated client function
  |  serialises to HTTP request
  v
internal/transport.Transport (http.RoundTripper)
  |  1. Inject User-Agent header
  |  2. Call CredentialProvider.Token(ctx) -> inject Authorization
  |  3. Propagate W3C trace context (traceparent, tracestate)
  |  4. Start OTel span
  |  5. Call base transport (net/http default or user-supplied)
  |  6. Capture X-Request-Id from response
  |  7. Retry decision (5xx, 429, transport errors)
  |  8. On final response: map HTTP status -> typed error
  |  9. End OTel span with attributes
  v
net/http -> TLS -> Intelligence Cloud API (Azure APIM)
```

## Key Design Principles

### Generated Core, Hand-Written Surface

Request/response types and low-level HTTP stubs are generated from the canonical OpenAPI document using `oapi-codegen`. The generated code lives behind `internal/` and is never imported directly by consumers. A thin hand-written layer on top provides the idiomatic Go API: functional options, typed errors, pagination iterators, and resource service namespaces. This keeps the SDK in lockstep with the backend contract while giving consumers a stable, ergonomic surface. See [[ADR-001 OpenAPI Codegen with oapi-codegen]].

### Composition via RoundTripper

Cross-cutting HTTP concerns (auth injection, retry, telemetry, secret redaction) are composed as a single `http.RoundTripper` wrapping the base transport. This is the canonical Go pattern for transport middleware and lets consumers supply their own base transport for testing or additional middleware. See [[HTTP Transport]] and [[ADR-006 RoundTripper-based Retry with Full Jitter]].

### Credential Provider Abstraction

The SDK does not know how to acquire tokens. It accepts a `CredentialProvider` interface with a single `Token(ctx) (Token, error)` method. Two built-in adapters ship: `StaticToken` (for testing and simple scripts) and `RefreshFunc` (wraps any callback with expiry-aware caching). This keeps OAuth, MSAL, and managed-identity concerns out of the core module. See [[Authentication]].

### Zero-Cost Observability

When no `TracerProvider` is configured, the SDK uses the OTel no-op tracer, which compiles span operations to near-zero overhead (benchmarked at under 100 ns/op additional). Consumers who want spans wire in their own provider. Similarly, the default logger is a no-op `slog.Handler`. See [[Telemetry Contract]] and [[ADR-007 OpenTelemetry Client Spans semconv 1.24]].

### Typed Error Classification

Every API error is returned as a concrete Go type that implements the `APIError` interface and wraps a sentinel for `errors.Is` matching. Eight concrete types cover the full HTTP status space. See [[Error Hierarchy]] and [[ADR-008 Typed Error Hierarchy with Sentinels]].

## Technology Choices

| Concern | Choice | ADR |
|---|---|---|
| Code generation | `oapi-codegen/v2` | [[ADR-001 OpenAPI Codegen with oapi-codegen]] |
| CLI framework | Cobra + Viper | [[ADR-002 Cobra and Viper for CLI]] |
| Credential storage | OS keychain via `go-keyring` | [[ADR-003 OS Keychain Credential Storage]] |
| Pre-1.0 stability | Experimental build tag | [[ADR-004 Pre-1.0 Experimental API Policy]] |
| OpenAPI sourcing | Pinned + drift PRs | [[ADR-005 Pinned OpenAPI with Drift PRs]] |
| Retry | RoundTripper, full jitter | [[ADR-006 RoundTripper-based Retry with Full Jitter]] |
| Telemetry | OTel client spans, semconv 1.24 | [[ADR-007 OpenTelemetry Client Spans semconv 1.24]] |
| Errors | Typed hierarchy + sentinels | [[ADR-008 Typed Error Hierarchy with Sentinels]] |
| Pagination | Generic Iterator[T] | [[ADR-009 Generic Iterator T for Pagination]] |
| Release | goreleaser + cosign keyless | [[ADR-010 Goreleaser plus Cosign Keyless Signing]] |
| Codegen workaround | Hand-curated generated code | [[ADR-011 Hand-curated Generated Code Stop-gap]] |

## Target Platforms

- **SDK**: any platform supported by Go 1.22+ (Linux, macOS, Windows, and others).
- **CLI binaries**: Linux amd64/arm64, macOS amd64/arm64, Windows amd64/arm64.
- **Go version**: targets Go 1.23 (current stable), minimum supported Go 1.22.
