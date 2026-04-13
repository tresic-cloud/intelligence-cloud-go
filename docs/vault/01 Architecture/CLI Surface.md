---
title: CLI Surface
type: architecture
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - architecture
  - cli
  - icctl
aliases:
  - CLI Schema
  - icctl Commands
  - Command Tree
related:
  - "[[Overview]]"
  - "[[SDK Surface]]"
  - "[[Quickstart - CLI]]"
  - "[[ADR-002 Cobra and Viper for CLI]]"
  - "[[ADR-003 OS Keychain Credential Storage]]"
---

# CLI Surface

The `icctl` binary exposes every SDK operation as a subcommand. It is built on [[ADR-002 Cobra and Viper for CLI|Cobra + Viper]] and follows the operator-facing stability contract: breaking changes to the command tree require a major version bump.

## Global Flags

These flags are available on every subcommand. Precedence: flag > environment variable > profile > built-in default.

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--profile <name>` | `ICCTL_PROFILE` | `default` | Select a named credential profile |
| `--environment <name>` | `ICCTL_ENVIRONMENT` | from profile | `production`, `staging`, `dev`, or custom |
| `--base-url <url>` | `ICCTL_BASE_URL` | from environment | Override environment URL mapping |
| `--token <t>` | `ICCTL_TOKEN` | from keychain/file | One-off token (does not update profile) |
| `--output <fmt>` | `ICCTL_OUTPUT` | `table` | `table` or `json` |
| `--yes` / `-y` | `ICCTL_ASSUME_YES` | unset | Bypass destructive-verb confirmation |
| `--log-level <lvl>` | `ICCTL_LOG_LEVEL` | `warn` | `debug`, `info`, `warn`, `error` |
| `--request-id <id>` | `ICCTL_REQUEST_ID` | auto-generated | Force `X-Request-Id` header |
| `--timeout <dur>` | `ICCTL_TIMEOUT` | `30s` | Per-call timeout |

## Command Tree

```
icctl
  version                           Print version, commit SHA, Go runtime
  help [command]                    Help for any command

  profile
    list                            List configured profiles
    show [name]                     Show profile details (no secrets)
    add <name> [flags]              Create a new profile
    set-default <name>              Switch the active profile
    set-token <name> [--token-stdin] Store a token in the keychain
    remove <name>                   Delete profile + keychain entry
    test <name>                     Verify credentials with a ping call

  me
    get                             Get current user
    update-notification-prefs       Update notification preferences
    upload-avatar <file>            Upload avatar image

  resellers
    list [--page-size N] [--max-items N] [--filter key=value]
    get <id>
    create [--from-file <json>] [flag fields...]
    update <id> [--from-file <json>] [flag fields...]
    delete <id>                     DESTRUCTIVE
    deactivate <id>                 DESTRUCTIVE
    reactivate <id>
    audit-logs <id> [--since] [--until]

  companies     { list, get, create, update, delete, connectors }
  locations     { list, get, create, update, delete, connectors }
  products      { list, get, create, update, delete }
  verticals     { list, get, create, update, delete }
  connectors    { list, get, create, update, delete }
  users         { list, get, create, update, delete }
  audit-logs    { search }
  completion    { bash, zsh, fish, powershell }
```

Every leaf subcommand maps 1:1 to an SDK method. As the canonical OpenAPI grows, new subcommands appear automatically.

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic failure (API error, validation, user cancelled) |
| `2` | Usage error (missing flag, unknown subcommand, non-TTY refused destructive verb) |
| `3` | Configuration error (no profile, invalid config, keychain unreachable) |
| `4` | Authentication error (token expired or rejected) |
| `5` | Authorization error (403) |
| `6` | Rate-limit error (429 after retries exhausted) |
| `7` | Server error (5xx after retries exhausted) |
| `124` | Timeout (per-call `--timeout` elapsed) |
| `130` | Interrupted (SIGINT) |

Errors are always written to `stderr`. Standard output remains parseable (or empty) on failure.

## Output Formats

### Table (default on TTY)

Human-readable columns, opinionated per resource. Boolean values render as checkmarks with ASCII fallback on non-UTF-8 terminals. Timestamps render in local timezone; `--utc` switches to UTC.

### JSON

Machine-readable. Single JSON document per response. List/search verbs wrap results:

```json
{
  "items": [...],
  "page_info": {
    "items_fetched": 123,
    "pages_fetched": 3,
    "has_next_page": false,
    "next_page_token": ""
  }
}
```

Errors under `--output json` are written to `stderr` as structured JSON with `kind`, `status`, `code`, `message`, `request_id`, and `operation` fields.

## Destructive Verb Preview

Destructive verbs (`delete`, `deactivate`, and any verb the backend documents as destructive) display a Pulumi-style preview before issuing the API call:

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

Under `--output json`, the preview is emitted as a structured plan object on stdout before the prompt. Non-TTY invocations without `--yes` or `ICCTL_ASSUME_YES=1` refuse with exit code 2. See [[Security Posture]] for the safety rationale.

## Configuration Files

Profiles are stored in two locations:

- **Non-secret metadata**: `$XDG_CONFIG_HOME/icctl/config.yaml` (mode `0600`)
- **Secrets**: OS keychain (service `icctl`, account `<profile-name>`) with fallback to `$XDG_CONFIG_HOME/icctl/credentials.yaml` (mode `0600`)

No plaintext config file ever contains a token. See [[ADR-003 OS Keychain Credential Storage]] and [[Authentication]].

## Help Contract

Every subcommand's `--help` output contains: one-line synopsis, usage line, description, flags table, and at least one examples block. This is enforced by a CLI-level integration test.
