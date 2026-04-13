---
title: Retry Policy
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - reference
  - sdk
  - retry
  - resilience
aliases:
  - Retry
  - Backoff
  - RetryPolicy
related:
  - "[[01 Architecture/HTTP Transport]]"
  - "[[02 Decisions/ADR-006 RoundTripper-based Retry with Full Jitter]]"
  - "[[Error Hierarchy]]"
  - "[[SDK Surface]]"
  - "[[Authentication]]"
---

# Retry Policy

The SDK automatically retries transient failures using a configurable `RetryPolicy` struct. Retry logic lives in the [[01 Architecture/HTTP Transport]] pipeline as a `http.RoundTripper` layer, so it applies uniformly to every API call without per-method duplication.

## RetryPolicy Struct

```go
type RetryPolicy struct {
    Enabled          bool           // Master switch; false disables all retry
    MaxAttempts      int            // Total attempts including the initial request (1 = no retries)
    BaseDelay        time.Duration  // Starting backoff delay before jitter
    MaxDelay         time.Duration  // Upper bound on computed backoff
    Jitter           JitterStrategy // Jitter algorithm applied to backoff
    RetryableStatus  []int          // HTTP status codes that trigger retry
    HonourRetryAfter bool           // Whether to parse and obey Retry-After headers
}
```

## Default Policy

The SDK ships a sensible default via `DefaultRetryPolicy()`:

| Field | Default value | Notes |
|---|---|---|
| `Enabled` | `true` | Retry is on by default |
| `MaxAttempts` | `4` | 1 initial request + 3 retries |
| `BaseDelay` | `100ms` | Starting delay before jitter |
| `MaxDelay` | `30s` | Absolute ceiling on any single sleep |
| `Jitter` | `FullJitter` | Best thundering-herd avoidance per AWS study |
| `RetryableStatus` | `[429, 500, 502, 503, 504]` | Rate-limit and server errors |
| `HonourRetryAfter` | `true` | Backend-specified delays are respected |

To disable retry entirely, use `WithDisableRetry()` or pass `NoRetry()`.

## Backoff Math

The SDK uses exponential backoff with configurable jitter. The delay for attempt `n` (zero-indexed, where attempt 0 is the first retry after the initial request) is computed as:

```
delay = random(0, min(MaxDelay, BaseDelay * 2^attempt))
```

This is the **full jitter** formula from the [AWS Architecture Blog -- Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/). Full jitter is proven to minimise correlated retry storms across many clients hitting the same backend.

### Concrete example (defaults)

| Attempt | Raw backoff | Jitter range (FullJitter) |
|---|---|---|
| 0 (first retry) | 100 ms | [0, 100 ms] |
| 1 | 200 ms | [0, 200 ms] |
| 2 | 400 ms | [0, 400 ms] |
| 3 | 800 ms | [0, 800 ms] |

All values are capped at `MaxDelay` (30 s by default).

## JitterStrategy Constants

```go
type JitterStrategy int

const (
    NoJitter    JitterStrategy = iota // delay = min(MaxDelay, BaseDelay * 2^attempt)
    EqualJitter                       // delay = half + random(0, half)
    FullJitter                        // delay = random(0, min(MaxDelay, BaseDelay * 2^attempt))
)
```

`FullJitter` is the recommended default. `NoJitter` and `EqualJitter` are available for controlled testing or specific operational requirements.

## Retry-After Parsing

When `HonourRetryAfter` is `true` and the backend returns a `Retry-After` header on a 429 or 5xx response, the SDK parses it and uses it instead of the computed backoff for that single attempt.

Two formats are supported per RFC 7231 section 7.1.3:

- **Integer seconds**: `Retry-After: 120` -- sleep 120 seconds
- **HTTP-date**: `Retry-After: Sun, 13 Apr 2026 14:30:00 GMT` -- sleep until that instant

The parsed delay is **clamped** to `MaxDelay`. If `Retry-After` exceeds `MaxDelay`, the SDK sleeps for `MaxDelay` and then retries (or gives up if attempts are exhausted).

## 401 Re-fetch Path

The 401 handling is **separate** from the general retry loop. When the backend returns a 401:

1. The transport invalidates the cached credential.
2. It re-invokes `CredentialProvider.Token(ctx)` to obtain a fresh token.
3. If the provider returns a new, different token, the request is retried **once** with the new token.
4. If the second attempt also returns 401, or if the provider returns the **same** token (same-token detection guards against infinite loops), the SDK surfaces a `*AuthenticationError`.

This bounds the re-fetch loop to exactly one additional attempt. The 401 re-fetch does **not** consume a retry attempt from `MaxAttempts` -- it is an orthogonal credential-refresh mechanism. See [[Authentication]] for the full token lifecycle.

## Context Deadline Always Wins

If the caller's `context.Context` is cancelled or its deadline is reached during a retry sleep, the sleep is aborted immediately and `ctx.Err()` is returned (wrapped). The SDK never sleeps past a context deadline, regardless of the computed backoff or `Retry-After` value.

## Validation

The `Validate()` method is called by `NewClient` during construction. It rejects invalid policies with a `*ConfigurationError`:

| Rule | Rejection |
|---|---|
| `MaxAttempts < 1` | At least one attempt is required |
| `BaseDelay <= 0` | Base delay must be positive |
| `MaxDelay < BaseDelay` | Max delay must be at least as large as base delay |

## Usage Example

```go
client, err := ic.NewClient(
    "https://api.intelligence.cloud",
    provider,
    ic.WithRetryPolicy(ic.RetryPolicy{
        Enabled:          true,
        MaxAttempts:       2,
        BaseDelay:         250 * time.Millisecond,
        MaxDelay:          10 * time.Second,
        Jitter:            ic.FullJitter,
        RetryableStatus:   []int{429, 500, 502, 503, 504},
        HonourRetryAfter:  true,
    }),
)
if err != nil {
    // *ConfigurationError if policy is invalid
    log.Fatal(err)
}
```

## Retry Observability

When retries occur, the [[01 Architecture/HTTP Transport]] emits:

- **OTel span events**: `retry.attempt_failed` after each failed attempt, `retry.giving_up` when retries are exhausted.
- **Structured log records**: `WARN`-level entries with operation, attempt number, backoff duration, and cause.

These are only emitted when a `TracerProvider` or `Logger` is configured on the client.

## See Also

- [[01 Architecture/HTTP Transport]] -- the transport pipeline where retry logic executes
- [[02 Decisions/ADR-006 RoundTripper-based Retry with Full Jitter]] -- the ADR justifying full jitter and RoundTripper placement
- [[Error Hierarchy]] -- how exhausted retries surface as typed errors
- [[Authentication]] -- the 401 re-fetch path in detail
- [[SDK Surface]] -- `WithRetryPolicy` and `WithDisableRetry` options
