---
title: Glossary
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - reference
  - glossary
aliases:
  - Terms
  - Definitions
related:
  - "[[05 Reference/_MOC|Reference MOC]]"
  - "[[Home]]"
---

# Glossary

One-line definitions of key terms used across this vault and the Intelligence Cloud Go codebase. Terms link to deeper notes where available.

| Term | Definition |
|---|---|
| **ADR** | Architectural Decision Record -- a structured note capturing a significant technical decision. See [[02 Decisions/_MOC]]. |
| **APIM** | Azure API Management, the Tresic platform's edge gateway that validates JWT bearer tokens and forwards them to the backend. |
| **Bearer token** | An OAuth 2.0 access token presented in the `Authorization` header. See [[Authentication]]. |
| **CredentialProvider** | SDK interface that returns a `Token` on demand. The client calls it before each request and caches the result until expiry or a 401 invalidation. See [[Authentication]]. |
| **Conventional Commits** | Commit-message specification that `release-please` uses to derive semantic version bumps automatically. See [[03 Guides/Contributing]]. |
| **Cosign** | Keyless artefact signing via GitHub OIDC. The release workflow uses Cosign to produce SLSA L2+ provenance without long-lived signing keys. See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]]. |
| **icctl** | The CLI binary that ships alongside the SDK. Provides subcommands that map 1:1 to SDK operations for terminal and scripting use. See [[01 Architecture/CLI Surface]]. |
| **Iterator[T]** | The SDK's generic paginator. Returns items one at a time while fetching pages on demand, bounded to one page in memory. See [[Pagination]]. |
| **MOC** | Map of Content -- an index note used by this vault to organise and cross-link related notes. |
| **OAS / OpenAPI** | Open API Specification; the SDK's typed methods and request/response types are generated from the canonical OpenAPI 3.1 document at `testdata/openapi.yaml`. |
| **oapi-codegen** | Go code-generation tool for OpenAPI. Produces typed request/response structs and low-level client stubs into `internal/generated/`. See [[02 Decisions/ADR-001 OpenAPI Codegen with oapi-codegen]]. |
| **OperationID** | The OpenAPI `operationId` field that names each HTTP operation; the SDK maps these to Go method names and OTel span names. See [[Operation IDs]]. |
| **OTel** | OpenTelemetry -- the vendor-neutral tracing and metrics API. The SDK emits client spans per HTTP semantic conventions when a `TracerProvider` is configured. See [[01 Architecture/Telemetry Contract]]. |
| **RequestID** | The `X-Request-Id` response header returned by the backend. Surfaced on every typed error and in OTel spans for support triage. See [[Error Hierarchy]]. |
| **RetryPolicy** | Tunable struct controlling the SDK's automatic retry layer -- max attempts, backoff timing, jitter strategy, and retryable status codes. See [[Retry Policy]]. |
| **RoundTripper** | Go's `http.RoundTripper` interface; the SDK's transport layer implements this to compose auth injection, retry, telemetry, and redaction into a single pipeline. See [[01 Architecture/HTTP Transport]]. |
| **Semver** | Semantic Versioning 2.0.0. The SDK follows it from `v1.0.0` onwards; pre-1.0 releases may include breaking changes between minor versions. |
| **SLSA** | Supply-chain Levels for Software Artifacts. The release workflow emits SLSA L2+ provenance via Cosign keyless signing. See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]]. |
| **Wikilink** | `[[Note Name]]` -- the internal cross-reference format used across this Obsidian vault. |
