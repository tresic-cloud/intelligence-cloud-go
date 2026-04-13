---
title: Compliance Controls
type: operational
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - ops
  - compliance
  - soc2
  - security
aliases:
  - Controls Matrix
  - Compliance Matrix
related:
  - "[[Security Posture]]"
  - "[[Observability Runbook]]"
  - "[[01 Architecture/HTTP Transport]]"
  - "[[01 Architecture/Telemetry Contract]]"
  - "[[02 Decisions/ADR-003 OS Keychain Credential Storage]]"
  - "[[02 Decisions/ADR-007 OpenTelemetry Client Spans semconv 1.24]]"
---

# Compliance Controls

This document records the compliance controls built into the Intelligence Cloud Go SDK and CLI. Each row maps a security concern to the control that addresses it, where the control is implemented, and how it is verified. The matrix derives from the project's compliance review and is aligned with the Tresic constitution's principle XI (Built for Compliance).

## Controls Matrix

| Concern | Control | Implementation | Verification |
|---|---|---|---|
| Bearer-token at rest (CLI) | OS keychain + `0600` file fallback | `cmd/icctl/profile/` -- see [[02 Decisions/ADR-003 OS Keychain Credential Storage]] and [[05 Reference/Authentication]] | unit + integration |
| Bearer-token in transit | HTTPS-only; TLS defaults | Client constructor rejects non-HTTPS unless localhost -- see [[01 Architecture/HTTP Transport]] | unit test |
| Secrets in logs | slog handler redaction regex | `telemetry.go` `RedactingHandler` -- see [[01 Architecture/Telemetry Contract]] | capture test |
| Secrets in errors | Error types carry capped `CauseBody` | `errors.go` -- see [[05 Reference/Error Hierarchy]] | unit test |
| Secrets in OTel spans | Attribute allow-list; `url.full` query-param redaction | `telemetry.go` + `internal/transport` -- see [[01 Architecture/Telemetry Contract]] and [[01 Architecture/HTTP Transport]] | span-capture test |
| Audit trail (SOC2) | `X-Request-Id` propagated to span attr + error field + log attr | `internal/transport` -- see [[01 Architecture/HTTP Transport]] | integration test |
| HIPAA-adjacent data | SDK does not persist response bodies | N/A -- callers are responsible for PHI handling downstream | documented |
| Data residency | SDK uses caller's base URL unchanged | N/A -- no cross-region rewriting | N/A |

## Standards Considered

The controls above are designed with the following compliance frameworks in mind:

- **SOC 2** -- the audit trail control (X-Request-Id correlation across spans, logs, and errors) directly supports the Common Criteria for monitoring and accountability. Secret redaction ensures that telemetry exports sent to third-party APM backends do not leak credentials.
- **ISO 27001** -- information security management controls around access (keychain storage), encryption in transit (HTTPS-only), and logging (structured, redacted logs) map to Annex A controls A.10 (cryptography) and A.12 (operations security).
- **HiTrust** -- the CSF control categories for access control and audit logging are addressed by the same keychain + redaction + request-ID mechanisms.
- **HIPAA** -- the SDK itself does not persist protected health information (PHI). Response bodies are returned to the caller but never written to disk or telemetry. This is documented as a shared-responsibility boundary: the SDK provides the transport layer; callers are responsible for PHI handling in their application layer.

## Threat Model Summary

Three categories of secrets flow through the SDK:

1. **Bearer tokens** -- acquired from the `CredentialProvider` and injected into the `Authorization` header. These grant API access and must be protected at rest (CLI credential store) and in transit (TLS).
2. **API keys and cookies** -- may appear in custom headers supplied by the caller. The SDK redacts any header in the `RedactedHeaders` set from all telemetry.
3. **Sensitive query parameters** -- some API endpoints accept tokens or keys as URL query parameters. The SDK's regex-based redaction (`/token|secret|key|password/i`) scrubs these from the `url.full` span attribute and from log output.

Where secrets MUST NOT flow:

- **Log output** -- the `RedactingHandler` wraps the consumer-supplied `slog.Logger` and strips matching attributes before they reach the underlying handler.
- **OTel span attributes and events** -- the transport only records attributes from the allow-list; the `Authorization` header value is never set as an attribute.
- **Error payloads** -- typed errors carry a bounded `CauseBody` (capped at 4 KiB) that excludes request headers.

## Compliance Rationale

### OS keychain for credential storage

Plaintext credential files are the most common source of credential leaks in CLI tools. Using the OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service) delegates key protection to hardware-backed or OS-level access controls. The `0600` file fallback exists only for environments where no keychain daemon is available (headless servers, containers) and matches the security posture of comparable tools such as `gh` and `gcloud`. See [[02 Decisions/ADR-003 OS Keychain Credential Storage]].

### HTTPS-only transport

The client constructor rejects `http://` base URLs unless the host is `localhost` or `127.0.0.1`. This prevents accidental cleartext transmission of bearer tokens over the network. The exception for localhost enables local development and testing without requiring self-signed certificates.

### Regex-based redaction

A single regex `/token|secret|key|password/i` is applied consistently across three surfaces: slog attributes, URL query parameters in span attributes, and the `RedactedHeaders` default set. This defence-in-depth approach ensures that even if a new secret-bearing header or parameter is introduced, it is likely caught by the regex before it reaches an external telemetry backend. See [[01 Architecture/Telemetry Contract#Prohibited content]].

### X-Request-Id as audit trail

Every API response from the Intelligence Cloud backend includes an `X-Request-Id` header. The SDK captures this value and attaches it to:

- The OTel span as `intelligencecloud.request_id`
- Every slog record emitted during that request
- The `RequestID` field on typed error objects

This three-way correlation allows operators and auditors to trace any user-visible error back to a specific backend request, satisfying SOC 2 monitoring requirements. See [[01 Architecture/HTTP Transport]] and [[02 Decisions/ADR-007 OpenTelemetry Client Spans semconv 1.24]].

### No response-body persistence

The SDK returns response bodies to the caller but never writes them to disk, cache, or telemetry. This ensures that sensitive data (including potential PHI in HIPAA-regulated deployments) is the caller's responsibility to handle according to their own compliance requirements. The `http.response.body.size` attribute is recorded for diagnostics, but the body content itself is excluded from all telemetry.

## See Also

- [[Security Posture]] -- broader security overview including supply-chain signing
- [[Observability Runbook]] -- how to interpret the telemetry that these controls produce
- [[01 Architecture/Overview]] -- system architecture context
