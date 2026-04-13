# Contract: SDK Public Go API

**Module path**: `github.com/tresic-cloud/intelligence-cloud-go`
**Root package**: `intelligencecloud`
**Stability**: this document is the surface the semver contract (FR-009) attaches to. Any change here between stable releases requires a major-version bump.

> Pseudocode below uses Go syntax for communication; exact exported names are authoritative.

## Package layout (public import paths)

| Import path | Purpose |
|---|---|
| `github.com/tresic-cloud/intelligence-cloud-go` | Root package — `Client`, options, errors, iterator, retry policy |
| `.../auth` | `CredentialProvider` interface + built-in adapters |
| `.../intelligencecloudtest` (optional, later release) | Test doubles (server stub); out of scope for v0.1 |

Everything under `.../internal/...` is NOT public — consumers MUST NOT import it.

## 1. Client construction

```go
// NewClient constructs a Client. baseURL is required and must be https://
// unless the host is localhost. provider is required.
func NewClient(baseURL string, provider auth.CredentialProvider, opts ...ClientOption) (*Client, error)

// Option setters (functional-options pattern):
func WithHTTPClient(c *http.Client) ClientOption
func WithRetryPolicy(p RetryPolicy) ClientOption
func WithDisableRetry() ClientOption
func WithTracerProvider(tp trace.TracerProvider) ClientOption
func WithPropagator(p propagation.TextMapPropagator) ClientOption
func WithLogger(l *slog.Logger) ClientOption
func WithUserAgent(ua string) ClientOption
```

**Error modes**: returns `*ConfigurationError` if baseURL is invalid or provider is nil.

## 2. Resource services

Each resource service is reached as a field on `*Client`:

```go
client.Me
client.Resellers
client.Companies
client.Locations
client.Products
client.Verticals
client.Connectors
client.AuditLogs
client.Users
client.Auth
```

Each service exposes CRUD + list verbs that mirror the canonical OpenAPI `operationId`s. Representative shape (actual operations tracked by the generated layer; FR-003 says all operations are covered):

```go
// Example: Resellers
func (s *ResellerService) List(ctx context.Context, opts ...ListOption) *Iterator[Reseller]
func (s *ResellerService) Get(ctx context.Context, id string) (*Reseller, error)
func (s *ResellerService) Create(ctx context.Context, req CreateResellerRequest) (*Reseller, error)
func (s *ResellerService) Update(ctx context.Context, id string, req UpdateResellerRequest) (*Reseller, error)
func (s *ResellerService) Delete(ctx context.Context, id string) error
func (s *ResellerService) Deactivate(ctx context.Context, id string) (*Reseller, error)
func (s *ResellerService) Reactivate(ctx context.Context, id string) (*Reseller, error)

// Me
func (s *MeService) Get(ctx context.Context) (*Me, error)
func (s *MeService) UpdateNotificationPreferences(ctx context.Context, req NotificationPreferences) (*NotificationPreferences, error)
// … etc.
```

### 2.1 Call options

```go
type CallOption interface{ applyCall(*callConfig) }

func WithIdempotencyKey(key string) CallOption            // passes X-Idempotency-Key header
func WithRequestTimeout(d time.Duration) CallOption        // per-call timeout, scoped via ctx.WithTimeout
func WithExtraHeader(name, value string) CallOption        // for future-proofing; duplicate Authorization forbidden
```

### 2.2 List options

```go
type ListOption interface{ applyList(*listConfig) }

func WithPageSize(n int) ListOption     // requires n > 0
func WithMaxItems(n int) ListOption     // requires n >= 0; 0 = unlimited
func WithFilter(key, value string) ListOption   // backend-defined filter params
```

## 3. Authentication (`.../auth`)

```go
package auth

type Token struct {
    AccessToken string
    ExpiresAt   time.Time
}

type CredentialProvider interface {
    Token(ctx context.Context) (Token, error)
}

func StaticToken(s string) CredentialProvider
func RefreshFunc(fn func(ctx context.Context) (Token, error)) CredentialProvider
```

**Guarantee**: the client never logs or traces the `AccessToken`.

## 4. Errors

```go
// Sentinel errors — safe for errors.Is
var (
    ErrAuthentication = errors.New("intelligencecloud: authentication failed")
    ErrAuthorization  = errors.New("intelligencecloud: not authorized")
    ErrValidation     = errors.New("intelligencecloud: validation failed")
    ErrNotFound       = errors.New("intelligencecloud: resource not found")
    ErrConflict       = errors.New("intelligencecloud: resource conflict")
    ErrRateLimit      = errors.New("intelligencecloud: rate-limited")
    ErrServer         = errors.New("intelligencecloud: server error")
)

// APIError is satisfied by every non-nil error the SDK returns from an API call.
type APIError interface {
    error
    Status() int
    Code() string
    RequestID() string
    Operation() string
}

// Concrete types — accessible via errors.As
type AuthenticationError struct { /* … */ }
type AuthorizationError  struct { RequiredScopes []string; /* … */ }
type ValidationError     struct { FieldErrors []FieldError; /* … */ }
type NotFoundError       struct { ResourceType, ResourceID string; /* … */ }
type ConflictError       struct { /* … */ }
type RateLimitError      struct { RetryAfter time.Duration; /* … */ }
type ServerError         struct { /* … */ }
type UnexpectedError     struct { CauseBody []byte; /* … */ }

type FieldError struct {
    Field   string
    Code    string
    Message string
}

// ConfigurationError — returned only from NewClient / option validation; does NOT implement APIError.
type ConfigurationError struct { /* … */ }
```

**Consumer patterns**:

```go
u, err := client.Me.Get(ctx)
switch {
case errors.Is(err, intelligencecloud.ErrAuthentication):
    // re-authenticate and retry
case errors.Is(err, intelligencecloud.ErrRateLimit):
    var rl *intelligencecloud.RateLimitError
    errors.As(err, &rl)
    time.Sleep(rl.RetryAfter)
case err != nil:
    return err
}
```

## 5. Pagination

```go
type Iterator[T any] struct{ /* … */ }

func (it *Iterator[T]) Next(ctx context.Context) bool
func (it *Iterator[T]) Value() T
func (it *Iterator[T]) Err() error
func (it *Iterator[T]) PageInfo() PageInfo
func (it *Iterator[T]) Close()

type PageInfo struct {
    ItemsFetched   int
    PagesFetched   int
    HasNextPage    bool
    NextPageToken  string
}
```

**Consumer pattern**:

```go
it := client.Resellers.List(ctx, intelligencecloud.WithPageSize(50))
defer it.Close()
for it.Next(ctx) {
    r := it.Value()
    // …
}
if err := it.Err(); err != nil {
    return err
}
```

## 6. Retry policy

```go
type RetryPolicy struct {
    Enabled          bool
    MaxAttempts      int
    BaseDelay        time.Duration
    MaxDelay         time.Duration
    Jitter           JitterStrategy
    RetryableStatus  []int
    HonourRetryAfter bool
}

type JitterStrategy int
const (
    NoJitter    JitterStrategy = iota
    EqualJitter
    FullJitter
)

func DefaultRetryPolicy() RetryPolicy   // {Enabled: true, MaxAttempts: 4, BaseDelay: 100ms, MaxDelay: 30s, Jitter: FullJitter, RetryableStatus: {429,500,502,503,504}, HonourRetryAfter: true}
func NoRetry() RetryPolicy              // Enabled=false
```

## 7. Observability

Controlled via `WithTracerProvider`, `WithPropagator`, and `WithLogger`. When none is supplied, the SDK is silent and allocates nothing for telemetry beyond tiny sentinel no-ops. See `contracts/telemetry.md` for the span attribute contract.

## 8. Stability markers

Experimental symbols carry:

```go
// Experimental: This API may change without notice. Exclude from production
// code paths that depend on stable behaviour.
```

…and live behind `//go:build experimental`. Consumers opt in with `go build -tags=experimental`.

## 9. Compatibility guarantees

Per FR-009:
- No removal or renaming of any symbol in this contract across minor or patch releases of a stable (`v1+`) major version.
- Error concrete-type field additions are non-breaking.
- New sentinel errors may be added in minor releases; callers that switch exhaustively on sentinels should use `default`.
- `ClientOption`, `ListOption`, and `CallOption` are open interfaces; additions are non-breaking.
