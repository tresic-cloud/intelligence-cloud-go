---
title: Quickstart - CLI
type: guide
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - guide
  - cli
  - quickstart
aliases:
  - CLI Quickstart
  - icctl Quickstart
related:
  - "[[Quickstart - SDK]]"
  - "[[Overview]]"
  - "[[CLI Surface]]"
  - "[[02 Decisions/ADR-002 Cobra and Viper for CLI]]"
  - "[[02 Decisions/ADR-003 OS Keychain Credential Storage]]"
---

# Quickstart: CLI

This guide walks you through installing `icctl`, configuring a credential profile, and performing your first read and write operations from the terminal. Everything here works on macOS, Linux, and Windows.

## Prerequisites

- A bearer token for Intelligence Cloud (ask your platform team).
- Familiarity with the environment you are targeting (production, staging, or local dev).

## Install the binary

### Option A: GitHub release download

Download the latest release for your platform from the [releases page](https://github.com/tresic-cloud/intelligence-cloud-go/releases), extract the archive, and move `icctl` onto your `$PATH`:

```sh
# Example for macOS arm64 — adjust the URL for your platform
gh release download --repo tresic-cloud/intelligence-cloud-go --pattern 'icctl_*_darwin_arm64.tar.gz'
tar xzf icctl_*_darwin_arm64.tar.gz
sudo mv icctl /usr/local/bin/
```

### Option B: Homebrew (planned)

A Homebrew tap is on the [[06 Project/Roadmap|roadmap]]. Once available:

```sh
brew install tresic-cloud/tap/icctl
```

### Verify the installation

```sh
icctl version
# => icctl v0.2.0 (commit abc1234, go1.23.x, darwin/arm64)
```

All release binaries are signed with cosign keyless (GitHub OIDC). See [[Releasing]] for verification instructions and [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing|ADR-010]] for the signing rationale.

## Create a profile

Profiles store connection settings for a named environment. The token is piped via `--token-stdin` so it never appears in shell history:

```sh
icctl profile add staging --environment staging --token-stdin
# paste your token, press Ctrl-D (Ctrl-Z on Windows)
```

### Where credentials are stored

Non-secret profile metadata (name, environment, base URL, default output format) lives in `~/.config/icctl/config.yaml` (mode `0600`). The token itself is stored in your OS keychain -- macOS Keychain, Windows Credential Manager, or the Secret Service API (libsecret) on Linux.

If no keychain is reachable (headless servers, containers), the CLI falls back to `~/.config/icctl/credentials.yaml`, also with `0600` permissions. See [[02 Decisions/ADR-003 OS Keychain Credential Storage]] for the full design and the compliance rationale.

### Test the profile

```sh
icctl profile test staging
# => OK — authenticated as Jason Goecke <jason@tresic.cloud>
```

## Read operations

### Human-readable output (default)

```sh
icctl --profile staging resellers list
```

Output:

```
ID                NAME              STATUS   CREATED
rsl_01HW3...     Acme Bakery       active   2025-12-04
rsl_01HX7...     Blue Harbor Pizza active   2026-01-17
```

The table format is the default and is designed for terminal use. Column widths adapt to content.

### Machine-readable output (JSON)

For scripting and piping into tools like `jq`:

```sh
icctl --profile staging resellers list --output json | jq '.items[] | {id, name}'
```

```json
{"id":"rsl_01HW3...","name":"Acme Bakery"}
{"id":"rsl_01HX7...","name":"Blue Harbor Pizza"}
```

JSON output writes to stdout with no decoration. Errors always go to stderr. This means scripts can safely parse stdout without catching error messages. See [[01 Architecture/CLI Surface]] for the complete exit code reference.

## Destructive operations

Destructive verbs (`delete`, `deactivate`, and others) show a Pulumi-style preview before executing. This is a safety net against accidental damage in production:

```sh
icctl --profile staging resellers delete rsl_01HW3ABCDEF
```

The preview:

```
The following operation is about to be performed:

  Environment:    staging (https://api.staging.intelligence.cloud)
  Operation:      Delete reseller
  Resource type:  reseller
  Resource id:    rsl_01HW3ABCDEF
  Irreversible:   yes
  Attributes:
    name:         Acme Bakery, Inc.
    status:       active

Proceed? (y/N):
```

Type `y` to confirm. Non-TTY invocations (piped input, CI) refuse to execute unless explicitly bypassed -- no accidental destructive calls in automation.

### Scripted bypass

For CI pipelines and scripts, use `--yes` or set the environment variable:

```sh
# Flag bypass
icctl --profile staging resellers delete rsl_01HW3ABCDEF --yes

# Environment variable bypass (session-wide)
export ICCTL_ASSUME_YES=1
icctl --profile staging resellers delete rsl_01HW3ABCDEF
```

The preview still appears in `--output json` mode as a structured plan object, so automation can inspect it before confirming. See [[04 Operations/Security Posture]] for the broader security model around destructive operations.

## Multi-environment workflow

You will typically work with more than one environment. Profiles make switching seamless:

```sh
# Add a production profile
icctl profile add prod --environment production --token-stdin

# Set a default so you don't have to type --profile every time
icctl profile set-default prod

# Override per-invocation when you need staging
icctl --profile staging resellers list
```

Configuration precedence follows: **flag > environment variable > profile > built-in default**. This is the standard layering provided by [[02 Decisions/ADR-002 Cobra and Viper for CLI|Cobra + Viper]].

## Getting help

Every subcommand includes a detailed help message with flags, descriptions, and examples:

```sh
icctl help
icctl resellers --help
icctl resellers list --help
```

## Next steps

- **SDK quickstart**: if you want to call the API from Go code, see [[Quickstart - SDK]].
- **Full command reference**: see [[01 Architecture/CLI Surface]] for the complete command tree, flag index, and exit codes.
- **Contributing**: want to add a new subcommand? Start with [[Contributing]].
- **Release verification**: learn how to verify binary signatures in [[Releasing]].
- **Architecture**: understand how the CLI layers on top of the SDK in [[Overview]].
