---
title: Operation IDs
type: reference
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - reference
  - api
  - operations
aliases:
  - Operations
  - API Operations
  - operationId
related:
  - "[[01 Architecture/SDK Surface]]"
  - "[[01 Architecture/HTTP Transport]]"
  - "[[Error Hierarchy]]"
  - "[[Pagination]]"
  - "[[02 Decisions/ADR-001 OpenAPI Codegen with oapi-codegen]]"
---

# Operation IDs

This note lists all 21 OpenAPI operations in v0.1 of the Intelligence Cloud API. Each `operationId` maps directly to a typed Go method on the corresponding resource service in the SDK, and to an OTel span named `intelligencecloud.{operationId}` in the [[01 Architecture/HTTP Transport]] pipeline.

> **Source**: Generated from `testdata/openapi.yaml` pinned at commit `718879d0`. Regenerate this note when the pinned spec changes -- the canonical operation set lives in that file.

## Auth

| operationId | Method | Path | Description |
|---|---|---|---|
| `login` | `POST` | `/api/v1/auth/login` | Authenticate Tresic admin and obtain tokens via Azure AD ROPC flow |

## Me

User profile, avatar, and preference endpoints for the currently authenticated user.

| operationId | Method | Path | Description |
|---|---|---|---|
| `getMe` | `GET` | `/api/v1/me` | Get current user profile (resolves across admin, reseller, company user types) |
| `createAvatarUploadURL` | `POST` | `/api/v1/me/avatar/upload-url` | Generate a pre-signed Azure Blob SAS URL for avatar upload (5 MB max, 15 min expiry) |
| `confirmAvatar` | `PATCH` | `/api/v1/me/avatar` | Confirm a previously generated SAS upload and store the asset URL |
| `deleteAvatar` | `DELETE` | `/api/v1/me/avatar` | Remove profile photo from blob storage and reset avatar URL |
| `getNotificationPreferences` | `GET` | `/api/v1/me/notification-preferences` | Get notification preferences (lazily creates defaults if none exist) |
| `patchNotificationPreferences` | `PATCH` | `/api/v1/me/notification-preferences` | Partially update notification preferences |
| `getDailyRecapPreferences` | `GET` | `/api/v1/me/daily-recap-preferences` | Get daily recap preferences from ACC user settings |
| `patchDailyRecapPreferences` | `PATCH` | `/api/v1/me/daily-recap-preferences` | Partially update daily recap preferences |

## Resellers

Full CRUD, lifecycle management, and audit logging for reseller accounts. Requires Tresic Admin or Reseller Admin scopes.

| operationId | Method | Path | Description |
|---|---|---|---|
| `createReseller` | `POST` | `/api/v1/resellers` | Create a new global reseller account |
| `listResellers` | `GET` | `/api/v1/resellers` | List all resellers (paginated, with search/country/active filters) |
| `getReseller` | `GET` | `/api/v1/resellers/{id}` | Get full details of a single reseller by UUID |
| `updateReseller` | `PUT` | `/api/v1/resellers/{id}` | Full replacement update of all reseller fields |
| `patchReseller` | `PATCH` | `/api/v1/resellers/{id}` | Partial update of reseller fields |
| `deactivateReseller` | `POST` | `/api/v1/resellers/{id}/deactivate` | Deactivate a reseller (no deletion; records audit trail) |
| `reactivateReseller` | `POST` | `/api/v1/resellers/{id}/reactivate` | Reactivate a previously deactivated reseller |
| `listResellerAuditLogs` | `GET` | `/api/v1/resellers/{resellerId}/audit-logs` | List paginated audit logs for a reseller (filterable by action, date range, actor) |

## Connectors

Connector listings scoped to locations or companies.

| operationId | Method | Path | Description |
|---|---|---|---|
| `listLocationConnectors` | `GET` | `/api/v1/locations/{locationId}/connectors` | List connectors for a location (paginated, with search and status filter) |
| `listCompanyConnectors` | `GET` | `/api/v1/companies/{companyId}/connectors` | List connectors across all locations for a company (includes company-wide error summary) |

## Users

| operationId | Method | Path | Description |
|---|---|---|---|
| `patchCompanyUser` | `PATCH` | `/api/v1/companies/{companyId}/users/{userId}` | Partially update a company user (includes department field) |

## Products

| operationId | Method | Path | Description |
|---|---|---|---|
| `listProducts` | `GET` | `/api/v1/products` | List all available products for assignment to resellers |

## Operation Count Summary

| Resource area | Count |
|---|---|
| Auth | 1 |
| Me | 8 |
| Resellers | 8 |
| Connectors | 2 |
| Users | 1 |
| Products | 1 |
| **Total** | **21** |

## SDK Mapping

Each `operationId` maps to a method on the corresponding resource service field of `*Client`:

- `login` is available via `client.Auth.Login(ctx, req)`
- `getMe` is available via `client.Me.Get(ctx)`
- `listResellers` is available via `client.Resellers.List(ctx, opts...)` returning `*Iterator[Reseller]`
- `createReseller` is available via `client.Resellers.Create(ctx, req)`

For the complete typed method signatures, see [[01 Architecture/SDK Surface]].

## See Also

- [[01 Architecture/SDK Surface]] -- full public API contract with method signatures
- [[01 Architecture/HTTP Transport]] -- OTel span naming convention using operation IDs
- [[Pagination]] -- how `List*` operations return `Iterator[T]`
- [[Error Hierarchy]] -- `Operation()` method on all typed errors returns the `operationId`
- [[02 Decisions/ADR-001 OpenAPI Codegen with oapi-codegen]] -- how operations are generated from the spec
