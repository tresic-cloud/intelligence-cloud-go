---
title: Operations MOC
type: moc
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - moc
  - operations
aliases:
  - Operations Index
related:
  - "[[Home]]"
  - "[[01 Architecture/_MOC|Architecture MOC]]"
  - "[[02 Decisions/_MOC|Decisions MOC]]"
---

# Operations

Operational concerns: compliance posture, security hardening, and how to interpret what the SDK and CLI emit at runtime.

## Compliance & Security

- [[Compliance Controls]] -- controls matrix mapping every security concern to its implementation and verification strategy
- [[Security Posture]] -- secrets lifecycle, redaction rules, transport security, and supply-chain signing

## Observability

- [[Observability Runbook]] -- how to read OTel spans, correlate logs to traces, and diagnose common failure patterns

## Related MOCs

- [[Home]] -- vault root
- [[01 Architecture/_MOC|Architecture MOC]] -- system design and cross-cutting concerns that underpin these operational controls
- [[02 Decisions/_MOC|Decisions MOC]] -- ADRs justifying keychain storage, OTel integration, and release signing
- [[03 Guides/_MOC|Guides MOC]] -- step-by-step walkthroughs for SDK and CLI usage
- [[05 Reference/_MOC|Reference MOC]] -- detailed reference for errors, retry, pagination, and authentication
- [[06 Project/_MOC|Project MOC]] -- roadmap, status, and governance
