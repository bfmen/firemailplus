# FireMailPlus OpenAPI-First Migration Research

Date: 2026-04-30

Scope: current Go/Gin backend, Next.js frontend, and `backend/BACKEND_ISSUES_TODO_AUDIT.md`.

## Executive Decision

FireMailPlus should move to an OpenAPI-first workflow, but it should not start by generating code from the current handler surface wholesale.

The current repository already has drift between:

- registered routes in `backend/cmd/firemail/main.go`;
- extra handlers that are implemented but not registered;
- frontend calls in `frontend/src/lib/api.ts`;
- handwritten frontend types in `frontend/src/types/api.ts`;
- GORM models and SQL migrations.

The correct migration is therefore:

1. Create a curated OpenAPI contract for the routes that are actually registered and product-supported.
2. Mark known unstable or intentionally hidden surfaces as internal, `501`, or backlog, rather than publishing them as public SDK methods.
3. Generate frontend SDK and backend server interfaces from that contract.
4. Adapt existing service implementations behind generated server interfaces.
5. Fail CI when the OpenAPI bundle, generated SDK, generated server types, and route inventory drift.

## Recommended Toolchain

### OpenAPI Version

Use OpenAPI `3.0.3` for the first implementation phase.

Reasoning:

- It is supported broadly by Go generators and TypeScript SDK generators.
- It avoids early incompatibilities around OpenAPI 3.1 JSON Schema semantics.
- The project can upgrade to OpenAPI 3.1 later after generation, lint, and validation tooling are stable.

### Backend Generator

Use `oapi-codegen` for Go server types and Gin-compatible route/server scaffolding.

Recommended generated output:

- `backend/internal/api/generated/firemail.gen.go`
- `backend/internal/api/generated/models.gen.go` if models are split later
- `backend/internal/api/server.go` as handwritten adapter code

Recommended rule:

- Generated files must never contain business logic.
- Existing services remain the implementation source.
- Handwritten adapter handlers translate generated request/response types to existing services.

### Frontend SDK Generator

Use `orval` for TypeScript client generation.

Recommended generated output:

- `frontend/src/api/generated/firemail.ts`
- `frontend/src/api/generated/firemail.schemas.ts`
- optionally React Query hooks after the low-level SDK is stable

Recommended rule:

- `frontend/src/lib/api.ts` should become a compatibility facade over the generated SDK during migration.
- New frontend code should import generated operation functions/types directly.
- Handwritten DTOs in `frontend/src/types/api.ts` should be deleted or reduced to UI-only view models once generated schemas cover the backend contract.

## Why Not Generate Everything Immediately

The audit file identifies several routes and services that should not be exposed as stable contract yet:

- `EmailSendHandler.RegisterRoutes` defines bulk send, status, resend, draft, and template APIs, but the main router does not register it.
- `StandardEmailSender.GetSendStatus` and resend persistence are not implemented.
- `sent_emails`, `email_drafts`, `email_quotas`, and `email_templates` model/table definitions are inconsistent with migrations.
- `DeduplicationHandler` is not registered and its account access validation currently returns nil.
- Some frontend calls target unregistered routes, including `/emails/stats` and `/emails/draft`.

If these are exported into OpenAPI and used for SDK generation now, the generated SDK will make broken features look supported. The contract should instead encode only stable behavior, and the migration backlog should explicitly list blocked endpoints.

## Contract Boundary For Phase 1

Publish these route groups first because they are registered in `setupRoutes` and match current product flows:

- `GET /health`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `GET /api/v1/oauth/gmail`
- `GET /api/v1/oauth/outlook`
- `GET /api/v1/oauth/{provider}/callback`
- `POST /api/v1/oauth/create-account`
- `POST /api/v1/oauth/manual-config`
- `GET /api/v1/accounts`
- `POST /api/v1/accounts`
- `POST /api/v1/accounts/custom`
- `GET /api/v1/accounts/{id}`
- `PUT /api/v1/accounts/{id}`
- `DELETE /api/v1/accounts/{id}`
- `POST /api/v1/accounts/{id}/test`
- `POST /api/v1/accounts/{id}/sync`
- `PUT /api/v1/accounts/{id}/mark-read`
- `POST /api/v1/accounts/batch/delete`
- `POST /api/v1/accounts/batch/sync`
- `POST /api/v1/accounts/batch/mark-read`
- `GET /api/v1/providers`
- `GET /api/v1/providers/detect`
- `GET /api/v1/emails`
- `GET /api/v1/emails/search`
- `GET /api/v1/emails/{id}`
- `PATCH /api/v1/emails/{id}`
- `POST /api/v1/emails/send`
- `DELETE /api/v1/emails/{id}`
- `PUT /api/v1/emails/{id}/read`
- `PUT /api/v1/emails/{id}/unread`
- `PUT /api/v1/emails/{id}/star`
- `PUT /api/v1/emails/{id}/move`
- `PUT /api/v1/emails/{id}/archive`
- `POST /api/v1/emails/{id}/reply`
- `POST /api/v1/emails/{id}/reply-all`
- `POST /api/v1/emails/{id}/forward`
- `POST /api/v1/emails/batch`
- `GET /api/v1/folders`
- `POST /api/v1/folders`
- `GET /api/v1/folders/{id}`
- `PUT /api/v1/folders/{id}`
- `DELETE /api/v1/folders/{id}`
- `PUT /api/v1/folders/{id}/mark-read`
- `PUT /api/v1/folders/{id}/sync`
- `GET /api/v1/groups`
- `POST /api/v1/groups`
- `PUT /api/v1/groups/reorder`
- `PUT /api/v1/groups/{id}`
- `PUT /api/v1/groups/{id}/default`
- `DELETE /api/v1/groups/{id}`
- attachment routes registered by `AttachmentHandler.RegisterRoutes`
- `GET /api/v1/sse`
- `GET /api/v1/sse/events`
- `GET /api/v1/sse/stats`
- `POST /api/v1/sse/test`

Do not publish these as stable Phase 1 contract until the audit blockers are fixed:

- `/api/v1/emails/send/bulk`
- `/api/v1/emails/send/{send_id}/status`
- `/api/v1/emails/send/{send_id}/resend`
- `/api/v1/emails/draft*`
- `/api/v1/emails/templates*`
- `/api/v1/deduplication*`
- `/api/v1/emails/stats`, unless a real backend handler is registered.

## Suggested Repository Layout

```text
openapi/
  firemail.yaml
  components/
    common.yaml
    auth.yaml
    accounts.yaml
    emails.yaml
    folders.yaml
    groups.yaml
    attachments.yaml
    sse.yaml

backend/
  internal/api/generated/
    firemail.gen.go
  internal/api/
    server.go
    responses.go
  tools/
    tools.go

frontend/
  orval.config.ts
  src/api/generated/
    firemail.ts
    firemail.schemas.ts
```

Start with a single `openapi/firemail.yaml` if split files slow the first migration. Split it only after generation is green.

## Generation Commands

Backend:

```bash
cd /root/Coding/General/firemailplus/backend
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
  -config ../openapi/oapi-codegen.yaml \
  ../openapi/firemail.yaml
```

Frontend:

```bash
cd /root/Coding/General/firemailplus/frontend
pnpm add -D orval
pnpm exec orval --config orval.config.ts
```

Recommended scripts after setup:

```json
{
  "scripts": {
    "generate:api": "orval --config orval.config.ts",
    "check:api": "orval --config orval.config.ts --dry-run"
  }
}
```

Recommended backend Make targets:

```makefile
.PHONY: generate-api
generate-api:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -config ../openapi/oapi-codegen.yaml ../openapi/firemail.yaml

.PHONY: check-api
check-api: generate-api
	git diff --exit-code -- internal/api/generated ../frontend/src/api/generated
```

## Backend Integration Pattern

Use generated server interfaces as the route boundary and keep existing services as implementation:

```text
HTTP request
  -> generated request binding/types
  -> handwritten adapter
  -> existing service method
  -> generated response type
  -> Gin response
```

Do not put database access, provider calls, token refresh, SSE publishing, or migration logic into generated files.

Recommended adapter behavior:

- Convert all IDs at boundary as unsigned integers only once.
- Return typed error responses with stable `code`, `message`, and optional `details`.
- Keep `SuccessResponse<T>` only if it remains a deliberate public envelope. Otherwise migrate endpoint-by-endpoint to direct resources.
- For long-running sync/send operations, return `202 Accepted` plus a job/status resource, not a misleading immediate success.

## Frontend Migration Pattern

Do not rewrite the entire frontend at once.

Recommended steps:

1. Generate low-level SDK from OpenAPI.
2. Create a generated-client wrapper that injects bearer token and handles `401` cleanup.
3. Reimplement `frontend/src/lib/api.ts` methods by delegating to generated functions.
4. Migrate hooks and components gradually from the facade to generated operation functions.
5. Delete duplicate handwritten types only after all call sites use generated schemas.

This avoids a risky big-bang rewrite while still making OpenAPI the source of truth.

## Response And Error Contract

The current backend mixes response shapes such as `SuccessResponse`, `ErrorResponse`, handler-specific JSON objects, and file/blob responses. OpenAPI migration should normalize these.

Recommended standard:

```yaml
ApiError:
  type: object
  required: [error, message]
  properties:
    error:
      type: string
    message:
      type: string
    code:
      type: string
    details:
      type: object
      additionalProperties: true

ApiResponse:
  type: object
  required: [success]
  properties:
    success:
      type: boolean
    message:
      type: string
    data: {}
```

Use stable error codes for frontend logic. Do not make the UI parse localized or human-readable messages.

## Security Contract

Define one bearer scheme:

```yaml
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

Security rules:

- Never log `Authorization` or token prefixes.
- Do not use query token authentication except for SSE if browser `EventSource` constraints require it.
- If SSE keeps `?token=`, document it as a constrained compatibility mechanism and scrub query strings from logs.
- Require every generated authenticated operation to include `security: [{ bearerAuth: [] }]`.

## SSE Contract

OpenAPI can document the SSE endpoint as `text/event-stream`, but it cannot fully type every runtime event without a disciplined event schema.

Recommended approach:

- Define `SseEventEnvelope` with `type`, `timestamp`, `user_id`, and `data`.
- Define known event payload schemas for mailbox, group, account, and email read-state events.
- Keep EventSource client code handwritten, but import generated event payload types.
- Add an SSE schema test that ensures backend event structs and OpenAPI schemas stay aligned.

## Attachment Contract

Attachment upload/download should be explicit in OpenAPI:

- Upload uses `multipart/form-data`.
- Download returns binary `application/octet-stream` or the stored MIME type.
- Attachment metadata returns JSON.

Do not force attachment binary traffic through the generic `ApiResponse<T>` envelope.

## CI Gate

Add a CI job with these checks:

1. Lint/bundle OpenAPI.
2. Generate backend server/types.
3. Generate frontend SDK.
4. Run `git diff --exit-code` against generated directories.
5. Run backend tests.
6. Run frontend type-check.

Minimum local gate:

```bash
cd /root/Coding/General/firemailplus/backend && go test ./...
cd /root/Coding/General/firemailplus/frontend && pnpm type-check
```

After generation is introduced:

```bash
cd /root/Coding/General/firemailplus && make generate-api
git diff --exit-code
```

## Migration Phases

### Phase 0: Contract Inventory

Deliverables:

- `openapi/firemail.yaml`
- route inventory generated from `setupRoutes`
- frontend API usage inventory
- blocked endpoint list linked to `BACKEND_ISSUES_TODO_AUDIT.md`

Exit criteria:

- OpenAPI only describes registered, product-supported routes.
- Every current frontend API call is classified as supported, drifted, or deprecated.

### Phase 1: SDK Generation Without Backend Router Replacement

Deliverables:

- `frontend/orval.config.ts`
- generated SDK under `frontend/src/api/generated`
- `frontend/src/lib/api.ts` delegates to generated functions for 3-5 stable route groups.

Exit criteria:

- TypeScript type-check passes.
- No duplicate DTO edits are required for migrated routes.

### Phase 2: Backend Generated Interface

Deliverables:

- `backend/internal/api/generated`
- handwritten adapter implementing generated server interface
- generated route registration isolated behind a feature branch or parallel router

Exit criteria:

- Existing backend tests pass.
- Generated server routes match the OpenAPI operation IDs.
- No business logic lives in generated files.

### Phase 3: Fix Blocked Backend Surfaces

Deliverables:

- remove token-prefix logging;
- decide strong-consistency delete semantics;
- fix model/migration drift for send, drafts, templates, quota;
- either fully register and implement send/draft/template APIs or remove them from public scope;
- implement deduplication account authorization before registering deduplication APIs.

Exit criteria:

- The next OpenAPI version can safely expose previously blocked operations.

### Phase 4: Contract Enforcement

Deliverables:

- CI generation-diff gate;
- generated SDK becomes the default frontend API source;
- route drift tests compare Gin route list with OpenAPI paths and methods.

Exit criteria:

- A backend route cannot be added without either documenting it or explicitly marking it internal.
- A frontend API call cannot target an undocumented route.

## Concrete Next Step

The next implementation commit should be small and mechanical:

1. Add `openapi/firemail.yaml` for Phase 1 supported routes.
2. Add `frontend/orval.config.ts`.
3. Add `openapi/oapi-codegen.yaml`.
4. Generate SDK and backend types.
5. Add generation commands to `frontend/package.json` and `backend/Makefile`.
6. Run backend tests and frontend type-check.

Do not register `EmailSendHandler` or `DeduplicationHandler` as part of that commit. Those are separate remediation tasks because the audit already shows correctness and authorization blockers.

## Primary References

- OpenAPI Specification: https://spec.openapis.org/oas/latest.html
- oapi-codegen project documentation: https://github.com/oapi-codegen/oapi-codegen
- Orval documentation: https://orval.dev
