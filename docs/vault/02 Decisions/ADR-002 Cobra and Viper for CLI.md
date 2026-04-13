---
title: "ADR-002: Cobra and Viper for CLI"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-002
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[CLI Surface]]"
tags:
  - decision/accepted
  - cli
  - cobra
  - viper
aliases:
  - Cobra ADR
  - CLI Framework ADR
  - R2
related:
  - "[[CLI Surface]]"
  - "[[Overview]]"
---

# ADR-002: Cobra and Viper for CLI

## Status

Accepted

## Context

The CLI (`icctl`) needs a command framework that supports deeply nested subcommands (one per resource, each with CRUD verbs), flag parsing, environment variable binding, configuration file layering, and automatic help generation. The CLI is expected to grow beyond 50 subcommands as the backend API surface expands.

The specification requires that every subcommand print a useful help message with flags, descriptions, and at least one example, and that configuration precedence follow: flag > environment variable > profile > built-in default.

## Decision

Use `github.com/spf13/cobra` for the command tree and flag parsing, and `github.com/spf13/viper` for configuration loading (YAML files, environment variable bindings, precedence resolution).

## Consequences

### Positive

- **De facto standard**: Cobra powers `kubectl`, `gh`, `hugo`, `docker`, and `pulumi`. Operators encounter a familiar CLI experience.
- **Built-in help**: Cobra's help generator produces the long-form usage output that the specification requires, including flag tables and example blocks.
- **Config layering**: Viper handles flag > env > file precedence without custom logic.
- **Ecosystem**: extensive documentation, community support, and integration patterns.

### Negative

- **Dependency weight**: Cobra + Viper pull in a non-trivial dependency tree, though this only affects the CLI binary, not the SDK library.

## Alternatives Considered

### urfave/cli/v3

Lighter than Cobra, but lacks the hierarchical help story needed for deeply nested resource verbs. Would require custom code to match the help format the specification demands.

### alecthomas/kong

Elegant struct-tag design with a smaller ecosystem. Its declarative style is appealing but less flexible for the dynamic subcommand registration needed as the OpenAPI surface grows.

### Standard library `flag`

Insufficient for 50+ subcommands. No built-in support for subcommand nesting, environment variable binding, or configuration file loading.

## References

- [[CLI Surface]]
- [Cobra on GitHub](https://github.com/spf13/cobra)
- [Viper on GitHub](https://github.com/spf13/viper)
