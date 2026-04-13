---
title: Authentication
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - reference
  - sdk
  - auth
  - credentials
aliases:
  - Auth
  - CredentialProvider
  - Token Lifecycle
related:
  - "[[02 Decisions/ADR-003 OS Keychain Credential Storage]]"
  - "[[04 Operations/Security Posture]]"
  - "[[Error Hierarchy]]"
  - "[[01 Architecture/HTTP Transport]]"
  - "[[SDK Surface]]"
---

# Authentication

The SDK authenticates to the Intelligence Cloud platform by presenting Azure AD bearer tokens to APIM (Azure API Management). The SDK itself does **not** acquire tokens -- it delegates token acquisition to a caller-supplied `CredentialProvider` and focuses on caching, injection, and error handling.

## Model

```
Caller code  --->  CredentialProvider  --->  Token
                       |
                       v
                   SDK Client  --[Authorization: Bearer <token>]-->  APIM  --->  Backend
```

The SDK takes a `CredentialProvider` at construction and does not know or care how the token was acquired -- interactive login, service principal, managed identity, device code flow, or any other mechanism. This keeps the core SDK free of OAuth/MSAL dependencies.

## CredentialProvider Interface

```go
package auth

type CredentialProvider interface {
    Token(ctx context.Context) (Token, error)
}
```

The `Token` method is called before each API request (subject to caching). It must return a valid bearer token or an error. The context allows the provider to honour deadlines and cancellation.

## Token Struct

```go
type Token struct {
    AccessToken string    // The bearer token value (never logged, never in spans)
    ExpiresAt   time.Time // When the token expires; zero value means "never expires"
}
```

`AccessToken` is treated as a secret throughout the SDK -- it never appears in logs, error messages, or OTel span attributes. See [[04 Operations/Security Posture]] for the full redaction posture.

## Built-in Adapters

### auth.StaticToken

The simplest provider -- always returns the same token with no expiry:

```go
provider := auth.StaticToken("eyJhbGciOi...")
client, _ := ic.NewClient("https://api.intelligence.cloud", provider)
```

Useful for testing, short-lived scripts, and environments where token rotation is handled externally.

### auth.RefreshFunc

Wraps a caller-supplied refresh callback with single-flight caching and a 30-second safety margin before `ExpiresAt`:

```go
provider := auth.RefreshFunc(func(ctx context.Context) (auth.Token, error) {
    tok := getFreshTokenFromMSAL(ctx) // caller's problem
    return auth.Token{
        AccessToken: tok.AccessToken,
        ExpiresAt:   tok.ExpiresAt,
    }, nil
})
client, _ := ic.NewClient("https://api.intelligence.cloud", provider)
```

The `RefreshFunc` adapter:

- Caches the returned token until `ExpiresAt - 30s` (the safety margin prevents using tokens that are about to expire).
- Re-invokes the callback when the cache expires or when a 401 response invalidates the cached token.
- Is safe for concurrent use -- multiple goroutines calling the client simultaneously will not trigger redundant refresh calls.

## Token Lifecycle

The full lifecycle of a token within a single client:

```
[no-cache]
    |  first API call
    v
[provider.Token(ctx) invoked] --error--> *AuthenticationError (terminal for this call)
    | ok
    v
[cache {token, expiresAt - 30s margin}]
    |
    v
[request sent with Authorization: Bearer <token>]
    |
    |-- 401 response --> invalidate cache --> provider.Token(ctx) again
    |                        |-- ok + different token --> retry once --> 401 --> *AuthenticationError
    |                        |-- ok + same token -----> *AuthenticationError (loop guard)
    |                        |-- error ----------------> *AuthenticationError
    |
    |-- success (2xx) --> cache retained until expiry
    |
    |-- expiry reached before next call --> cache invalidated; next call re-invokes provider
```

Key invariants:

- **At most one re-fetch per call**: a 401 triggers exactly one provider re-invocation. If the second attempt also fails, the SDK surfaces a `*AuthenticationError` immediately.
- **Same-token detection**: if the provider returns the same `AccessToken` after a 401 invalidation, the SDK surfaces an error instead of retrying with a token that already failed.
- **Provider errors are fatal**: if `Token(ctx)` returns an error, the SDK wraps it in a `*AuthenticationError` with the cause chain preserved.

## v1 Scope

The core SDK module ships **no** built-in OAuth, MSAL, device-code, or managed-identity flows. Callers wire those themselves via the `CredentialProvider` interface. This is a deliberate design choice:

- Keeps the core module dependency-free of Azure SDK and MSAL libraries.
- Allows callers to use any identity provider, not just Azure AD.
- Companion auth modules (e.g. `intelligence-cloud-go/auth/azure`) may arrive later as separately versioned submodules without a breaking change to the core API.

For CLI-side credential storage (OS keychain, file fallback), see [[02 Decisions/ADR-003 OS Keychain Credential Storage]].

## Security Posture

Tokens are treated as secrets throughout the SDK stack:

- **Never logged**: the `Authorization` header value never appears in `slog` output. The redacting log handler strips any attribute whose key matches `/token|secret|key|password/i`.
- **Never in errors**: typed errors carry HTTP status, error code, and request ID -- never the bearer token.
- **Never in OTel spans**: the `Authorization` header is explicitly excluded from span attributes. URL query parameters matching secret patterns are replaced with `[REDACTED]`.
- **Never persisted**: the SDK does not write tokens to disk. Persistence is the CLI's responsibility (via OS keychain; see [[02 Decisions/ADR-003 OS Keychain Credential Storage]]).

See [[04 Operations/Security Posture]] for the complete threat model and redaction controls.

## See Also

- [[02 Decisions/ADR-003 OS Keychain Credential Storage]] -- CLI-side credential persistence
- [[04 Operations/Security Posture]] -- threat model and redaction guarantees
- [[Error Hierarchy]] -- `*AuthenticationError` details
- [[01 Architecture/HTTP Transport]] -- where token injection and 401 re-fetch happen in the pipeline
- [[Retry Policy]] -- how the 401 re-fetch path interacts with the general retry loop
- [[SDK Surface]] -- `NewClient` constructor and `WithHTTPClient` option
