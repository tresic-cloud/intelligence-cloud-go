---
title: Quickstart - SDK
type: guide
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - guide
  - sdk
  - quickstart
aliases:
  - SDK Quickstart
  - First API Call
related:
  - "[[Quickstart - CLI]]"
  - "[[Overview]]"
  - "[[SDK Surface]]"
  - "[[05 Reference/Authentication]]"
  - "[[05 Reference/Error Hierarchy]]"
  - "[[05 Reference/Pagination]]"
---

# Quickstart: SDK

This guide takes you from zero to your first authenticated API call in under five minutes. By the end you will have a running Go program that talks to the Intelligence Cloud API, handles errors idiomatically, and iterates through a paginated list.

## Prerequisites

Before you start, make sure you have:

- **Go 1.22 or later** -- verify with `go version`.
- **A bearer token** for Intelligence Cloud. Ask your platform team, or use a mock auth token if running the backend locally.
- **The base URL** for your target environment:

| Environment | Base URL |
|---|---|
| Production | `https://api.intelligence.cloud` |
| Staging | `https://api.staging.intelligence.cloud` |
| Local dev | `http://localhost:8080` |

## Install the module

In your Go project directory:

```sh
go get github.com/tresic-cloud/intelligence-cloud-go@latest
```

This pulls the SDK and all its transitive dependencies. The module path is stable and versioned with semver tags -- see [[Releasing]] for how tags are managed.

## Your first API call

Create a file called `hello.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/auth"
)

func main() {
	token := os.Getenv("IC_TOKEN")
	if token == "" {
		log.Fatal("IC_TOKEN environment variable is required")
	}

	client, err := ic.NewClient(
		"https://api.staging.intelligence.cloud",
		auth.StaticToken(token),
	)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	me, err := client.Me.Get(ctx)
	if err != nil {
		switch {
		case errors.Is(err, ic.ErrAuthentication):
			log.Fatal("token expired or invalid")
		case errors.Is(err, ic.ErrRateLimit):
			log.Fatal("rate-limited -- try again later")
		}
		log.Fatalf("me.Get: %v", err)
	}

	fmt.Printf("Hello, %s <%s>\n", me.DisplayName, me.Email)
}
```

Run it:

```sh
export IC_TOKEN="<your bearer token>"
go run hello.go
# => Hello, Jason Goecke <jason@tresic.cloud>
```

### Why this works

- `ic.NewClient` accepts a base URL and a [[05 Reference/Authentication|CredentialProvider]]. `auth.StaticToken` is the simplest provider -- it wraps a raw bearer string. For production use with token refresh, see the `auth.RefreshFunc` adapter.
- `client.Me.Get(ctx)` is a typed method on the `Me` resource service. Every SDK operation takes a `context.Context` and respects deadlines and cancellation.
- The `errors.Is` checks use sentinel values from the [[05 Reference/Error Hierarchy|typed error hierarchy]]. Eight concrete error types cover the full HTTP status space, and every error carries the backend's `X-Request-Id` for support triage.

## Paginated lists

List endpoints return an `Iterator[T]` that fetches pages on demand. You never touch page tokens directly -- see [[05 Reference/Pagination]] for the full contract.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/auth"
)

func main() {
	client, err := ic.NewClient(
		"https://api.staging.intelligence.cloud",
		auth.StaticToken(os.Getenv("IC_TOKEN")),
	)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	it := client.Resellers.List(ctx, ic.WithPageSize(50))
	defer it.Close()

	for it.Next(ctx) {
		r := it.Value()
		fmt.Printf("%s\t%s\n", r.ID, r.Name)
	}
	if err := it.Err(); err != nil {
		log.Fatal(err)
	}
}
```

The iterator handles page boundaries transparently. Use `ic.WithMaxItems(100)` to cap the total items fetched without exhausting memory on large result sets.

## Opt-in observability

The SDK ships with a zero-cost observability contract: when no tracer is configured, span operations compile to near-zero overhead (under 100 ns/op). To enable OpenTelemetry spans, supply your existing tracer provider and logger -- see [[01 Architecture/Telemetry Contract]] for the full attribute list.

```go
package main

import (
	"log/slog"
	"os"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	"go.opentelemetry.io/otel"
)

func main() {
	client, _ := ic.NewClient(
		"https://api.staging.intelligence.cloud",
		auth.StaticToken(os.Getenv("IC_TOKEN")),
		ic.WithTracerProvider(otel.GetTracerProvider()),
		ic.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
	)
	_ = client // use client as shown above
}
```

The SDK emits spans named `intelligencecloud.{operationID}` with attributes per [[02 Decisions/ADR-007 OpenTelemetry Client Spans semconv 1.24|semconv 1.24]]. These spans work with any OTel exporter -- Datadog, Honeycomb, Jaeger, OTLP -- without additional configuration.

## Error handling in depth

Every API error implements the `APIError` interface. Beyond `errors.Is` for sentinel matching, you can use `errors.As` to extract structured details:

```go
var apiErr ic.APIError
if errors.As(err, &apiErr) {
	fmt.Printf("status=%d code=%s request_id=%s\n",
		apiErr.StatusCode(),
		apiErr.Code(),
		apiErr.RequestID(),
	)
}
```

See [[05 Reference/Error Hierarchy]] for the complete hierarchy and [[02 Decisions/ADR-008 Typed Error Hierarchy with Sentinels|ADR-008]] for the design rationale.

## Next steps

- **CLI quickstart**: if you prefer a terminal workflow, see [[Quickstart - CLI]].
- **Custom retry policy**: the SDK auto-retries 5xx, 429, and transport errors with exponential backoff and full jitter. Tune via `ic.WithRetryPolicy(...)` -- see [[05 Reference/Retry Policy]].
- **Contributing**: ready to hack on the SDK itself? Start with [[Contributing]].
- **Architecture**: understand the full layering in [[Overview]].
