---
title: Vault README
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - meta
  - vault
aliases:
  - About This Vault
related:
  - "[[Home]]"
---

# Intelligence Cloud Go -- Documentation Vault

This is an [Obsidian](https://obsidian.md) vault containing all human-facing documentation for the `intelligence-cloud-go` project (the Go SDK and `icctl` CLI for the Intelligence Cloud platform).

## How to Open

1. Install [Obsidian](https://obsidian.md) (free for personal use).
2. Choose **Open folder as vault**.
3. Point to `docs/vault/` inside your clone of `intelligence-cloud-go`.

Obsidian will read the `.obsidian/` configuration automatically. No plugins need to be installed -- the vault uses only core features.

## Folder Structure

| Folder | Contents |
|---|---|
| `01 Architecture/` | System overview, public API surface, CLI command tree, transport, telemetry |
| `02 Decisions/` | Architectural Decision Records (ADR-001 through ADR-011) |
| `03 Guides/` | SDK and CLI quickstarts, contributing guide, release guide |
| `04 Operations/` | Compliance controls, security posture, observability runbook |
| `05 Reference/` | Glossary, error hierarchy, retry policy, pagination, auth, operation IDs |
| `06 Project/` | Roadmap, implementation status, constitution compliance, linked issues |
| `_attachments/` | Images and other binary assets (empty initially) |
| `_templates/` | Obsidian templates for new ADRs and reference notes |
| `.obsidian/` | Minimal vault configuration |

Numeric prefixes on folders ensure stable sort order in the Obsidian file explorer.

## Conventions

### Frontmatter

Every Markdown file begins with YAML frontmatter containing at minimum: `title`, `type`, `status`, `created`, `updated`, `tags`, `aliases`, and `related`. ADRs carry additional fields (`adr_id`, `adr_status`, `decision_date`, etc.).

### Internal Links

All cross-references between vault notes use Obsidian **wikilinks**: `[[Note Name]]` or `[[Note Name|display text]]`. Standard Markdown links (`[text](url)`) are reserved for external URLs.

### Tags

Tags use a nested hierarchy for searchability:

- `#sdk/auth`, `#sdk/transport`, `#sdk/errors`, `#sdk/pagination`
- `#cli/profile`, `#cli/output`, `#cli/destructive`
- `#decision/accepted`, `#decision/proposed`
- `#compliance/soc2`, `#compliance/hipaa`

Flat tags like `#go`, `#opentelemetry`, `#cobra` are also used where appropriate.

### Maps of Content (MOC)

Each top-level folder has a `_MOC.md` that indexes and categorises every note in that folder. The vault root has `Home.md` as the master MOC linking to all folder MOCs.

### Templates

Use the templates in `_templates/` when adding new notes:

- **ADR.md** -- full ADR skeleton with frontmatter
- **Reference.md** -- reference-note skeleton

In Obsidian, use the Templates core plugin (already enabled) to insert these.

## Relationship to speckit

The `specs/` directory contains machine-managed speckit artefacts (feature specs, implementation plans, task breakdowns). This vault **derives** from those artefacts but does not duplicate them. The vault is optimised for a developer or operator who needs to understand and use the project, not for the specification process itself.

Do not edit files in `specs/`, `.specify/`, or `CLAUDE.md` from this vault. Those are maintained separately.

## Contributing New Notes

1. Create a new note in the appropriate folder.
2. Use the matching template from `_templates/`.
3. Fill in frontmatter completely.
4. Add wikilinks to related notes and update the folder's `_MOC.md`.
5. If adding a new ADR, assign the next sequential number and update `[[02 Decisions/_MOC]]`.
