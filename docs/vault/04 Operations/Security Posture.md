---
title: Security Posture
type: operational
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - ops
  - security
  - secrets
  - supply-chain
aliases:
  - Security Overview
  - Threat Model
related:
  - "[[Compliance Controls]]"
  - "[[Observability Runbook]]"
  - "[[01 Architecture/HTTP Transport]]"
  - "[[01 Architecture/Telemetry Contract]]"
  - "[[02 Decisions/ADR-003 OS Keychain Credential Storage]]"
  - "[[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]]"
---

# Security Posture

This document describes the security properties of the Intelligence Cloud Go SDK and CLI from an operator's perspective: what secrets exist, how they are protected, and what assurances the toolchain provides at build and runtime.

## Secrets Lifecycle

### CredentialProvider Model

The SDK does not manage token acquisition directly. It accepts a `CredentialProvider` interface with a single method:

```go
type CredentialProvider interface {
    Token(ctx context.Context) (Token, error)
}
```

Two built-in adapters ship:

- **StaticToken** -- wraps a fixed bearer token string. Suitable for testing and short-lived scripts.
- **RefreshFunc** -- wraps any callback that returns a token with an expiry time. The adapter caches the token and re-invokes the callback when the cached token is within a **30-second safety margin** of expiry. This margin prevents clock-skew-related 401s without requiring tight NTP synchronisation.

The SDK itself never persists tokens. The cached token lives only in process memory and is invalidated on 401 responses (with exactly one re-fetch attempt before surfacing an `*AuthenticationError`).

See [[05 Reference/Authentication]] for the full API contract.

### CLI Credential Storage

The CLI (`icctl`) stores bearer tokens for named profiles using the OS keychain:

- **macOS**: Keychain Services (`security` framework)
- **Windows**: Windows Credential Manager
- **Linux**: Secret Service API via D-Bus (libsecret)

When no keychain daemon is available (headless servers, containers), the CLI falls back to a user-scoped file at `$XDG_CONFIG_HOME/icctl/credentials.yaml` with `0600` permissions.

Non-secret profile metadata (profile name, environment selector, base URL, default output format) is stored in a separate plaintext configuration file and never contains credential material.

See [[02 Decisions/ADR-003 OS Keychain Credential Storage]] for the full decision record.

## Redaction Rules

The SDK applies a consistent redaction strategy across three telemetry surfaces to prevent secrets from leaking to external APM backends, log aggregators, or error-tracking systems.

### Attribute key regex

Any slog attribute whose key matches the following regex is stripped before reaching the underlying log handler:

```
/token|secret|key|password/i
```

This regex is intentionally broad to catch novel secret-bearing attributes without requiring an explicit blocklist update.

### URL query-parameter redaction

When recording the `url.full` span attribute, the transport replaces the value of any query parameter whose name matches the same regex with `[REDACTED]`:

```
https://api.example.com/v1/resource?api_key=[REDACTED]&page=2
```

### RedactedHeaders default set

The following headers are excluded from all telemetry (spans, logs, and error payloads):

- `Authorization`
- `Cookie`
- `X-Api-Key`
- `X-Token`

Consumers can extend this set but cannot remove the defaults.

See [[01 Architecture/Telemetry Contract#Prohibited content]] and [[01 Architecture/HTTP Transport]] for implementation details.

## Transport Security

The SDK enforces HTTPS for all outbound API traffic. The client constructor rejects base URLs with an `http://` scheme unless the host is `localhost` or `127.0.0.1`. This prevents accidental cleartext transmission of bearer tokens.

TLS configuration uses Go's `crypto/tls` defaults, which negotiate TLS 1.2+ with a modern cipher suite. Consumers who require specific TLS settings (e.g. certificate pinning, mutual TLS) can supply a custom `http.Transport` via the `WithHTTPClient` option.

The transport injects W3C trace-context headers (`traceparent`, `tracestate`, `baggage`) on every outbound request for distributed tracing continuity. These headers contain no secret material.

## Supply Chain

### Binary signing

Release binaries are built by `goreleaser` in a GitHub Actions workflow triggered by semver tag pushes to `main`. Each binary and its checksum file are signed with **cosign keyless** signing, which uses GitHub's OIDC identity provider as the trust anchor. No long-lived signing keys exist.

Consumers verify release binaries with:

```bash
cosign verify-blob \
  --certificate-identity-regexp "github.com/tresic-cloud/intelligence-cloud-go" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --bundle intelligence-cloud-go_<version>_<os>_<arch>.sig.bundle \
  intelligence-cloud-go_<version>_<os>_<arch>.tar.gz
```

This satisfies SLSA Level 2+ provenance requirements.

See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]] for the full decision record.

### Go module integrity

The SDK is published as a Go module. Consumers fetch it via `go get`, which verifies the module against the Go checksum database (`sum.golang.org`). This provides tamper detection for source-level dependencies.

## Disclosure Policy

Security vulnerabilities should be reported via **GitHub Security Advisories** on the `intelligence-cloud-go` repository. The repository's `SECURITY.md` file is the canonical source for the current disclosure process, supported versions, and expected response times.

> **Note**: `SECURITY.md` will be added to the repository as part of the initial release preparation. Until then, report vulnerabilities directly to the repository maintainers via GitHub's private vulnerability reporting feature.

## See Also

- [[Compliance Controls]] -- the full controls matrix with verification strategies
- [[Observability Runbook]] -- interpreting the telemetry that redaction produces
- [[01 Architecture/Overview]] -- system architecture context
- [[01 Architecture/HTTP Transport]] -- transport pipeline details
- [[01 Architecture/Telemetry Contract]] -- complete span attribute contract
