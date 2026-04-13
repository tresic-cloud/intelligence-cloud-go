# Feature Specification: Intelligence Cloud Go SDK & CLI Foundation

**Feature Branch**: `001-sdk-cli-foundation`
**Created**: 2026-04-13
**Status**: Draft
**Input**: User description: "We want to do this ticket IDB-1353, details on the spec itself are in IDB-1354, source in ../intelligence-cloud, in ./intelligence-cloud-go"
**Tracking**: [IDB-1353](https://tresic.atlassian.net/browse/IDB-1353) · Depends on [IDB-1354](https://tresic.atlassian.net/browse/IDB-1354)

## Clarifications

### Session 2026-04-13

- Q: Retry & rate-limit policy for transient failures → A: SDK auto-retries transient failures (5xx, 429, network errors) with exponential backoff + jitter; honours `Retry-After`; never retries non-429 4xx; retries tunable or disable-able via client options; context deadline always wins.
- Q: CLI credential storage at rest → A: OS keychain first (macOS Keychain, Windows Credential Manager, libsecret on Linux) with a `0600` file fallback under `$XDG_CONFIG_HOME/icctl/`; non-secret profile metadata always in a plaintext config file.
- Q: Observability contract → A: OpenTelemetry spans (method, path, status, duration, request ID, retry count) emitted when an OTel tracer provider is configured, otherwise silent; optional `slog.Logger` supplied via client options for structured debug/warn logs; no hard dependency on tracer/exporter — consumer wires them.
- Q: Token-expiry behaviour → A: Credential-provider interface returning a fresh token on demand (with small client-side cache); SDK surfaces a typed auth error if the provider errors; ship `StaticToken` and `RefreshFunc(ctx) (token, expiry, error)` adapters in v1; no built-in OAuth/MSAL flows in core.
- Q: Destructive-operation safety in the CLI → A: Destructive verbs (delete, deactivate, etc.) show a Pulumi-style preview of the operation — target environment, resource type, identifier, action — and then prompt for confirmation on a TTY; `--yes`/`-y` or `ICCTL_ASSUME_YES=1` bypasses the prompt for scripting; non-TTY invocations without an explicit bypass refuse to execute.

## Overview

The Intelligence Cloud platform currently exposes its HTTP API (authenticated through Azure API Management) without an officially supported Go client. Internal Go services, ops scripts, and integration partners each write their own request/response plumbing, duplicating work and drifting from the canonical contract. This feature delivers an idiomatic, typed Go SDK plus a companion command-line tool so that any Go program — and any operator at a terminal — can authenticate to Intelligence Cloud and invoke the full public API surface without writing bespoke HTTP code.

The SDK and CLI live in a new repository (`intelligence-cloud-go`) and are consumed via standard Go module tooling and versioned binary releases.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Go Developer Calls the API from Application Code (Priority: P1)

A backend engineer working on an internal Go service needs to fetch the signed-in user's profile, list companies under a reseller, and create new records against Intelligence Cloud. Today they hand-roll an HTTP client, copy auth boilerplate, and re-derive request/response types from the OpenAPI document. With this feature, they import a single Go module, supply credentials, and call typed methods that return typed results with typed errors.

**Why this priority**: This is the core reason the SDK exists. Until in-app calls work, nothing downstream (CLI, codegen pipelines, partner integrations) delivers value. A working SDK on its own is a shippable MVP — existing callers can migrate immediately.

**Independent Test**: A developer can, in a fresh Go project, `go get` the module, construct a client with a bearer token, call a read endpoint (e.g. "get me"), and receive a typed response with correct fields. A write endpoint (e.g. "create product") accepts a typed request and returns the created entity. Errors returned by the API surface as typed errors that expose status code, API error code, and message.

**Acceptance Scenarios**:

1. **Given** a valid API credential, **When** a consumer calls a read operation such as "get current user," **Then** the SDK returns a typed user record and no error.
2. **Given** an invalid or expired credential, **When** any operation is called, **Then** the SDK returns a typed authentication error distinguishable from other failures, with enough detail for the caller to prompt re-authentication.
3. **Given** an endpoint that returns a paginated list, **When** the consumer requests results, **Then** the SDK exposes a clear iteration mechanism that fetches subsequent pages without the caller reconstructing pagination tokens by hand.
4. **Given** the backend returns an error response, **When** the SDK surfaces it, **Then** the error includes HTTP status, backend error code (if present), human-readable message, and the originating request identifier for support triage.
5. **Given** a consumer configures a custom HTTP transport, base URL, timeout, or logger, **When** the client is constructed, **Then** those options take effect for all subsequent calls.

---

### User Story 2 — Operator Uses the CLI for Ad-Hoc and Scripted Workflows (Priority: P2)

A platform operator or support engineer needs to inspect and modify Intelligence Cloud resources without writing Go code — for example: listing resellers during an incident, creating a company on behalf of a customer, rotating a user's profile, or exporting audit logs. They install a single binary (`icctl` or equivalent), configure credentials once, and invoke subcommands that map 1:1 to SDK operations. Output is machine-readable (JSON) for piping into other tools and human-readable (table) for terminal use.

**Why this priority**: High leverage for ops and support, but not required for the SDK itself to deliver value to application developers. Operators can continue using direct API calls or the web UI in the interim.

**Independent Test**: An operator downloads the release binary, runs a configuration command to store credentials, executes a read subcommand (e.g. list resellers) and sees both a formatted table and a `--output json` variant. A write subcommand (e.g. create product) accepts flag inputs or a JSON payload and returns the created entity. Non-zero exit codes on failures are scriptable.

**Acceptance Scenarios**:

1. **Given** a fresh install, **When** the operator runs the help command, **Then** they see every available resource group and subcommand without needing external documentation.
2. **Given** valid credentials configured, **When** the operator runs a read subcommand with `--output json`, **Then** the output is valid JSON suitable for piping into `jq`.
3. **Given** valid credentials configured, **When** the operator runs a read subcommand without `--output json`, **Then** a readable table is printed to the terminal.
4. **Given** an operation fails, **When** the CLI exits, **Then** the exit code is non-zero, a concise error is written to standard error, and standard output remains parseable (or empty) so scripts can distinguish success from failure.
5. **Given** the operator invokes a write subcommand, **When** required arguments are missing or invalid, **Then** the CLI rejects the call with a clear usage message before any request is sent.

---

### User Story 3 — Release, Discoverability, and Onboarding (Priority: P3)

A new internal developer or integration partner discovers that an Intelligence Cloud Go SDK exists. They find a public repository with a clear README, a quickstart that works copy-paste, generated API reference documentation, and tagged semver releases they can pin. A release pipeline produces signed CLI binaries for Linux, macOS, and Windows without manual intervention.

**Why this priority**: Critical for adoption but not required for a working MVP. Early consumers can use an untagged commit if necessary.

**Independent Test**: Given the repository URL, an engineer who has never seen the SDK before can follow the README quickstart and successfully make their first authenticated API call within a short onboarding window. A tagged release is installable via standard Go module tooling; CLI binaries are downloadable from the release artifacts.

**Acceptance Scenarios**:

1. **Given** the repository, **When** a newcomer follows the quickstart, **Then** they complete a working "hello-world" API call without consulting a maintainer.
2. **Given** a tagged release, **When** a consumer pins to that version, **Then** the module builds reproducibly and the CLI binary for their platform is available as a release artifact.
3. **Given** the SDK code, **When** a consumer views the generated reference documentation, **Then** every public type and method has a description, parameter notes, and at least one usage example for each resource area.

---

### Edge Cases

- What happens when the caller's token has expired mid-operation — the SDK re-consults its credential provider (which may refresh the token) before retrying the request once; if the provider returns a fresh token the call proceeds, otherwise the SDK surfaces a typed authentication error.
- How does the SDK behave when the backend returns a response that doesn't conform to the declared contract (schema drift, new fields, missing required fields)?
- How are transient network failures, 5xx responses, and 429 responses handled — the SDK auto-retries with exponential backoff + jitter, honours `Retry-After` when present, and never retries non-429 4xx responses. Retry count, base delay, max delay, and on/off are exposed as client options; an enclosing context deadline always overrides retry behaviour.
- How are rate limits signalled to callers — a typed rate-limit error is surfaced only after retries are exhausted (or disabled), carrying the `Retry-After` value when provided.
- What happens when a CLI invocation targets an environment (e.g. staging) whose API surface differs from production — does the CLI gracefully reject unknown subcommands?
- How does pagination behave for very large result sets, and is there a clear way to cap or stream results without exhausting memory?
- What happens when two CLI users share a machine and each needs their own credential profile?
- How does the SDK handle clock skew between client and APIM for token validation?
- How does the CLI behave on terminals without color or without a TTY (piped output, CI logs)?

## Requirements *(mandatory)*

### Functional Requirements

**SDK core**

- **FR-001**: The SDK MUST expose a single, configurable client that consumers instantiate once per process, not per call.
- **FR-002**: The SDK MUST accept credentials via a credential-provider interface whose implementations return a bearer token (and an optional expiry hint) on demand. The client MUST consult the provider before each request, MUST apply a short-lived in-memory cache keyed by the provider's reported expiry, and MUST surface a typed authentication error if the provider returns an error. The SDK MUST ship at least two in-tree provider adapters in v1: a static-token provider and a refresh-function adapter that wraps any caller-supplied `func(ctx) (token, expiry, error)`. The SDK MUST NOT bundle a specific OAuth, MSAL, device-code, or managed-identity implementation in the core module; those are delivered by the caller or by separately versioned companion modules.
- **FR-003**: The SDK MUST cover every operation present in the canonical Intelligence Cloud OpenAPI document (see Assumptions) with a typed method, typed request input, and typed response output.
- **FR-004**: The SDK MUST expose typed errors that distinguish, at minimum, authentication failures, authorization failures, validation failures, not-found, rate-limit, and server-side errors, and MUST preserve the backend-provided error code, message, and request identifier when present.
- **FR-005**: The SDK MUST support a configurable base URL so the same binary can target production, staging, and local development environments.
- **FR-006**: The SDK MUST support context-based cancellation and deadlines on every operation.
- **FR-007**: The SDK MUST expose pagination for list endpoints in a way that does not require callers to manually assemble next-page tokens.
- **FR-008**: The SDK MUST allow consumers to inject a custom HTTP transport or request interceptor for logging, tracing, and test doubles.
- **FR-008a**: The SDK MUST emit an OpenTelemetry span for every outbound API call when an OTel tracer provider is configured on the client, and MUST emit nothing when no tracer is configured (zero runtime cost for non-observing consumers). Each span MUST carry, at minimum, the HTTP method, request path template (cardinality-safe — parameterised, not interpolated), response status, duration, backend request identifier (when returned), and retry attempt count. The SDK MUST accept a `slog.Logger` via client options for structured debug/warn logging and MUST NOT take a hard runtime dependency on any specific tracer, exporter, or log sink.
- **FR-009**: The SDK MUST remain backward compatible across patch releases and MUST follow semantic versioning for breaking changes.
- **FR-009a**: The SDK MUST automatically retry transient failures — server-side 5xx, 429 Too Many Requests, and transport-level errors — using exponential backoff with jitter, and MUST honour any `Retry-After` response hint. Non-429 4xx responses MUST NOT be retried. The retry count, base delay, maximum delay, and master on/off MUST be configurable via client options, and an enclosing context deadline MUST always override remaining retry attempts.

**CLI**

- **FR-010**: The CLI MUST provide a subcommand for every SDK resource area, with each operation reachable as a distinct verb.
- **FR-011**: The CLI MUST support at least two output formats: machine-readable structured output (JSON) and human-readable tabular output, selected by a flag, with JSON exit-clean for scripting.
- **FR-012**: The CLI MUST allow credential configuration via a persistent profile store, environment variables, and direct flags, with explicit precedence (flag > environment > profile).
- **FR-012a**: The CLI MUST store profile secrets (bearer tokens and any other credential material) in the operating system's native secret store when one is available — macOS Keychain, Windows Credential Manager, or the Secret Service API (libsecret) on Linux — and MUST fall back to a user-scoped file with `0600` permissions under `$XDG_CONFIG_HOME/icctl/` (or the platform equivalent) when no secret store is reachable. Non-secret profile metadata (profile name, environment selector, base URL, default output format) MUST be stored in a plaintext config file and MUST NOT contain credential material.
- **FR-013**: The CLI MUST support selection of named environments (production, staging, etc.) without requiring the user to memorize base URLs.
- **FR-014**: The CLI MUST return non-zero exit codes on any failure and write all error output to standard error, never standard output.
- **FR-015**: The CLI MUST print a useful help message for every subcommand that lists all flags, required inputs, and at least one example.
- **FR-015a**: Destructive CLI verbs (including but not limited to `delete`, `deactivate`, and any verb the backend documents as destructive) MUST display a Pulumi-style preview before issuing the API call. The preview MUST show, at minimum, the target environment name, the resource type, the resource identifier, and the action to be taken. The CLI MUST then prompt the operator for interactive confirmation on a TTY; `--yes`/`-y` MUST bypass the prompt, and the environment variable `ICCTL_ASSUME_YES=1` MUST provide a session-wide equivalent for scripted usage. Non-TTY invocations (piped input or CI) MUST refuse to execute a destructive verb unless the bypass flag or environment variable is explicitly present. The preview MUST also appear in `--output json` as a structured plan object so automation can inspect it before confirming.

**Packaging & operations**

- **FR-016**: The SDK MUST be installable via standard Go module tooling at a stable, public module path.
- **FR-017**: CLI binaries MUST be published as release artifacts for Linux, macOS, and Windows on common CPU architectures.
- **FR-018**: Every release MUST carry an immutable version tag and changelog entry describing what changed.
- **FR-019**: The project MUST ship a README with a runnable quickstart, and every public SDK symbol MUST carry reference documentation.

**Quality bar**

- **FR-020**: The project MUST maintain unit-test coverage at a level sufficient to exercise all happy paths, error paths, and pagination edges for every SDK method (target: ≥80% line coverage).
- **FR-021**: The project MUST include an integration smoke-test suite that runs against a non-production Intelligence Cloud environment and validates authentication plus a representative read and write against each resource area.
- **FR-022**: Every change MUST pass automated checks for linting, vulnerability scanning, and test execution before it can be released.

### Key Entities *(include if feature involves data)*

- **API Client**: The top-level SDK handle. Holds configuration (base URL, credentials, HTTP transport, timeout, logger) and exposes resource clients. Thread-safe and reusable across goroutines.
- **Resource Client**: A typed namespace under the API client for each resource area (users/me, resellers, companies, locations, products, connectors, audit logs, alerts, insights, etc.). Exposes the operations for that area.
- **Credentials**: The authentication material a consumer supplies. Represents a bearer token accepted by Azure API Management, plus optional supporting identifiers.
- **Request / Response Types**: Typed Go structs representing inputs and outputs for each operation, derived from the canonical OpenAPI contract.
- **Paginated Result**: A typed wrapper around list responses that exposes items plus a mechanism to advance through pages.
- **API Error**: A typed error carrying HTTP status, backend error code, human-readable message, request identifier, and the originating operation. Supports error classification via standard Go error semantics (`errors.Is` / `errors.As`).
- **CLI Command**: A named operator-facing invocation that maps to one SDK operation, with flags for required/optional inputs and output-format selection.
- **CLI Profile**: A named, on-disk credential and environment configuration that operators can switch between without retyping.
- **Output Formatter**: The component responsible for rendering an SDK response as either machine-readable JSON or a human-readable table.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The SDK covers 100% of operations listed in the canonical Intelligence Cloud OpenAPI document at the time of each release.
- **SC-002**: A first-time consumer can follow the quickstart and successfully execute their first authenticated API call in under 10 minutes.
- **SC-003**: At least 80% of lines in SDK packages are covered by automated unit tests, and every resource area has at least one passing integration smoke test against a non-production environment.
- **SC-004**: The CLI can execute every SDK operation and, for any operation, produces both a valid JSON output and a human-readable table output selectable by a single flag.
- **SC-005**: Tagged releases are installable via standard Go module tooling at a stable module path, and CLI binaries for Linux, macOS, and Windows are downloadable from the release page within minutes of the tag being pushed.
- **SC-006**: No production Go service inside Tresic hand-rolls its own Intelligence Cloud HTTP client six months after the SDK's first stable release; all such services have migrated to the SDK.
- **SC-007**: A change to the Intelligence Cloud API contract (as reflected in the canonical OpenAPI document) is reflected in an SDK release within one normal release cycle, without manual re-transcription of request/response types.
- **SC-008**: Every SDK error returned to a caller can be classified by kind (auth, authz, validation, not-found, rate-limit, server) using only the public error API.

## Assumptions

- **Single canonical API contract**: The canonical OpenAPI document at `backend/docs/api/openapi.yaml` in the `intelligence-cloud` repository is the authoritative source for the SDK's covered surface. IDB-1354 consolidates the API contracts; the SDK's initial release may ship against the current consolidated document, with coverage tracking as that document grows.
- **Authentication model**: Authentication is Azure AD bearer tokens presented to Azure API Management. The SDK exposes a credential-provider interface (see FR-002) rather than a raw token field; a `StaticToken` provider and a `RefreshFunc` adapter ship in v1. Acquiring the token (interactive login, service principal, managed identity, device-code flow) remains the caller's responsibility for v1 — the core SDK bundles no OAuth/MSAL implementation. Companion auth modules (e.g. `intelligence-cloud-go/auth/azure`) may be added later without a breaking change.
- **Code-derivation strategy**: Request/response types and low-level method stubs are derived from the canonical OpenAPI document via code generation, and a thin hand-written layer on top adds idiomatic options, pagination, typed errors, and context support. This keeps the SDK in lockstep with the contract and minimizes hand-maintained surface area.
- **Module path**: The Go module path is `github.com/tresic-cloud/intelligence-cloud-go` (the repository being specified).
- **CLI binary name**: The CLI is distributed as a single binary; the working name is `icctl`. The name is not load-bearing for the spec and may be refined before first release.
- **Transport security**: All communication occurs over HTTPS; plaintext HTTP is not supported for non-local environments.
- **Go version support**: The SDK targets a currently supported Go release (current stable and one prior minor version).
- **Out of scope for v1**: Generating the OpenAPI document itself (that is IDB-1354); providing language bindings other than Go; building a GUI front-end; persisting call history or results locally; acting as a long-running daemon.
- **Dependency on IDB-1354**: IDB-1354 ("Consolidate Intelligence Cloud OpenAPI contracts into single source of truth") is an explicit upstream. If the consolidated document is incomplete at codegen time, the SDK's coverage is bounded by what the document contains; the SDK itself MUST NOT transcribe operations from non-canonical sources.
- **Release cadence**: Releases follow semantic versioning. Pre-1.0 releases may break compatibility; post-1.0 releases do not break compatibility across patch or minor versions.
