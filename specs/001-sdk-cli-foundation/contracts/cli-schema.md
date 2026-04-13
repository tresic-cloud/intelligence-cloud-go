# Contract: `icctl` Command Schema

**Binary**: `icctl`
**Source**: `cmd/icctl/`
**Stability**: operator-facing surface. Breaking changes require a major-version bump.

## Global flags

Available on every subcommand:

| Flag | Env | Default | Notes |
|---|---|---|---|
| `--profile <name>` | `ICCTL_PROFILE` | `default` | Selects the Profile |
| `--environment <name>` | `ICCTL_ENVIRONMENT` | profile's env | `production` / `staging` / `dev` / custom. Resolves `--base-url`. |
| `--base-url <url>` | `ICCTL_BASE_URL` | resolved from env | Overrides environment mapping |
| `--token <t>` | `ICCTL_TOKEN` | from keychain/file | One-off token; does not update the profile |
| `--output <fmt>` | `ICCTL_OUTPUT` | `table` | `table` \| `json` |
| `--yes`, `-y` | `ICCTL_ASSUME_YES` | unset | Bypass destructive-verb confirmation (FR-015a) |
| `--log-level <lvl>` | `ICCTL_LOG_LEVEL` | `warn` | `debug` \| `info` \| `warn` \| `error` |
| `--request-id <id>` | `ICCTL_REQUEST_ID` | auto-generated | Forces the `X-Request-Id` header — useful for support triage |
| `--timeout <dur>` | `ICCTL_TIMEOUT` | `30s` | Per-call timeout |

**Precedence**: flag > environment variable > profile > built-in default.

## Top-level commands

```
icctl
├── version                         # print semver + commit SHA + go runtime version
├── help [command]
├── profile
│   ├── list
│   ├── show [name]
│   ├── add <name> [--environment …] [--base-url …] [--token-stdin]
│   ├── set-default <name>
│   ├── set-token <name> [--token-stdin]   # writes to keychain; file fallback if unreachable
│   ├── remove <name>
│   └── test <name>                         # issues a ping-equivalent call to confirm credentials
├── me
│   ├── get
│   ├── update-notification-prefs [--from-file <json>] [flag fields…]
│   └── upload-avatar <file>
├── resellers
│   ├── list [--page-size N] [--max-items N] [--filter key=value]…
│   ├── get <id>
│   ├── create [--from-file <json>] [flag fields…]
│   ├── update <id> [--from-file <json>] [flag fields…]
│   ├── delete <id>                         # DESTRUCTIVE
│   ├── deactivate <id>                     # DESTRUCTIVE
│   ├── reactivate <id>
│   └── audit-logs <id> [--since <ts>] [--until <ts>]
├── companies     { list, get, create, update, delete, connectors }
├── locations     { list, get, create, update, delete, connectors }
├── products      { list, get, create, update, delete }
├── verticals     { list, get, create, update, delete }
├── connectors    { list, get, create, update, delete }
├── users         { list, get, create, update, delete }
├── audit-logs    { search }
└── completion    { bash | zsh | fish | powershell }
```

Every leaf subcommand mirrors an SDK method 1:1 (FR-010). Subcommand set grows automatically as the canonical OpenAPI grows (SC-004).

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic failure — API error, validation error, or user cancelled confirmation |
| `2` | Usage error — missing flag, unknown subcommand, non-TTY refused destructive verb |
| `3` | Configuration error — no profile, invalid config file, keychain unreachable and no fallback |
| `4` | Authentication error — token expired or rejected |
| `5` | Authorization error — 403 from backend |
| `6` | Rate-limit error — 429 after retries exhausted |
| `7` | Server error — 5xx after retries exhausted |
| `124` | Timeout — per-call `--timeout` elapsed |
| `130` | Interrupted (SIGINT) |

## Output formats

### 1. `--output table` (default for TTY)

Human-readable. Columns are opinionated per resource (documented in each subcommand's `--help`). Boolean values render as `✓` / `✗` (ASCII fallback when not a UTF-8 terminal). Timestamps render in the local timezone; `--utc` switches to UTC.

Table output is written to `stdout`.

### 2. `--output json`

Machine-readable. Every response is a single JSON document (not NDJSON) except for `list`/`search` verbs, which emit a JSON object:

```json
{
  "items": [ /* … */ ],
  "page_info": { "items_fetched": 123, "pages_fetched": 3, "has_next_page": false, "next_page_token": "" }
}
```

Errors always go to `stderr` as a JSON document under `--output json`, never to stdout:

```json
{
  "error": {
    "kind": "authentication" | "authorization" | "validation" | "not_found" | "conflict" | "rate_limit" | "server" | "configuration" | "unexpected",
    "status": 401,
    "code": "token_expired",
    "message": "Bearer token expired",
    "request_id": "req_01HW…",
    "operation": "GetMe"
  }
}
```

### 3. Destructive-verb preview (FR-015a)

**TTY / `--output table`** — rendered to `stderr`:

```
The following operation is about to be performed:

  Environment:    production (https://api.intelligence.cloud)
  Operation:      Delete reseller
  Resource type:  reseller
  Resource id:    rsl_01HW3XYZ...
  Irreversible:   yes
  Attributes:
    name:         Acme Bakery, Inc.
    status:       active

Proceed? (y/N):
```

**`--output json`** — emitted to `stdout` as a structured plan BEFORE the prompt (scriptable):

```json
{
  "plan": {
    "environment": {"name": "production", "base_url": "https://api.intelligence.cloud"},
    "operation": "Delete reseller",
    "resource_type": "reseller",
    "resource_id": "rsl_01HW3XYZ...",
    "irreversible": true,
    "attributes": {"name": "Acme Bakery, Inc.", "status": "active"}
  }
}
```

After the plan, the confirmation prompt follows on `stderr` (TTY) or the command refuses with exit 2 (non-TTY without `--yes`).

## Help contract (FR-015)

Every subcommand's `--help` output contains, in order:
1. One-line synopsis
2. Usage line
3. Description (≥ 1 sentence)
4. Flags table (name, type, default, description)
5. At least one `Examples:` block

Enforced by a CLI-level integration test that walks the command tree.

## Configuration files

**`$XDG_CONFIG_HOME/icctl/config.yaml`** (`0600`):

```yaml
current_profile: staging
profiles:
  default:
    environment: production
    base_url: https://api.intelligence.cloud
    default_output: table
  staging:
    environment: staging
    base_url: https://api.staging.intelligence.cloud
    default_output: json
    tracing_endpoint: https://otlp.internal.example.com
```

**`$XDG_CONFIG_HOME/icctl/credentials.yaml`** (fallback, `0600`, only when keychain unavailable):

```yaml
staging:
  access_token: "eyJ…"
  expires_at: 2026-04-13T20:00:00Z
```

**Keychain entries** (preferred):
- service: `icctl`
- account: `<profile-name>`
- password: JSON-encoded `{"access_token": "…", "expires_at": "…"}`

No plaintext profile config ever contains a token.
