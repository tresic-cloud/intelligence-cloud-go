---
title: Home
type: moc
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - moc
  - home
aliases:
  - Index
  - Start Here
related:
  - "[[_README]]"
---

# Intelligence Cloud Go SDK & CLI

Welcome to the engineering knowledge base for `intelligence-cloud-go` -- the official Go SDK and CLI (`icctl`) for the Intelligence Cloud platform.

This vault is organised for two audiences: **developers** who consume the SDK from Go application code, and **operators** who use the CLI for ad-hoc and scripted platform workflows. Whether you are onboarding for the first time or looking up a specific retry formula, start here and follow the links.

## Architecture

Understand how the SDK and CLI are structured, what surfaces they expose, and how cross-cutting concerns (transport, telemetry, error handling) are layered.

- [[01 Architecture/_MOC|Architecture MOC]]
- [[Overview]] -- high-level system picture
- [[SDK Surface]] -- public Go API contract
- [[CLI Surface]] -- `icctl` command tree
- [[HTTP Transport]] -- RoundTripper pipeline
- [[Telemetry Contract]] -- OTel spans and structured logging

## Decisions

Every significant technical choice is recorded as an Architectural Decision Record (ADR). Eleven decisions underpin the SDK's foundation.

- [[02 Decisions/_MOC|Decisions MOC]]

## Guides

Step-by-step walkthroughs for common tasks.

- [[03 Guides/_MOC|Guides MOC]]
- [[Quickstart - SDK]] -- your first API call from Go code
- [[Quickstart - CLI]] -- install `icctl` and query the platform
- [[Contributing]] -- dev setup, commit conventions, branch naming
- [[Releasing]] -- release-please flow and binary verification

## Operations

Compliance, security, and observability documentation for platform and security teams.

- [[04 Operations/_MOC|Operations MOC]]
- [[Compliance Controls]] -- controls matrix from the compliance review
- [[Security Posture]] -- threat model, secrets handling, redaction
- [[Observability Runbook]] -- interpreting OTel spans and logs

## Reference

Detailed reference material for daily development.

- [[05 Reference/_MOC|Reference MOC]]
- [[Glossary]] -- key terms and abbreviations
- [[Error Hierarchy]] -- eight error types, sentinels, and usage patterns
- [[Retry Policy]] -- backoff math and Retry-After parsing
- [[Pagination]] -- Iterator[T] usage
- [[Authentication]] -- CredentialProvider model
- [[Operation IDs]] -- the 21 OpenAPI operations

## Project

Roadmap, status, and governance.

- [[06 Project/_MOC|Project MOC]]
- [[Roadmap]] -- waves 0 through 6 with current status
- [[Status]] -- implementation state and coverage
- [[Constitution Compliance]] -- how each of the 11 principles is met
- [[Linked Issues]] -- task ID to GitHub issue mapping
