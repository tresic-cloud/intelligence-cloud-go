---
title: Error Hierarchy
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - reference
  - sdk
  - errors
aliases:
  - Errors
  - Typed Errors
  - SDK Errors
related:
  - "[[01 Architecture/HTTP Transport]]"
  - "[[02 Decisions/ADR-008 Typed Error Hierarchy with Sentinels]]"
  - "[[SDK Surface]]"
  - "[[04 Operations/Compliance Controls]]"
  - "[[Retry Policy]]"
---

# Error Hierarchy

The SDK defines a structured error hierarchy that lets consumers classify errors with `errors.Is` (quick check) and inspect details with `errors.As` (full access). Every non-nil error returned from an API call implements the [[#APIError Interface]]. Construction errors from `NewClient` are a separate type that does **not** implement `APIError`.

## Sentinel Errors

Seven package-level sentinel errors support `errors.Is` matching:

| Sentinel | Message |
|---|---|
| `ErrAuthentication` | `intelligencecloud: authentication failed` |
| `ErrAuthorization` | `intelligencecloud: not authorized` |
| `ErrValidation` | `intelligencecloud: validation failed` |
| `ErrNotFound` | `intelligencecloud: resource not found` |
| `ErrConflict` | `intelligencecloud: resource conflict` |
| `ErrRateLimit` | `intelligencecloud: rate-limited` |
| `ErrServer` | `intelligencecloud: server error` |

Each concrete error type implements `Unwrap()` returning its sentinel, so `errors.Is(err, ic.ErrNotFound)` works out of the box without type assertions.

## APIError Interface

Every API-call error satisfies this interface:

```go
type APIError interface {
    error
    Status() int       // HTTP status code (e.g. 404)
    Code() string      // Backend error code (e.g. "RESOURCE_NOT_FOUND")
    RequestID() string // X-Request-Id header value for support triage
    Operation() string // OpenAPI operationId that produced this error
}
```

The `RequestID` is critical for correlating SDK errors with backend logs during incident response. It is also emitted on the corresponding OTel span as `intelligencecloud.request_id`.

## Concrete Types

Eight concrete error types provide detailed, type-safe access to error metadata:

| Concrete type | Sentinel | HTTP status | Extra fields |
|---|---|---|---|
| `*AuthenticationError` | `ErrAuthentication` | 401 | -- |
| `*AuthorizationError` | `ErrAuthorization` | 403 | `RequiredScopes []string` |
| `*ValidationError` | `ErrValidation` | 400, 422 | `FieldErrors []FieldError` |
| `*NotFoundError` | `ErrNotFound` | 404 | `ResourceType string`, `ResourceID string` |
| `*ConflictError` | `ErrConflict` | 409 | -- |
| `*RateLimitError` | `ErrRateLimit` | 429 | `RetryAfter time.Duration` |
| `*ServerError` | `ErrServer` | 5xx | -- |
| `*UnexpectedError` | *(none)* | anything else | `CauseBody []byte` (capped at 4 KiB) |

The `FieldError` struct carried by `*ValidationError` breaks down per-field problems:

```go
type FieldError struct {
    Field   string // JSON path of the invalid field
    Code    string // Machine-readable validation code
    Message string // Human-readable description
}
```

## Usage Patterns

### Quick classification with `errors.Is`

Use sentinel errors for branching logic that does not need error details:

```go
me, err := client.Me.Get(ctx)
switch {
case errors.Is(err, ic.ErrAuthentication):
    // token invalid or expired -- prompt re-authentication
case errors.Is(err, ic.ErrRateLimit):
    // back off (the SDK has already retried if retry is enabled)
case errors.Is(err, ic.ErrNotFound):
    // resource does not exist
case err != nil:
    return fmt.Errorf("unexpected: %w", err)
}
```

### Detailed inspection with `errors.As`

Use `errors.As` to access type-specific fields:

```go
var rl *ic.RateLimitError
if errors.As(err, &rl) {
    log.Printf("rate-limited on %s (request %s), retry after %v",
        rl.Operation(), rl.RequestID(), rl.RetryAfter)
}

var ve *ic.ValidationError
if errors.As(err, &ve) {
    for _, fe := range ve.FieldErrors {
        log.Printf("field %s: %s (%s)", fe.Field, fe.Message, fe.Code)
    }
}
```

### Combined pattern

A common production pattern handles the 2-3 cases that matter and lets everything else propagate:

```go
u, err := client.Me.Get(ctx)
switch {
case errors.Is(err, ic.ErrAuthentication):
    // re-authenticate and retry
case errors.Is(err, ic.ErrRateLimit):
    var rl *ic.RateLimitError
    errors.As(err, &rl)
    time.Sleep(rl.RetryAfter)
case err != nil:
    return err
}
```

## ConfigurationError

`*ConfigurationError` is returned **only** from `NewClient` and option validation. It implements `error` but **not** `APIError`, because it represents a local construction failure rather than a backend response. Examples: nil `CredentialProvider`, non-HTTPS base URL, invalid [[Retry Policy]] parameters.

## UnexpectedError

`*UnexpectedError` is the catch-all for HTTP status codes that do not map to one of the seven known sentinels. This can happen during backend schema drift or when the API returns an undocumented status code. The `CauseBody` field holds up to 4 KiB of the response body for debugging -- it is deliberately bounded to prevent unbounded memory growth on large error payloads.

`UnexpectedError` implements `APIError` but does **not** unwrap to any sentinel. Consumers should handle it via a `default` branch or a generic `err != nil` check.

## Error Extensibility

New sentinel errors and concrete types may be added in minor SDK releases without a breaking change. Callers that switch exhaustively on sentinels should always include a `default` case to handle future additions gracefully.

## See Also

- [[01 Architecture/HTTP Transport]] -- where errors are mapped from HTTP responses
- [[02 Decisions/ADR-008 Typed Error Hierarchy with Sentinels]] -- the ADR justifying this design
- [[Retry Policy]] -- how transient errors trigger automatic retries
- [[04 Operations/Compliance Controls]] -- how `RequestID` supports audit requirements
- [[SDK Surface]] -- the full public API contract
