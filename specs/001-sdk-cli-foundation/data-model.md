# Phase 1 Data Model: Intelligence Cloud Go SDK & CLI

**Date**: 2026-04-13
**Spec**: [spec.md](./spec.md) · **Plan**: [plan.md](./plan.md) · **Research**: [research.md](./research.md)

Scope: the internal entities the SDK and CLI own. Backend resource schemas (Reseller, Company, Location, …) are generated from the canonical OpenAPI (`testdata/openapi.yaml`) into `internal/generated/types.gen.go` — they are re-exported at the package root but their *shape* is not authored here, it is owned by the backend contract.

## 1. SDK-internal entities

### 1.1 `Client`

The top-level handle consumers instantiate once per process.

| Field | Type | Notes |
|---|---|---|
| `baseURL` | `*url.URL` | HTTPS-validated at construction (localhost HTTP permitted) |
| `credentialProvider` | `auth.CredentialProvider` | Required; no default |
| `httpClient` | `*http.Client` | Default: `http.Client{}` wrapped with `transport.Transport` |
| `retryPolicy` | `RetryPolicy` | Default: `DefaultRetryPolicy()` (R4) |
| `tracerProvider` | `trace.TracerProvider` | Default: no-op |
| `propagator` | `propagation.TextMapPropagator` | Default: W3C trace-context + baggage |
| `logger` | `*slog.Logger` | Default: silent handler |
| `userAgent` | `string` | Default: `"intelligence-cloud-go/vX.Y.Z (os/arch)"` |
| `Resellers` | `*ResellerService` | Resource service, lazily wired |
| `Companies` | `*CompanyService` | " |
| `Locations` | `*LocationService` | " |
| `Products` | `*ProductService` | " |
| `Verticals` | `*VerticalService` | " |
| `Connectors` | `*ConnectorService` | " |
| `AuditLogs` | `*AuditLogService` | " |
| `Users` | `*UserService` | " |
| `Me` | `*MeService` | " |
| `Auth` | `*AuthService` | " |

**Lifecycle**: immutable after construction; safe for concurrent use by multiple goroutines. Options that change per-call (timeouts, retry overrides, request-ID propagation) are supplied via `CallOption` variadics on each method, not by mutating the client.

**Invariants**:
- `credentialProvider` is non-nil (construction returns an error otherwise).
- `baseURL.Scheme` is `https` unless host is `localhost` or `127.0.0.1` (construction returns an error otherwise).
- All resource services hold a non-owning pointer back to the client.

### 1.2 `auth.CredentialProvider` (interface)

```text
interface CredentialProvider
  Token(ctx) (Token, error)
```

```text
struct Token
  AccessToken string          // never logged, never in spans
  ExpiresAt   time.Time       // zero value = "never expires" (treated as opaque)
```

**Built-in implementations**:
- `StaticToken(s string)` — returns `{s, time.Time{}}` on every call.
- `RefreshFunc(fn func(ctx) (Token, error))` — wraps any caller-supplied refresh callback. The client maintains an in-memory cache keyed by the returned `ExpiresAt` minus a 30 s safety margin; the provider is re-invoked on expiry or on an upstream `401`.

**Contract**: the client MUST treat a returned error as a fatal authentication failure for the current call and surface it as `*AuthenticationError` with cause chain preserved.

### 1.3 `RetryPolicy`

| Field | Type | Default |
|---|---|---|
| `Enabled` | `bool` | `true` |
| `MaxAttempts` | `int` | `4` (1 initial + 3 retries) |
| `BaseDelay` | `time.Duration` | `100 * time.Millisecond` |
| `MaxDelay` | `time.Duration` | `30 * time.Second` |
| `Jitter` | `JitterStrategy` | `FullJitter` |
| `RetryableStatus` | `[]int` | `{429, 500, 502, 503, 504}` |
| `HonourRetryAfter` | `bool` | `true` |

**Behaviour**:
- If `ctx.Err() != nil` the current attempt aborts and the ctx error is surfaced — no further retries.
- `Retry-After` (when honoured) overrides computed backoff for that single attempt but is clamped by `MaxDelay`.
- Transport-level errors (DNS, connection, TLS handshake) count as retryable.
- Non-429 4xx responses are never retried regardless of `RetryableStatus`.

### 1.4 `Iterator[T]`

Generic paginator returned by every `List*` method.

```text
type Iterator[T any]
  Next(ctx) bool              // fetches next page if needed; returns false on exhaustion OR error
  Value() T                   // current item; undefined before first Next()
  Err() error                 // nil unless Next returned false due to error
  PageInfo() PageInfo         // { ItemsFetched, PagesFetched, HasNextPage, NextPageToken }
  Close()                     // releases any pooled buffers (idempotent, safe to defer)
```

**Invariants**:
- `Next` is safe to call after a prior `false` return — always returns `false`.
- `Next` must be called before the first `Value()`.
- Memory usage is bounded to one page worth of items plus the next-page token.

**List-call options** (affecting Iterator behaviour):
- `WithPageSize(n int)` — hints the backend's page size (`page_size` query param).
- `WithMaxItems(n int)` — hard cap; iterator returns `false` with nil error once reached.

### 1.5 Error hierarchy

Per R7. All concrete errors implement:

```text
interface APIError
  error
  Status() int
  Code() string
  RequestID() string
  Operation() string
```

Sentinel errors for `errors.Is`:

```text
var ErrAuthentication = errors.New("intelligencecloud: authentication failed")
var ErrAuthorization  = errors.New("intelligencecloud: not authorized")
var ErrValidation     = errors.New("intelligencecloud: validation failed")
var ErrNotFound       = errors.New("intelligencecloud: resource not found")
var ErrConflict       = errors.New("intelligencecloud: resource conflict")
var ErrRateLimit      = errors.New("intelligencecloud: rate-limited")
var ErrServer         = errors.New("intelligencecloud: server error")
```

Concrete types carry the sentinels via `Unwrap()` so `errors.Is(err, ErrAuthentication)` works, and callers who need details use `errors.As(err, &target)`:

| Concrete type | HTTP status(es) | Extra fields |
|---|---|---|
| `*AuthenticationError` | 401 | — |
| `*AuthorizationError` | 403 | `RequiredScopes []string` (if backend returns them) |
| `*ValidationError` | 400, 422 | `FieldErrors []FieldError` |
| `*NotFoundError` | 404 | `ResourceType, ResourceID string` |
| `*ConflictError` | 409 | — |
| `*RateLimitError` | 429 | `RetryAfter time.Duration` |
| `*ServerError` | 5xx | — |
| `*UnexpectedError` | anything else | `CauseBody []byte` (≤4 KiB) |

## 2. CLI-internal entities

### 2.1 `Profile`

Named, persistent configuration bundle used by the CLI.

**Non-secret fields** (stored plaintext in `$XDG_CONFIG_HOME/icctl/config.yaml`):

| Field | Type | Notes |
|---|---|---|
| `Name` | `string` | Unique within the config file. Identifier `[a-z0-9-]+`. |
| `Environment` | `string` | `production` / `staging` / `dev` / custom |
| `BaseURL` | `string` | Resolved from environment name via a built-in map; overridable |
| `DefaultOutput` | `enum{json,table}` | Default: `table` |
| `TracingEndpoint` | `string` (optional) | OTLP exporter URL for operators who want traces from their CLI calls |

**Secret fields** (stored in OS keychain with `service=icctl`, `account=<profile-name>`):

| Field | Type | Notes |
|---|---|---|
| `BearerToken` | `string` | The access token |
| `RefreshToken` | `string` (optional) | Only present when an interactive login flow was used; reserved — v0.1 does not populate it |

**Invariants**:
- Listing profiles never reveals secrets.
- Deleting a profile MUST delete both the config entry and the keychain entry; failure to delete the keychain entry is logged as a warning but does not fail the command.
- The active profile is selected by (precedence): `--profile` flag → `ICCTL_PROFILE` env → `current_profile` key in config.

### 2.2 `State: current profile`

Global non-persistent state scoped to a single CLI invocation.

| Field | Type | Notes |
|---|---|---|
| `ActiveProfile` | `*Profile` | Resolved once at command start-up |
| `EffectiveBaseURL` | `string` | After precedence: flag > env > profile |
| `EffectiveToken` | `string` | Loaded from keychain / fallback file / `ICCTL_TOKEN` env / `--token` flag |
| `EffectiveOutput` | `enum{json,table}` | After precedence: flag > profile default |
| `IsTTY` | `bool` | Reflects `os.Stdout.Fd()` — drives table rendering + confirmation prompts |
| `AssumeYes` | `bool` | Reflects `--yes`/`-y` flag and `ICCTL_ASSUME_YES` env var |

### 2.3 `DestructivePreview`

The structured plan emitted before destructive verbs (FR-015a).

| Field | Type |
|---|---|
| `Environment` | `string` (profile's environment name + base URL) |
| `Operation` | `string` (human-readable, e.g. `"Delete reseller"`) |
| `ResourceType` | `string` |
| `ResourceID` | `string` |
| `Attributes` | `map[string]string` (key/value of what is being changed or removed) |
| `Irreversible` | `bool` |

**Rendered** both as a formatted terminal preview and (under `--output json`) as a JSON object on stdout *before* the prompt is issued. Scripts can pipe the JSON form into a policy check before confirming.

## 3. Relationships

```
Client ──owns──▶ CredentialProvider
Client ──owns──▶ RetryPolicy
Client ──owns──▶ *http.Client ──wraps──▶ transport.Transport ──wraps──▶ user-supplied or stdlib RoundTripper
Client ──owns──▶ ResellerService, CompanyService, …  (resource services)
ResourceService.List*(…) ──returns──▶ Iterator[T]
ResourceService.Any*(…) ──on error──▶ APIError ──implements──▶ error (sentinel-aware)

icctl (CLI)
  Profile ──stored in──▶ config.yaml (non-secret) + keychain (secret)
  Command ──reads──▶ ActiveProfile ──derives──▶ Client
  DestructiveCommand ──builds──▶ DestructivePreview ──prompts──▶ user ──executes──▶ Client.<Delete|Deactivate|…>
```

## 4. State transitions

### 4.1 Token lifecycle (inside a single client, per R4 / R7)

```
[no-cache]
    │  first call
    ▼
[provider.Token() invoked] ──error──▶ *AuthenticationError (terminal for this call)
    │ ok
    ▼
[cache {token, expiresAt - 30s margin}]
    │
    ▼
[request sent with token]
    │
    ├── 401 response ──▶ invalidate cache ──▶ provider.Token() again ──ok──▶ retry once ──401──▶ *AuthenticationError
    ├── success       ──▶ cache retained until expiry
    └── expiry reached before next call ──▶ cache invalidated; next call re-invokes provider
```

### 4.2 Retry state machine

```
attempt = 0
├──▶ issue request
│     ├── success (2xx/3xx)     ──▶ return response
│     ├── 4xx non-429            ──▶ return typed APIError (no retry)
│     ├── 401                    ──▶ one provider re-fetch + one retry (see 4.1); else typed error
│     ├── 429 / 5xx / transport  ──▶ if attempt+1 < MaxAttempts ──▶ sleep backoff (full jitter, Retry-After honoured) ──▶ attempt++
│     └── ctx cancelled/deadline ──▶ return ctx.Err() wrapped
```

### 4.3 CLI destructive-verb flow

```
[invoke icctl <resource> <destructive-verb> <id>]
    │
    ▼
[build DestructivePreview]
    │
    ▼
[render preview to stderr (or stdout in --output json)]
    │
    ├── --yes / ICCTL_ASSUME_YES ──▶ [execute]
    ├── TTY                        ──▶ [prompt "Proceed? (y/N)"] ──y──▶ [execute] ──n──▶ exit 1
    └── non-TTY (no --yes)         ──▶ exit 2 with "refusing without --yes" error
```

## 5. Validation rules

| Rule | Enforced where |
|---|---|
| `baseURL` must be HTTPS unless host is localhost | `Client` constructor |
| `CredentialProvider` must be non-nil | `Client` constructor |
| `RetryPolicy.MaxAttempts >= 1`, `BaseDelay > 0`, `MaxDelay >= BaseDelay` | `RetryPolicy.Validate()` called by `Client` constructor |
| `Profile.Name` matches `^[a-z0-9][a-z0-9-]*$` | `profile.Store.Save()` |
| CLI destructive verbs without `--yes` on non-TTY | Pre-flight check in each destructive command; rejects before any API call |
| `ICCTL_ASSUME_YES` must be `1`, `true`, or `yes` (case-insensitive) to count as set | Parsed once at CLI init |
| List options: `WithPageSize(n)` requires `n > 0`, `WithMaxItems(n)` requires `n >= 0` | Option constructors |
