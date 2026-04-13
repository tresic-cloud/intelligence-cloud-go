---
title: SDK Surface
type: architecture
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - architecture
  - sdk
  - api
  - go
aliases:
  - SDK API
  - Public API
  - Go API Contract
related:
  - "[[Overview]]"
  - "[[CLI Surface]]"
  - "[[Error Hierarchy]]"
  - "[[Pagination]]"
  - "[[Authentication]]"
  - "[[Retry Policy]]"
---

# SDK Surface

The SDK's public Go API is the surface that the semver contract attaches to. Any change to an exported symbol between stable releases requires a major version bump. This note describes the shape of that surface.

**Module path**: `github.com/tresic-cloud/intelligence-cloud-go`
**Root package**: `intelligencecloud`

## Public Import Paths

| Import path | Purpose |
|---|---|
| `github.com/tresic-cloud/intelligence-cloud-go` | Root -- `Client`, options, errors, iterator, retry policy |
| `.../auth` | `CredentialProvider` interface + built-in adapters |

Everything under `.../internal/...` is private. Consumers must not import it.

## Client Construction

```go
func NewClient(baseURL string, provider auth.CredentialProvider, opts ...ClientOption) (*Client, error)
```

The `baseURL` must be `https://` unless the host is `localhost` or `127.0.0.1`. The `provider` is required -- pass `auth.StaticToken("...")` for simple cases or `auth.RefreshFunc(fn)` for token refresh. Returns a `*ConfigurationError` if validation fails.

### Functional Options

| Option | Effect |
|---|---|
| `WithHTTPClient(c)` | Custom `*http.Client` (e.g. for proxies or test doubles) |
| `WithRetryPolicy(p)` | Override the default retry policy |
| `WithDisableRetry()` | Turn off automatic retry entirely |
| `WithTracerProvider(tp)` | Enable OTel span emission |
| `WithPropagator(p)` | Custom trace-context propagator |
| `WithLogger(l)` | Inject a `*slog.Logger` for structured debug/warn logs |
| `WithUserAgent(ua)` | Override the default User-Agent string |

## Resource Services

Each resource area is accessed as a field on `*Client`:

| Field | Resource area |
|---|---|
| `client.Me` | Current user profile, avatar, notification preferences |
| `client.Resellers` | Reseller CRUD, deactivation, reactivation, audit logs |
| `client.Companies` | Company CRUD, connectors |
| `client.Locations` | Location CRUD, connectors |
| `client.Products` | Product CRUD |
| `client.Verticals` | Vertical CRUD |
| `client.Connectors` | Connector CRUD |
| `client.AuditLogs` | Audit log search |
| `client.Users` | User CRUD |
| `client.Auth` | Login |

Each service exposes typed methods that mirror the OpenAPI `operationId`s. For example:

```go
// Resellers
func (s *ResellerService) List(ctx, ...ListOption) *Iterator[Reseller]
func (s *ResellerService) Get(ctx, id string) (*Reseller, error)
func (s *ResellerService) Create(ctx, req CreateResellerRequest) (*Reseller, error)
func (s *ResellerService) Update(ctx, id string, req UpdateResellerRequest) (*Reseller, error)
func (s *ResellerService) Delete(ctx, id string) error
func (s *ResellerService) Deactivate(ctx, id string) (*Reseller, error)
func (s *ResellerService) Reactivate(ctx, id string) (*Reseller, error)
```

For the full list of operations, see [[Operation IDs]].

## Call Options

Per-call overrides are applied as variadic arguments:

```go
func WithIdempotencyKey(key string) CallOption  // X-Idempotency-Key header
func WithRequestTimeout(d time.Duration) CallOption
func WithExtraHeader(name, value string) CallOption
```

## List Options

Control pagination behaviour on `List*` methods:

```go
func WithPageSize(n int) ListOption    // n > 0; hints backend page size
func WithMaxItems(n int) ListOption    // n >= 0; 0 = unlimited
func WithFilter(key, value string) ListOption
```

## Error Handling

Every non-nil error from an API call implements the `APIError` interface. Seven sentinel errors enable `errors.Is` matching; eight concrete types enable `errors.As` for detailed inspection. See [[Error Hierarchy]] for full details.

```go
me, err := client.Me.Get(ctx)
if errors.Is(err, ic.ErrAuthentication) {
    // re-authenticate
}
```

## Pagination

`List*` methods return `*Iterator[T]`, a generic paginator that fetches pages on demand. See [[Pagination]].

```go
it := client.Resellers.List(ctx, ic.WithPageSize(50))
defer it.Close()
for it.Next(ctx) {
    r := it.Value()
}
```

## Observability

Controlled by `WithTracerProvider`, `WithPropagator`, and `WithLogger`. When none is supplied, the SDK is silent. See [[Telemetry Contract]].

## Stability Markers

Experimental symbols carry a `// Experimental:` GoDoc prefix and live behind the `experimental` build tag. Consumers opt in with `go build -tags=experimental`. No semver compatibility commitment applies to experimental symbols. See [[ADR-004 Pre-1.0 Experimental API Policy]].

## Compatibility Guarantees

- No removal or renaming of stable symbols across minor or patch releases of a `v1+` major version.
- New error sentinel additions in minor releases are non-breaking; callers should use a `default` case in exhaustive switches.
- `ClientOption`, `ListOption`, and `CallOption` are open interfaces; additions are non-breaking.
- Error concrete-type field additions are non-breaking.
