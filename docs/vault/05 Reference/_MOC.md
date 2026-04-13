---
title: Reference MOC
type: moc
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - moc
  - reference
aliases:
  - Reference Index
related:
  - "[[Home]]"
  - "[[01 Architecture/_MOC|Architecture MOC]]"
  - "[[02 Decisions/_MOC|Decisions MOC]]"
---

# Reference

Factual, technical reference notes for the SDK's public surface. Each note here is canonical -- if these contradict the source spec, the source wins and these get updated.

## Terminology

- [[Glossary]] -- key terms and abbreviations used across this vault and the codebase

## SDK Public API

- [[Error Hierarchy]] -- sentinel errors, concrete types, `errors.Is` / `errors.As` patterns
- [[Retry Policy]] -- backoff math, `Retry-After` parsing, jitter strategies, validation rules
- [[Pagination]] -- `Iterator[T]` surface, consumer loop, memory guarantees
- [[Authentication]] -- `CredentialProvider` model, token lifecycle, built-in adapters

## API Surface

- [[Operation IDs]] -- the 21 OpenAPI operations in v0.1, organised by resource area

## Related MOCs

- [[01 Architecture/_MOC|Architecture MOC]] -- system design and transport pipeline
- [[02 Decisions/_MOC|Decisions MOC]] -- ADRs that justify these reference designs
- [[03 Guides/_MOC|Guides MOC]] -- step-by-step walkthroughs for common tasks
- [[04 Operations/_MOC|Operations MOC]] -- compliance, security, and observability
- [[06 Project/_MOC|Project MOC]] -- roadmap, status, and governance
