---
title: Linked Issues
type: project
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - project
  - issues
  - github
  - tracking
aliases:
  - Issue Map
  - Task to Issue Mapping
related:
  - "[[Roadmap]]"
  - "[[Status]]"
  - "[[Constitution Compliance]]"
  - "[[01 Architecture/Overview]]"
---

# Linked Issues

The 125 tasks in `specs/001-sdk-cli-foundation/tasks.md` were fanned out to GitHub issues on 2026-04-13 under the `v0.1.0` milestone. Task IDs like `[A-1]` are the stable cross-reference; issue numbers are not (tasks were filed in order A, B, C, D, E, so issue numbers do not match task numbers).

## Label Taxonomy

Fifteen labels organise the issue backlog. Every issue carries exactly one label from each of the `stream`, `priority`, and `kind` groups, plus one or more `jira` labels for upstream traceability.

### Stream labels

| Label | Stream | Tasks |
|-------|--------|-------|
| `stream:sdk-core` | A -- SDK core (client, options, errors, retry, pagination, telemetry, auth) | 28 |
| `stream:codegen` | B -- OpenAPI codegen pipeline + resource wrappers | 23 |
| `stream:transport` | C -- HTTP transport (retry, OTel, auth inject, redaction, 401 re-fetch) | 16 |
| `stream:cli` | D -- `icctl` CLI (commands, profile store, output, destructive preview) | 32 |
| `stream:release` | E -- Release, CI, docs, semver automation, supply-chain signing | 26 |

### Priority labels

| Label | Meaning | Maps to |
|-------|---------|---------|
| `priority:p1` | MVP SDK surface (User Story 1) | 64 tasks |
| `priority:p2` | Operator-facing CLI (User Story 2) | 47 tasks |
| `priority:p3` | Release polish, docs, discoverability (User Story 3) | 14 tasks |

### Kind labels

| Label | Task types |
|-------|-----------|
| `kind:test` | Failing-test-first tasks |
| `kind:impl` | Implementation tasks |
| `kind:docs` | Documentation tasks |
| `kind:config` | Configuration and infrastructure |
| `kind:script` | Build/CI scripts |
| `kind:bench` | Benchmarks |
| `kind:refactor` | Refactoring tasks |

### Jira labels

| Label | Applied to |
|-------|-----------|
| `jira:IDB-1353` | All 125 tasks |
| `jira:IDB-1354` | Tasks also referencing the upstream OpenAPI consolidation work (B-1, B-4, B-7, B-19, D-22, D-23, D-24) |

## Finding Issues

### By task ID

Every GitHub issue title begins with the task ID in brackets. To find the issue for task `[A-1]`:

```
is:issue "[A-1]" in:title repo:tresic-cloud/intelligence-cloud-go
```

Or via the `gh` CLI:

```bash
gh issue list --search '"[A-1]" in:title' --repo tresic-cloud/intelligence-cloud-go
```

### By stream

Filter by stream label to see all issues in a stream:

```bash
gh issue list --label "stream:sdk-core" --repo tresic-cloud/intelligence-cloud-go
```

### Next actionable issues

Find unassigned P1 issues ready for work (excluding the release stream, which is gated):

```
is:issue is:open no:assignee label:priority:p1 -label:stream:release milestone:v0.1.0
```

```bash
gh issue list --search 'is:open no:assignee label:priority:p1 -label:stream:release milestone:v0.1.0' --repo tresic-cloud/intelligence-cloud-go
```

### By wave

Issues do not carry a wave label, but you can approximate by combining stream and task-ID range. See [[Roadmap]] for the wave-to-task mapping.

## Stream-to-Issue-Number Mapping

Because issues were filed in stream order (A, B, C, D, E), the approximate issue-number ranges are:

| Stream | Label | Task ID range | Approximate issue range |
|--------|-------|---------------|------------------------|
| A | `stream:sdk-core` | `[A-1]` -- `[A-28]` | #1 -- #28 |
| B | `stream:codegen` | `[B-1]` -- `[B-23]` | #29 -- #51 |
| C | `stream:transport` | `[C-1]` -- `[C-16]` | #52 -- #67 |
| D | `stream:cli` | `[D-1]` -- `[D-32]` | #68 -- #99 |
| E | `stream:release` | `[E-1]` -- `[E-26]` | #100 -- #125 |

These ranges are approximate -- the exact mapping depends on filing order and any issues that were created or deleted between streams.

## Issue Body Format

Each issue body contains:

1. The full task block from `tasks.md` (acceptance criteria, dependencies, effort estimate)
2. A `Related: IDB-1353` / `IDB-1354` trailer for Jira traceability
3. Links back to `spec.md` and `plan.md` in the feature directory

## Milestone

All 125 issues target the **`v0.1.0`** milestone. Post-v0.1 planning will split remaining work into `v0.2` (CLI polish) and `v0.3` (release polish). See [[Roadmap]] for wave details.

## See Also

- [[Roadmap]] -- wave structure and stream-to-wave mapping
- [[Status]] -- current implementation progress and issue closure count
- [[Constitution Compliance]] -- how tracking supports constitutional principles
- [[01 Architecture/Overview]] -- the system being built
