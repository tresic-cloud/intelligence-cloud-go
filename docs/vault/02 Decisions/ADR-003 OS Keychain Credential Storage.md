---
title: "ADR-003: OS Keychain Credential Storage"
type: adr
status: active
created: 2026-04-13
updated: 2026-04-13
adr_id: ADR-003
adr_status: accepted
decision_date: 2026-04-13
supersedes: []
superseded_by: []
context_links:
  - "[[CLI Surface]]"
  - "[[Security Posture]]"
tags:
  - decision/accepted
  - cli/profile
  - security
  - compliance/soc2
aliases:
  - Keychain ADR
  - Credential Storage ADR
  - R3
related:
  - "[[Authentication]]"
  - "[[Security Posture]]"
  - "[[Compliance Controls]]"
---

# ADR-003: OS Keychain Credential Storage

## Status

Accepted

## Context

The CLI stores bearer tokens for named credential profiles. These tokens grant access to the Intelligence Cloud API and must be protected at rest. The project's compliance posture (SOC 2, HIPAA-adjacent) requires that secrets not be stored in plaintext configuration files when a more secure alternative is available.

The CLI runs on macOS, Linux, and Windows. Each platform has a native secret store (macOS Keychain, Windows Credential Manager, Linux Secret Service API via libsecret), but not all environments have one available (e.g. headless servers, containers).

## Decision

Use `github.com/zalando/go-keyring` for cross-platform OS keychain access. Tokens are stored with `service=icctl` and `account=<profile-name>`. When no keychain is reachable, fall back to a user-scoped file at `$XDG_CONFIG_HOME/icctl/credentials.yaml` with `0600` permissions.

Non-secret profile metadata (profile name, environment selector, base URL, default output format) is always stored in a plaintext configuration file and never contains credential material.

## Consequences

### Positive

- **Platform-native security**: tokens are protected by the OS keychain on supported platforms, which integrates with system-level access controls and (on macOS) biometric unlock.
- **Minimal dependency**: `go-keyring` is a single dependency with no cgo on macOS and Windows.
- **Graceful degradation**: the file fallback ensures the CLI works in headless environments.
- **Clear separation**: non-secret metadata is always plaintext; secrets are always in the keychain or a dedicated credentials file.

### Negative

- **Fallback is less secure**: the `0600` file fallback is weaker than a hardware-backed keychain. However, it matches the security posture of other Go CLIs (e.g. `gh`).
- **D-Bus dependency on Linux**: libsecret requires a running D-Bus session, which may not be available in minimal containers.

## Alternatives Considered

### 99designs/keyring

Supports more backends (encrypted file, `pass`, GPG), but carries a heavier dependency tree with several historically stale backends. The broader backend matrix is not justified given the "OS keychain or `0600` file" requirement.

### keybase/go-keychain

macOS-only. Does not meet the cross-platform requirement.

### Shell-out to platform tools

Calling `security` (macOS), `keyring` (Linux), or `cmdkey` (Windows) directly. Fragile on Windows and not fully scriptable.

## References

- [[Security Posture]]
- [[Compliance Controls]]
- [[CLI Surface]]
- [go-keyring on GitHub](https://github.com/zalando/go-keyring)
