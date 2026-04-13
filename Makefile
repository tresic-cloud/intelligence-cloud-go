.DEFAULT_GOAL := help

## help: print this help message
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'

## build: compile all packages
.PHONY: build
build:
	go build ./...

## test: run tests with race detector and coverage
.PHONY: test
test:
	go test -race -coverprofile=coverage.out ./...

## coverage: open HTML coverage report (runs test first)
.PHONY: coverage
coverage: test
	go tool cover -html=coverage.out

## lint: run golangci-lint
.PHONY: lint
lint:
	golangci-lint run ./...

## vuln: run govulncheck for known vulnerabilities
.PHONY: vuln
vuln:
	govulncheck ./...

## static: run go vet and staticcheck
.PHONY: static
static:
	go vet ./...
	staticcheck ./...

## codegen: regenerate code from OpenAPI spec
.PHONY: codegen
codegen:
	bash scripts/codegen.sh

## pull-openapi: fetch latest OpenAPI spec from sibling repo
.PHONY: pull-openapi
pull-openapi:
	bash scripts/pull-openapi.sh

## bench: run benchmarks
.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

## release-snapshot: build a local goreleaser snapshot
.PHONY: release-snapshot
release-snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not found. Install: go install github.com/goreleaser/goreleaser/v2@latest"; exit 1; }
	goreleaser release --snapshot --clean

## docs: generate CLI reference from cobra
.PHONY: docs
docs:
	@if [ -f cmd/icctl/main.go ]; then go run ./cmd/icctl -- --generate-docs docs/ 2>/dev/null || echo "CLI doc gen not yet supported"; else echo "CLI binary not available yet"; fi

## clean: remove build artifacts and coverage files
.PHONY: clean
clean:
	rm -f coverage.out
	rm -rf dist/
	rm -f *.out
