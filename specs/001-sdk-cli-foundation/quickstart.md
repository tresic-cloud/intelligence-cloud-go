# Quickstart: Intelligence Cloud Go SDK & `icctl`

> Target: a new developer goes from zero to first authenticated API call in under 10 minutes (SC-002).

## Prerequisites

- Go 1.22 or later installed (`go version`)
- A bearer token for Intelligence Cloud (ask your platform team; for local dev the backend can be run with mock auth)
- The base URL for your target environment:
  - **Production**: `https://api.intelligence.cloud`
  - **Staging**: `https://api.staging.intelligence.cloud`
  - **Local dev**: `http://localhost:8080`

---

## Part A — Using the SDK from Go code (≈ 3 minutes)

### 1. Add the dependency

```sh
go get github.com/tresic-cloud/intelligence-cloud-go@latest
```

### 2. Write a minimal program

`hello.go`:

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
			log.Fatal("rate-limited — try again later")
		}
		log.Fatalf("me.Get: %v", err)
	}

	fmt.Printf("Hello, %s <%s>\n", me.DisplayName, me.Email)
}
```

### 3. Run it

```sh
export IC_TOKEN="<your bearer token>"
go run hello.go
# => Hello, Jason Goecke <jason@tresic.cloud>
```

### 4. Paginated list

```go
it := client.Resellers.List(ctx, ic.WithPageSize(50))
defer it.Close()

for it.Next(ctx) {
	r := it.Value()
	fmt.Printf("%s\t%s\n", r.ID, r.Name)
}
if err := it.Err(); err != nil {
	log.Fatal(err)
}
```

### 5. Observability (opt-in)

Want OpenTelemetry spans? Supply a tracer provider from your existing setup:

```go
client, _ := ic.NewClient(
	baseURL,
	auth.StaticToken(token),
	ic.WithTracerProvider(otel.GetTracerProvider()),
	ic.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
)
```

The SDK now emits spans matching your OTel exporter (Datadog, OTLP, …). With no tracer configured, the SDK allocates nothing for telemetry.

---

## Part B — Using the `icctl` CLI (≈ 4 minutes)

### 1. Install the binary

Download the latest release archive for your platform from the [releases page](https://github.com/tresic-cloud/intelligence-cloud-go/releases) and drop `icctl` on your `$PATH`.

Verify:

```sh
icctl version
# => icctl v0.2.0 (commit abc1234, go1.23.x, linux/amd64)
```

### 2. Create a profile

```sh
icctl profile add staging --environment staging --token-stdin
# paste your token, press Ctrl-D
```

This stores non-secret profile data in `~/.config/icctl/config.yaml` (mode `0600`) and the token in your OS keychain (macOS Keychain / Windows Credential Manager / libsecret on Linux). If no keychain is reachable it falls back to `~/.config/icctl/credentials.yaml` (also `0600`).

### 3. Test the profile

```sh
icctl profile test staging
# => OK — authenticated as Jason Goecke <jason@tresic.cloud>
```

### 4. Read — human-readable

```sh
icctl --profile staging resellers list
# ID                NAME              STATUS   CREATED
# rsl_01HW3…        Acme Bakery       active   2025-12-04
# rsl_01HX7…        Blue Harbor Pizza active   2026-01-17
```

### 5. Read — scriptable

```sh
icctl --profile staging resellers list --output json | jq '.items[] | {id, name}'
# {"id":"rsl_01HW3…","name":"Acme Bakery"}
# {"id":"rsl_01HX7…","name":"Blue Harbor Pizza"}
```

### 6. Destructive — with preview (FR-015a)

```sh
icctl --profile staging resellers delete rsl_01HW3ABCDEF
```

You'll see:

```
The following operation is about to be performed:

  Environment:    staging (https://api.staging.intelligence.cloud)
  Operation:      Delete reseller
  Resource type:  reseller
  Resource id:    rsl_01HW3ABCDEF
  Irreversible:   yes
  Attributes:
    name:         Acme Bakery, Inc.
    status:       active

Proceed? (y/N):
```

Type `y` to confirm. For scripted usage, pass `--yes` or set `ICCTL_ASSUME_YES=1`:

```sh
icctl --profile staging resellers delete rsl_01HW3ABCDEF --yes
```

Non-TTY invocations (piped input, CI) without `--yes` refuse with exit code 2 — no accidental destructive calls in logs.

### 7. Switch environments

```sh
icctl profile add prod --environment production --token-stdin
icctl profile set-default prod

# or per-invocation override:
icctl --environment production resellers list
```

---

## Next steps

- **Error handling patterns**: see `docs/architecture.md#error-handling`
- **Custom retry policy**: `ic.WithRetryPolicy(ic.RetryPolicy{…})`
- **Testing against a mock server**: examples in `examples/test-mock/`
- **Full CLI reference**: `icctl help` or `docs/cli-reference.md`
- **Full SDK reference**: [pkg.go.dev/github.com/tresic-cloud/intelligence-cloud-go](https://pkg.go.dev/github.com/tresic-cloud/intelligence-cloud-go)

If you hit a problem, file an issue with the `X-Request-Id` from your error output — every SDK error carries one (`err.(ic.APIError).RequestID()`), and the CLI surfaces it automatically in `--output json` error payloads.
