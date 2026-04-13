---
title: Roadmap
type: project
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - project
  - roadmap
  - planning
aliases:
  - Project Roadmap
  - Wave Plan
related:
  - "[[Status]]"
  - "[[Linked Issues]]"
  - "[[01 Architecture/_MOC|Architecture MOC]]"
  - "[[02 Decisions/_MOC|Decisions MOC]]"
---

# Roadmap

The Intelligence Cloud Go SDK and CLI (`icctl`) are built in seven waves. Each wave groups tasks that can run in parallel, and every wave depends on the prior wave being complete. The full task backlog (125 tasks across 5 streams) lives in `specs/001-sdk-cli-foundation/tasks.md`; this note provides the high-level view.

## Wave Dependency Graph

```
Wave 0  Foundation         ─────────────────────────────────────────────┐
  (4 parallel agents)                                                   │
                                                                        v
Wave 1  Client + Transport + Codegen  ──────────────────────────────────┐
  (3 parallel agents)                                                   │
                                                                        v
Wave 2  Resource Wrappers + CLI Foundation + CI  ───────────────────────┐
  (4 parallel agents)                                                   │
                                                                        v
Wave 3  CLI Core Commands  ─────────────────────────────────────────────┐
  (3 parallel agents)                                                   │
                                                                        v
Wave 4  CLI Resource Commands + Integration Tests  ─────────────────────┐
  (3 parallel agents)                                                   │
                                                                        v
Wave 5  Release Engineering    (gated on LICENSE -- resolved as MIT)  ───┐
  (2 parallel agents)                                                    │
                                                                         v
Wave 6  First Release (v0.1.0)    manual, user-initiated  ──────────── DONE
```

## Parallelism Budget

| Wave | Parallel agents | Approximate tasks |
|------|----------------|-------------------|
| 0 | 4 | 14 |
| 1 | 3 | 28 |
| 2 | 4 | 18 |
| 3 | 3 | 18 |
| 4 | 3 | 12 |
| 5 | 2 | 12 |
| 6 | 1 (manual) | user-initiated |

---

## Wave 0 -- Foundation

**Status**: complete

Four parallel agents laid the foundation that all subsequent waves depend on.

| Agent | Scope | Key deliverables |
|-------|-------|-----------------|
| W0-1 | Module + lint + version | `go.mod`, `.golangci.yaml`, `internal/version` package |
| W0-2 | Auth package | `auth.Token`, `CredentialProvider` interface, `StaticToken`, `RefreshFunc` |
| W0-3 | Errors + retry + options | Typed error hierarchy (8 concrete types + sentinels), `RetryPolicy`, `JitterStrategy`, `ClientOption`/`CallOption`/`ListOption` |
| W0-4 | Pagination + telemetry + redaction | Generic `Iterator[T]`, OTel span helpers, slog redaction handler, transport redaction |

All Wave 0 tasks follow strict TDD -- test tasks (`[A-1]`, `[A-3]`, `[A-5]`, etc.) landed before their implementation counterparts.

Current-wave progress: [[Status]].

---

## Wave 1 -- Client + Transport + Codegen

**Status**: complete

Three parallel agents delivered the SDK's client, HTTP transport pipeline, and code generation infrastructure.

| Agent | Scope | Key deliverables |
|-------|-------|-----------------|
| W1-1 | OpenAPI codegen pipeline | Pinned OpenAPI snapshot, `oapi-codegen` config, `scripts/codegen.sh`, `scripts/pull-openapi.sh`, drift test, compilation test. Hand-curated generated code as stop-gap for the OpenAPI 3.1 collision -- see [[02 Decisions/ADR-011 Hand-curated Generated Code Stop-gap]]. |
| W1-2 | Client + NewClient + ClientOption | `Client` struct with `NewClient` constructor, functional options, default `User-Agent` header wired from `internal/version` |
| W1-3 | HTTP transport RoundTripper | Single composable `http.RoundTripper` implementing: auth injection, retry with full jitter, OTel span emission, slog integration, secret redaction, 401 re-fetch, typed error mapping. 16 tasks in one agent due to tight internal coupling. |

Current-wave progress: [[Status]].

---

## Wave 2 -- Resource Wrappers + CLI Foundation + CI

**Status**: next

Four parallel agents will wire everything together into a usable SDK surface and set up the CLI skeleton and CI pipeline.

| Agent | Scope | Key deliverables |
|-------|-------|-----------------|
| W2-1 | Resource services (Me, Resellers) | Hand-written ergonomic wrappers connecting generated code to the public `Client` |
| W2-2 | Resource services (Auth, AuditLogs, Connectors) | Additional resource wrappers |
| W2-3 | Resource services (Products, Users, scaffold Companies/Locations/Verticals) | Remaining resource area scaffolding |
| W2-4 | CI + infrastructure | 9-stage CI workflow (lint, security, static, unit, coverage, codegen-drift, example-build, cross-platform build matrix, optional integration), `Makefile`, `Dependabot`, `README.md`, ADRs |

The CI pipeline enforces the project's quality gates from day one: 80% coverage threshold, no TODO/FIXME, `govulncheck`, and codegen drift detection.

Current-wave progress: [[Status]].

---

## Wave 3 -- CLI Core Commands

**Status**: planned

Three parallel agents will build the CLI command tree, profile management, output formatters, and the destructive-verb preview/confirmation flow.

| Agent | Scope | Key deliverables |
|-------|-------|-----------------|
| W3-1 | Profile store + formatters + destructive preview | `SecretStore` (keychain + file fallback), JSON/table formatters, Pulumi-style preview + prompt |
| W3-2 | Root command + version + profile subcommand | `icctl` root, `version` subcommand, exit-code mapper, `profile` sub-tree |
| W3-3 | Completion + SIGINT + GoDoc | Shell completion generators, graceful SIGINT handling, GoDoc audit pass |

See [[01 Architecture/CLI Surface]] for the full command tree design.

Current-wave progress: [[Status]].

---

## Wave 4 -- CLI Resource Commands + Integration Tests

**Status**: planned

Three parallel agents will implement per-resource CLI subcommands and integration test suites.

| Agent | Scope | Key deliverables |
|-------|-------|-----------------|
| W4-1 | `resellers` + `me` subcommands | CRUD subcommands for resellers and me endpoints |
| W4-2 | `companies` + `locations` + remaining resources | CRUD subcommands for remaining resource areas |
| W4-3 | Integration + security tests | Help-contract test, token-never-logged assertion, quickstart integration, permissions audit |

Current-wave progress: [[Status]].

---

## Wave 5 -- Release Engineering

**Status**: planned (gated on LICENSE confirmation -- resolved as **MIT**)

Two parallel agents will deliver the release toolchain and supporting documentation.

| Agent | Scope | Key deliverables |
|-------|-------|-----------------|
| W5-1 | Release toolchain | `.goreleaser.yaml`, release workflow, `release-please`, PR title lint, build matrix |
| W5-2 | Docs + governance | Contributing guide, architecture docs, drift workflow, markdown lint, `make docs`, post-release verification, `CODEOWNERS`, security reporting |

See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]] for the release signing strategy.

Current-wave progress: [[Status]].

---

## Wave 6 -- First Release (`v0.1.0`)

**Status**: manual, user-initiated

This wave is not automated. The steps are:

1. Confirm LICENSE (MIT -- confirmed), `CODEOWNERS` team slug, and security reporting channel.
2. Merge the `release-please` PR -- this creates the `v0.1.0` tag automatically.
3. The release workflow fires: `goreleaser` builds cross-platform binaries, `cosign` signs them keylessly, and the GitHub release is published.
4. Run post-release verification (`[E-22]`): confirm the Go module proxy resolves `v0.1.0`, CLI binaries download and run, and cosign signatures verify.

See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]] and the [[Releasing]] guide for full details.

---

## Stream-to-Wave Mapping

| Stream | Description | Primary waves |
|--------|-------------|---------------|
| A (28 tasks) | SDK core -- client, options, errors, retry, pagination, telemetry, auth | 0, 1 |
| B (23 tasks) | OpenAPI codegen pipeline + resource wrappers | 1, 2 |
| C (16 tasks) | HTTP transport -- retry, OTel, auth inject, redaction, 401 re-fetch | 1 |
| D (32 tasks) | CLI -- commands, profile store, output, destructive preview | 3, 4 |
| E (26 tasks) | Release, CI, docs, semver automation, supply-chain signing | 0 (partial), 2 (partial), 5 |

## See Also

- [[Status]] -- current implementation snapshot
- [[Linked Issues]] -- task IDs to GitHub issue mapping
- [[01 Architecture/Overview]] -- system architecture that the roadmap delivers
