---
title: Architecture MOC
type: moc
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - moc
  - architecture
aliases:
  - Architecture Index
related:
  - "[[Home]]"
  - "[[02 Decisions/_MOC|Decisions MOC]]"
  - "[[05 Reference/_MOC|Reference MOC]]"
---

# Architecture

This section documents how the Intelligence Cloud Go SDK and CLI are structured, what public surfaces they expose, and how cross-cutting concerns are layered.

## System Design

- [[Overview]] -- high-level system picture: module layout, dependency graph, data flow
- [[HTTP Transport]] -- the RoundTripper pipeline that composes auth, retry, telemetry, and redaction

## Public Surfaces

- [[SDK Surface]] -- the Go API contract that semver attaches to
- [[CLI Surface]] -- the `icctl` command tree, flags, exit codes, and output formats

## Cross-Cutting Concerns

- [[Telemetry Contract]] -- OpenTelemetry span attributes, slog integration, and the zero-cost no-op guarantee

## Related MOCs

- [[02 Decisions/_MOC|Decisions MOC]] -- the ADRs that justify these architecture choices
- [[05 Reference/_MOC|Reference MOC]] -- detailed reference for errors, retry math, pagination, and auth
- [[04 Operations/_MOC|Operations MOC]] -- operational implications of these architecture decisions
