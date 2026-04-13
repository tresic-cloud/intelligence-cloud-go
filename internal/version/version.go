// Package version exposes build-time version information injected via
// Go linker flags (ldflags). When the binary is built without ldflags
// (e.g. via plain "go run" or "go test"), the variables retain their
// default development values.
//
// At release time, .goreleaser.yaml (or the Makefile "build" target)
// passes ldflags that set Version, Commit, and Date to the tag name,
// full commit SHA, and RFC 3339 UTC build timestamp respectively:
//
//	-ldflags "-s -w \
//	  -X github.com/tresic-cloud/intelligence-cloud-go/internal/version.Version={{.Tag}} \
//	  -X github.com/tresic-cloud/intelligence-cloud-go/internal/version.Commit={{.FullCommit}} \
//	  -X github.com/tresic-cloud/intelligence-cloud-go/internal/version.Date={{.CommitDate}}"
//
// Consumers within this module (e.g. the icctl CLI "version" subcommand
// and the SDK's default User-Agent header) call String() to obtain a
// human-readable combined version string.
package version

import "fmt"

// Version is the semantic version tag (e.g. "v0.1.0"). It defaults to
// "dev" when no ldflags are applied.
var Version = "dev"

// Commit is the full git commit SHA of the build. It defaults to
// "unknown" when no ldflags are applied.
var Commit = "unknown"

// Date is the RFC 3339 UTC timestamp of the build. It defaults to
// "unknown" when no ldflags are applied.
var Date = "unknown"

// String returns a human-readable combined version string in the format
// "v0.1.0 (abc1234, 2026-04-13T10:00:00Z)".
func String() string {
	return fmt.Sprintf("%s (%s, %s)", Version, Commit, Date)
}
