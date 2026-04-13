# intelligence-cloud-go

[![Go](https://img.shields.io/github/go-mod/go-version/tresic-cloud/intelligence-cloud-go)](https://go.dev/)
[![CI](https://github.com/tresic-cloud/intelligence-cloud-go/actions/workflows/ci.yaml/badge.svg)](https://github.com/tresic-cloud/intelligence-cloud-go/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tresic-cloud/intelligence-cloud-go.svg)](https://pkg.go.dev/github.com/tresic-cloud/intelligence-cloud-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

Go SDK and CLI (`icctl`) for the [Intelligence Cloud](https://intelligence.cloud) HTTP API.

The SDK is an idiomatic Go client with typed errors, automatic retry with
exponential backoff, streaming pagination, and opt-in OpenTelemetry tracing.
The CLI wraps the SDK with human-readable table output, JSON mode for
scripting, OS-keychain credential storage, and a Pulumi-style confirmation
prompt for destructive operations.

## Install

```sh
go get github.com/tresic-cloud/intelligence-cloud-go@latest
```

## SDK quickstart

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

Set your token and run:

```sh
export IC_TOKEN="<your bearer token>"
go run main.go
```

Paginated endpoints return an `Iterator[T]` that streams pages on demand:

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

## CLI quickstart

Download a binary from the [releases page](https://github.com/tresic-cloud/intelligence-cloud-go/releases) or build from source:

```sh
go install github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl@latest
```

Create a profile (the token is read from stdin so it never appears in shell history):

```sh
icctl profile add staging --environment staging --token-stdin
```

Run commands:

```sh
icctl --profile staging resellers list
icctl --profile staging resellers list --output json | jq '.items[].name'
```

Destructive operations show a preview and require confirmation before executing.
Use `--yes` or `ICCTL_ASSUME_YES=1` to bypass in CI.

## Development

Prerequisites: Go 1.25+, golangci-lint, govulncheck, staticcheck.

```sh
make help        # list all targets
make build       # compile
make test        # test with race detector + coverage
make lint        # golangci-lint
make vuln        # govulncheck
make static      # go vet + staticcheck
make codegen     # regenerate from OpenAPI spec
make bench       # benchmarks
make clean       # remove artifacts
```

## Documentation

- [Architecture and guides](./docs/vault/Home.md) (Obsidian vault)
- [SDK quickstart](./docs/vault/03%20Guides/Quickstart%20-%20SDK.md)
- [CLI quickstart](./docs/vault/03%20Guides/Quickstart%20-%20CLI.md)
- [Contributing](./docs/vault/03%20Guides/Contributing.md)
- [GoDoc (pkg.go.dev)](https://pkg.go.dev/github.com/tresic-cloud/intelligence-cloud-go)

## Contributing

Contributions are welcome. Please read the
[Contributing guide](./docs/vault/03%20Guides/Contributing.md) for dev
environment setup, the conventional-commits requirement, branch naming
conventions, TDD discipline, and the pull request process.

## License

[MIT](./LICENSE)
