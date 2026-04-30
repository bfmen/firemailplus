# FireMailPlus OpenAPI Migration Tasks

Last updated: 2026-04-30

This is the single canonical execution file for the FireMailPlus OpenAPI migration and related backend issue remediation. Do not create parallel task plans for this effort. Update this file before and after every task attempt.

## Global Rules

- Source of truth: this file is the only task tracker for the migration.
- Contract first: new or changed public API surfaces must be modeled in OpenAPI before backend adapters or frontend SDK usage are changed.
- Compatibility first: keep `/api/v1` and the existing `SuccessResponse<T>` / `ErrorResponse` wire envelope unless a task explicitly records and verifies an exception.
- Generator ownership: generated Go code belongs under `backend/internal/api/generated`; generated TypeScript SDK code belongs under `frontend/src/api/generated`; no business logic may be written into generated files.
- Stable routes only: do not publish unregistered, dead, fake-success, or schema-blocked endpoints as stable OpenAPI paths until their blockers are fixed.
- Migrations truth: versioned SQL migrations are the database truth source; do not hide model/schema drift with `AutoMigrate`.
- SSE contract: EventSource remains supported; OpenAPI must describe `text/event-stream` and event payload schemas; query-token compatibility is allowed only with log redaction.
- Per-task workflow: read this file, mark the current task `in_progress`, inspect the task's listed code, record findings, implement changes, self-review the diff, run acceptance commands, record results, then mark `done` or `blocked`.
- Failure rule: any failed acceptance command must be recorded under `Errors Encountered` with the next different strategy before another task begins.
- Strict full gate: each task must run `cd backend && go test ./...`, `cd frontend && pnpm type-check`, OpenAPI lint, backend codegen, frontend SDK codegen, and generated diff checks once those commands exist. Before tooling exists, record those checks as not yet available and do not claim OpenAPI/codegen validation.

## Current Baseline

- Repository: `/root/Coding/General/firemailplus`
- Branch at T00 start: `main`
- Local state at T00 start: branch was ahead of `origin/main` by 2 commits; `backend/BACKEND_ISSUES_TODO_AUDIT.md` and `docs/openapi-first-migration.md` were untracked input documents.
- Backend stack: Go/Gin, module `firemail`, Go `1.23.0`, toolchain `go1.24.3`.
- Frontend stack: Next.js/React, TypeScript, `pnpm type-check`; API client is handwritten in `frontend/src/lib/api.ts`.
- API status: stable backend routes are registered manually in `backend/cmd/firemail/main.go`; attachments are registered via `AttachmentHandler.RegisterRoutes(api)`.
- OpenAPI status: no canonical `openapi/firemail.yaml`, Redocly config, oapi-codegen config, Orval config, or generated SDK/server artifacts existed at T00 start.
- Planned tool versions: OpenAPI `3.0.3`, `oapi-codegen v2.6.0`, `orval 8.9.0`, `@redocly/cli 2.30.3`.
- Input documents: `backend/BACKEND_ISSUES_TODO_AUDIT.md` and `docs/openapi-first-migration.md`.
- T00 verification results: `cd backend && go test ./...` passed; `cd frontend && pnpm type-check` passed.

## Task Index

| ID | Status | Goal | Exit Gate |
| --- | --- | --- | --- |
| T00 | done | Establish this task file and freeze the executable baseline. | Task file complete; backend tests and frontend type-check rerun and recorded. |
| T01 | done | Inventory registered Gin routes, frontend API calls, unregistered handlers, and model/migration drift. | Every API is classified as `stable`, `drifted`, `blocked`, or `to-fix-and-expose`. |
| T02 | done | Land OpenAPI tooling: spec, Redocly config, oapi-codegen config, Orval config, and generation commands. | OpenAPI lint passes; backend and frontend generated artifacts are reproducible and contain no business logic. |
| T03 | done | Build the Phase 1 OpenAPI contract for currently registered stable auth, oauth, accounts, providers, emails, folders, groups, attachments, SSE, and health routes. | OpenAPI paths match real stable routes; unstable routes are not falsely marked stable. |
| T04 | done | Generate the TypeScript SDK and make `frontend/src/lib/api.ts` delegate gradually while preserving caller compatibility. | Frontend type-check passes; migrated facade methods have SDK mapping coverage. |
| T05 | done | Generate Go server/types and add handwritten adapters that call existing services. | Backend tests pass; generated files contain no business logic; route snapshot and OpenAPI do not drift. |
| T06 | done | Fix security and misleading behavior around token logging, weak production admin defaults, dirty migration force, and SSE query-token logs. | Config/log/migration tests pass; backend full tests pass. |
| T07 | done | Repair send/draft/template/quota schema using versioned migrations and unify draft semantics on `drafts` / `models.Draft`. | Empty DB migration and key CRUD tests pass; GORM models match SQL schema. |
| T08 | done | Complete and expose extended send capability: bulk, status, resend, drafts, templates, send events. | OpenAPI exposes completed routes; race and persistence tests pass. |
| T09 | done | Complete scheduled send and template composition integration. | Scheduler state-machine, template processing, and retry tests pass. |
| T10 | done | Repair mail consistency for delete, move/archive token refresh, and nested folder sync. | Remote-failure, OAuth refresh, and nested folder tests pass. |
| T11 | done | Complete and expose deduplication capability with account/user permissions and schedule semantics. | Cross-user denial and schedule/cancel/report tests pass. |
| T12 | done | Fix attachment preview/capability behavior, compression/checksum config, SMTP capability truth, and RFC encoding. | Attachment, capability, and Chinese header/filename tests pass. |
| T13 | done | Resolve dead management routes: auth refresh/change-password/profile, backup, soft-delete. | Admin permission tests pass; non-admin access is denied; OpenAPI covers complete routes. |
| T14 | done | Complete cache/search/P3 maintainability fixes. | Side-effect, cache isolation, reply subject, and HTML policy tests pass. |
| T15 | done | Final migration cleanup and full strict validation. | Backend tests, frontend type-check, OpenAPI lint/codegen diff, route drift, and SDK drift all pass. |
| T16 | done | Land E2E investigation baseline and append the E2E remediation task chain. | Investigation doc is tracked; task file consistency is fixed; full baseline gates pass; commit created. |
| T17 | done | Fix auth refresh so every valid token can be rolled forward. | Fresh token refresh returns success; invalid/expired tokens still fail; OpenAPI/generated artifacts and tests pass. |
| T18 | pending | Fix email-group default semantics so user default groups can be renamed safely. | First custom group can be updated; system groups remain protected; default delete protection remains tested. |
| T19 | pending | Convert batch account mark-read to an asynchronous, observable job. | API returns accepted job data quickly; job status/SSE progress are test-covered; no 60s request timeout. |
| T20 | pending | Stabilize single-email read-state remote sync errors and frontend rewrite behavior. | Remote/provider failures return typed errors, not opaque 500; direct backend and frontend rewrite behavior match. |
| T21 | pending | Fix dedup stats/report fallback and schedule defaults/validation. | Stats/report no longer 500 without enhanced dedup; empty schedule uses defaults; invalid schedule returns 400. |
| T22 | pending | Make admin soft-delete cleanup usable with an empty body. | Empty body uses default retention days; OpenAPI request body is optional; focused tests pass. |
| T23 | pending | Harden SSE heartbeat/reconnect behavior and redact frontend token logs. | 120s browser smoke receives heartbeat; console/HAR/log scans contain no token/JWT leakage. |
| T24 | pending | Fix search page folder loading and query/empty-state behavior. | HAR has no folder request without `account_id`; search URL and empty state reflect current query. |
| T25 | pending | Improve Docker build resilience around external base images. | Build supports mirror/base-image overrides and retry; Docker or documented fallback validation passes. |
| T26 | pending | Add reproducible local production E2E harness and reporting. | Backend curl and frontend jshook flows produce redacted artifacts under `/tmp/firemailplus-e2e-artifacts`. |
| T27 | pending | Rebuild, deploy a clean test instance, import the two Outlook accounts, and rerun full E2E. | Backend curl and frontend jshook pass with no unexpected 4xx/5xx/timeout/leaks. |
| T28 | pending | Record final E2E acceptance and cleanup. | Task file records final commands, commits, risks, and clean generated/worktree status. |

## Per-Task Log

### T00 - Establish Task File And Freeze Baseline

- ID: T00
- Status: done
- Goal: Create this single canonical task file, capture the current repo/tooling baseline, map audit inputs to the migration task list, and rerun baseline backend/frontend checks.
- Code To Inspect: `backend/BACKEND_ISSUES_TODO_AUDIT.md`, `docs/openapi-first-migration.md`, `backend/go.mod`, `frontend/package.json`, `backend/cmd/firemail/main.go`, `frontend/src/lib/api.ts`, `Makefile`.
- Allowed Changes: create/update `OPENAPI_MIGRATION_TASKS.md` only.
- Implementation Notes:
  - `backend/BACKEND_ISSUES_TODO_AUDIT.md` lists dead routes, not-implemented send status/resend persistence, schema drift, token logging, delete consistency, dedup permission gaps, scheduler retry drift, race risk, OAuth refresh gaps, and folder path drift.
  - `docs/openapi-first-migration.md` recommends OpenAPI `3.0.3`, `oapi-codegen`, `orval`, stable Phase 1 route boundaries, generated outputs, and compatibility facade migration.
  - `backend/cmd/firemail/main.go` confirms hand-registered stable route groups plus attachment/SSE registration.
  - `frontend/src/lib/api.ts` confirms a handwritten fetch facade with the existing response envelope.
- Self Review Checklist:
  - [x] Required top-level sections exist.
  - [x] Every task item includes ID, Status, Goal, Code To Inspect, Allowed Changes, Implementation Notes, Self Review Checklist, Acceptance Commands, and Exit Result either in detail or in the shared task template below.
  - [x] Backend baseline command rerun and recorded.
  - [x] Frontend baseline command rerun and recorded.
- Acceptance Commands:
  - `cd /root/Coding/General/firemailplus/backend && go test ./...`
  - `cd /root/Coding/General/firemailplus/frontend && pnpm type-check`
- Exit Result: passed on 2026-04-30.
  - `cd /root/Coding/General/firemailplus/backend && go test ./...`: passed. Packages with tests under `internal/database`, `internal/encoding`, `internal/handlers`, `internal/models`, `internal/providers`, `internal/security`, `internal/services`, and `internal/sse` passed; other listed packages had no tests.
  - `cd /root/Coding/General/firemailplus/frontend && pnpm type-check`: passed with `tsc --noEmit`.

### T01 - Route, Frontend Call, Handler, And Schema Inventory

- ID: T01
- Status: done
- Goal: Script or otherwise enumerate Gin registered routes, frontend API facade calls, unregistered handlers, and model/migration drift.
- Code To Inspect: `backend/cmd/firemail/main.go`, `backend/internal/handlers`, `backend/internal/models`, `backend/database/migrations`, `frontend/src/lib/api.ts`, `frontend/src/types`.
- Allowed Changes: `OPENAPI_MIGRATION_TASKS.md`; optional read-only inventory scripts if committed later under an explicit tooling task.
- Implementation Notes:
  - Started after T00 passed.
  - Registered stable backend route inventory from `backend/cmd/firemail/main.go`:
    - `stable`: `GET /health`.
    - `stable`: auth routes `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`.
    - `stable`: OAuth routes `GET /api/v1/oauth/gmail`, `GET /api/v1/oauth/outlook`, `GET /api/v1/oauth/{provider}/callback`, `POST /api/v1/oauth/create-account`, `POST /api/v1/oauth/manual-config`.
    - `stable`: account routes `GET /api/v1/accounts`, `POST /api/v1/accounts`, `POST /api/v1/accounts/custom`, `GET /api/v1/accounts/{id}`, `PUT /api/v1/accounts/{id}`, `DELETE /api/v1/accounts/{id}`, `POST /api/v1/accounts/{id}/test`, `POST /api/v1/accounts/{id}/sync`, `PUT /api/v1/accounts/{id}/mark-read`, `POST /api/v1/accounts/batch/delete`, `POST /api/v1/accounts/batch/sync`, `POST /api/v1/accounts/batch/mark-read`.
    - `stable`: provider routes `GET /api/v1/providers`, `GET /api/v1/providers/detect`.
    - `stable`: mail routes `GET /api/v1/emails`, `GET /api/v1/emails/search`, `GET /api/v1/emails/{id}`, `PATCH /api/v1/emails/{id}`, `POST /api/v1/emails/send`, `DELETE /api/v1/emails/{id}`, `PUT /api/v1/emails/{id}/read`, `PUT /api/v1/emails/{id}/unread`, `PUT /api/v1/emails/{id}/star`, `PUT /api/v1/emails/{id}/move`, `PUT /api/v1/emails/{id}/archive`, `POST /api/v1/emails/{id}/reply`, `POST /api/v1/emails/{id}/reply-all`, `POST /api/v1/emails/{id}/forward`, `POST /api/v1/emails/batch`.
    - `stable`: folder routes `GET /api/v1/folders`, `POST /api/v1/folders`, `GET /api/v1/folders/{id}`, `PUT /api/v1/folders/{id}`, `DELETE /api/v1/folders/{id}`, `PUT /api/v1/folders/{id}/mark-read`, `PUT /api/v1/folders/{id}/sync`.
    - `stable`: group routes `GET /api/v1/groups`, `POST /api/v1/groups`, `PUT /api/v1/groups/reorder`, `PUT /api/v1/groups/{id}`, `PUT /api/v1/groups/{id}/default`, `DELETE /api/v1/groups/{id}`.
    - `stable`: SSE routes `GET /api/v1/sse`, `GET /api/v1/sse/events`, `GET /api/v1/sse/stats`, `POST /api/v1/sse/test`.
    - `stable-but-non-json`: static root/demo routes `GET /`, `GET /sse-demo`; exclude from `/api/v1` OpenAPI Phase 1 unless product docs need web pages.
  - Registered attachment route inventory from `backend/internal/handlers/attachment_handler.go` via `AttachmentHandler.RegisterRoutes(api)`:
    - `stable`: `POST /api/v1/attachments/upload`.
    - `stable`: `GET /api/v1/attachments/{id}/download`.
    - `stable`: `GET /api/v1/attachments/{id}/preview`.
    - `stable`: `GET /api/v1/attachments/{id}/progress`.
    - `stable`: `POST /api/v1/attachments/{id}/download`.
    - `stable`: `GET /api/v1/emails/{id}/attachments`.
    - `stable`: `POST /api/v1/emails/{id}/attachments/download`.
  - Frontend facade inventory from `frontend/src/lib/api.ts`:
    - `stable`: calls matching registered backend groups include auth, OAuth, accounts, groups, providers through account helpers, emails list/search/detail/update/delete/read/unread/star/move/archive/reply/reply-all/forward/batch/send, folders CRUD/mark-read/sync, and attachment download.
    - `drifted`: `GET /api/v1/emails/stats` is called by `getEmailStats()` but no matching registered route exists in `backend/cmd/firemail/main.go`.
    - `drifted`: `POST /api/v1/emails/draft` is called by `saveDraft()` but `EmailSendHandler.RegisterRoutes` is not registered by `setupRoutes`.
  - Unregistered handler route inventory:
    - `blocked`: `EmailSendHandler.RegisterRoutes` defines `POST /api/v1/emails/send/bulk`, `GET /api/v1/emails/send/{send_id}/status`, `POST /api/v1/emails/send/{send_id}/resend`, `POST /api/v1/emails/draft`, `PUT /api/v1/emails/draft/{id}`, `GET /api/v1/emails/draft/{id}`, `GET /api/v1/emails/drafts`, `DELETE /api/v1/emails/draft/{id}`, `POST /api/v1/emails/template`, `PUT /api/v1/emails/template/{id}`, `GET /api/v1/emails/template/{id}`, `GET /api/v1/emails/templates`, and `DELETE /api/v1/emails/template/{id}`. These must stay out of stable Phase 1 OpenAPI until T07/T08/T09 repair schema, persistence, and service semantics.
    - `blocked`: `DeduplicationHandler.RegisterRoutes` defines `/api/v1/deduplication/**`, but the handler is not registered and `validateAccountAccess` is an empty permission check. These routes move to T11.
    - `to-fix-and-expose`: `Handler` has auth refresh/change-password/profile, backup management, and soft-delete management methods, but no corresponding registered routes. These move to T13 with admin/security decisions.
  - Model and migration inventory:
    - `stable`: tables with matching model names include `users`, `email_accounts`, `folders`, `emails`, `attachments`, `drafts`, `oauth2_states`, `send_queue`, and `email_groups`.
    - `blocked`: `models.SentEmail.TableName()` returns `sent_emails`, but migrations do not create `sent_emails`.
    - `blocked`: `models.EmailDraft.TableName()` returns `email_drafts`, but migrations create `drafts`; `models.Draft` already maps to `drafts` and should remain the draft truth source.
    - `blocked`: `models.EmailQuota.TableName()` returns `email_quotas`, but migrations do not create `email_quotas`.
    - `blocked`: `models.EmailTemplate` maps to `email_templates`, but model columns (`description`, `text_body`, `html_body`, `variables`, `is_active`, `is_default`, `usage_count`, `last_used_at`) drift from migration `000008_create_email_templates_table.up.sql`.
- Self Review Checklist:
  - [x] route groups are complete enough for Phase 1 planning.
  - [x] frontend-only calls are identified.
  - [x] unregistered handlers are classified as blocked or to-fix-and-expose, not stable.
  - [x] schema drift is mapped to later repair tasks.
- Acceptance Commands: backend full tests; frontend type-check; inventory reviewed into `Findings`.
- Exit Result: passed on 2026-04-30.
  - `cd /root/Coding/General/firemailplus/backend && go test ./...`: passed.
  - `cd /root/Coding/General/firemailplus/frontend && pnpm type-check`: passed.
  - API classifications are recorded above and in `Findings` as `stable`, `drifted`, `blocked`, or `to-fix-and-expose`.

### T02 - OpenAPI Toolchain

- ID: T02
- Status: done
- Goal: Add the initial OpenAPI contract file, Redocly configuration, oapi-codegen configuration, Orval configuration, and repeatable generation commands.
- Code To Inspect: `backend/go.mod`, `frontend/package.json`, existing build scripts, current lockfiles.
- Allowed Changes: `openapi/**`, backend tool/config files, frontend generator config/package scripts/lockfile, generated directories.
- Implementation Notes:
  - Added `openapi/firemail.yaml` with OpenAPI `3.0.3`, a minimal `/health` operation, common `ErrorResponse`, `SuccessResponse`, and `bearerAuth` components.
  - Added `openapi/oapi-codegen.yaml` targeting `backend/internal/api/generated/firemail.gen.go`.
  - Added `redocly.yaml` and frontend scripts for `openapi:lint`, `generate:api`, and `check:api`.
  - Added `frontend/orval.config.ts` and pinned `orval` `8.9.0` plus `@redocly/cli` `2.30.3` in `frontend/package.json` / `frontend/pnpm-lock.yaml`.
  - Added root Make targets: `lint-openapi`, `generate-api-backend`, `generate-api-frontend`, `generate-api`, and `check-api-generated`.
  - Generated backend Go bindings under `backend/internal/api/generated/firemail.gen.go`.
  - Generated frontend SDK under `frontend/src/api/generated/firemail.ts` and schemas under `frontend/src/api/generated/model/**`.
  - Redocly lint currently exits successfully with 3 warnings: proprietary license has no URL, `/health` has no 4xx response, and `SuccessResponse` is defined for the v1 envelope but not used by the minimal T02 `/health` operation. These are acceptable for T02 and should be revisited during T03.
- Self Review Checklist:
  - [x] generated files are marked generator-owned.
  - [x] generated files contain no business logic.
  - [x] commands are deterministic and rerunnable.
  - [x] generated frontend import path compiles (`./model`).
- Acceptance Commands: OpenAPI lint; backend generate; frontend generate; backend full tests; frontend type-check; generated diff check.
- Exit Result: passed on 2026-04-30.
  - `cd frontend && pnpm install`: passed and updated `pnpm-lock.yaml`.
  - `cd frontend && pnpm openapi:lint`: passed with 3 warnings recorded above.
  - `cd backend && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0 -config ../openapi/oapi-codegen.yaml ../openapi/firemail.yaml`: passed.
  - `cd frontend && pnpm generate:api`: passed.
  - `cd backend && go test ./...`: passed, including the generated package compile check.
  - `cd frontend && pnpm type-check`: passed.
  - `git diff --exit-code -- backend/internal/api/generated frontend/src/api/generated`: passed after regeneration.
  - `make check-api-generated`: passed.

### T03 - Phase 1 OpenAPI Contract

- ID: T03
- Status: done
- Goal: Model current stable registered routes for health, auth, OAuth, accounts, providers, emails, folders, groups, attachments, SSE, and common response envelopes.
- Code To Inspect: `backend/cmd/firemail/main.go`, stable handler request/response DTOs, `frontend/src/lib/api.ts`, `frontend/src/types/api.ts`, `frontend/src/types/email.ts`.
- Allowed Changes: OpenAPI sources and generated artifacts only, plus this task file.
- Implementation Notes:
  - Started after T02 passed.
  - Do not publish `/emails/send/bulk`, send status/resend, draft/template, deduplication, or `/emails/stats` until blockers are fixed.
  - Expanded `openapi/firemail.yaml` to cover 62 Phase 1 stable registered routes.
  - Added `scripts/check-openapi-routes.mjs` and `make check-openapi-routes` to fail if stable routes are missing or blocked routes are accidentally published.
  - Removed `output-options.include-tags` from `openapi/oapi-codegen.yaml` so all Phase 1 operations generate.
  - Added `github.com/oapi-codegen/runtime` compatible with `oapi-codegen v2.6.0` because generated Gin parameter binding imports it.
- Self Review Checklist:
  - [x] path/method inventory matches registered routes.
  - [x] SSE endpoints use `text/event-stream`.
  - [x] binary attachment download uses `application/octet-stream`.
  - [x] blocked `/emails/stats`, draft/template, extended send, and deduplication routes are absent.
- Acceptance Commands: OpenAPI lint; backend generate; frontend generate; route drift check; backend full tests; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd frontend && pnpm openapi:lint`: passed with no warnings.
  - `node scripts/check-openapi-routes.mjs`: passed for 62 Phase 1 routes.
  - `cd backend && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0 -config ../openapi/oapi-codegen.yaml ../openapi/firemail.yaml`: passed.
  - `cd frontend && pnpm generate:api`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `git diff --exit-code -- backend/internal/api/generated frontend/src/api/generated`: passed after regeneration.
  - `make check-api-generated`: passed.

### T04 - Frontend SDK Compatibility Facade

- ID: T04
- Status: done
- Goal: Generate TypeScript SDK and make `frontend/src/lib/api.ts` delegate to it incrementally while preserving existing caller behavior.
- Code To Inspect: `frontend/src/lib/api.ts`, generated SDK, auth store behavior, API consumers.
- Allowed Changes: frontend SDK config/generated files, compatibility wrapper, focused facade tests or static checks.
- Implementation Notes:
  - Preserved 401 cleanup and existing `ApiResponse<T>` shape by keeping `ApiClient.request` as the execution path.
  - Stable methods now use generated Orval URL helpers through `generatedEndpoint()` so generated `/api/v1/**` paths remain compatible with existing `API_BASE_URL`.
  - Added `scripts/check-frontend-sdk-facade.mjs` and `make check-frontend-sdk-facade`.
  - The only remaining hardcoded request endpoints are `/emails/stats` and `/emails/draft`, which are T01 drifted/unregistered paths and intentionally not mapped to generated helpers.
- Self Review Checklist:
  - [x] no broad UI rewrites.
  - [x] migrated stable methods map to generated operations.
  - [x] legacy callers type-check.
  - [x] drifted unregistered methods remain visible instead of being falsely generated.
- Acceptance Commands: frontend type-check; SDK generation check; facade mapping check; backend full tests.
- Exit Result: passed on 2026-04-30.
  - `node scripts/check-frontend-sdk-facade.mjs`: passed with 49 generated helper mappings.
  - `cd frontend && pnpm type-check`: passed.
  - `cd frontend && pnpm openapi:lint`: passed.
  - backend and frontend code generation: passed.
  - `cd backend && go test ./...`: passed.
  - `make check-api-generated`: passed.

### T05 - Backend Generated Server Skeleton

- ID: T05
- Status: done
- Goal: Generate Go server/types and add a handwritten adapter that routes generated contracts to existing services.
- Code To Inspect: `backend/internal/handlers`, generated Go output, Gin route setup, response helpers.
- Allowed Changes: backend generated files, handwritten API adapter, route registration integration, route drift tests.
- Implementation Notes:
  - Added `backend/internal/api/server.go`.
  - The handwritten `api.Server` implements `generated.ServerInterface` and delegates to existing `handlers.Handler` plus `AttachmentHandler`.
  - Added `api.RegisterHandlers(router, handler)` as the generated route integration boundary. The production binary still uses the current handwritten router to avoid a duplicate-route cutover in this task.
  - Existing services remain business source; generated files stay logic-free.
- Self Review Checklist:
  - [x] route behavior remains compatible because main route registration is unchanged.
  - [x] generated DTOs are not hand-edited.
  - [x] adapter compiles against the generated server interface.
  - [x] route drift check remains green.
- Acceptance Commands: backend full tests; route drift check; OpenAPI lint/codegen; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./...`: passed.
  - `make check-api-generated`: passed.
  - `cd frontend && pnpm type-check`: passed.

### T06 - Security And Misleading Behavior Fixes

- ID: T06
- Status: done
- Goal: Remove token/header logging, reject weak production admin defaults, prohibit automatic dirty migration force, and redact SSE query-token logs.
- Code To Inspect: `backend/internal/middleware/auth.go`, config/bootstrap code, database migration startup, SSE handlers/logs.
- Allowed Changes: backend config/middleware/database/SSE code and focused tests.
- Implementation Notes:
  - `AuthRequiredWithService` now logs only auth header/token presence and length, never Authorization content or token prefixes.
  - Added `middleware.RedactedLogger()` and replaced `gin.Logger()` in `cmd/firemail/main.go` so SSE `?token=` and token-like query parameters are redacted from access logs.
  - Production admin bootstrap now rejects missing `ADMIN_PASSWORD` and the weak default `admin123`.
  - Dirty migration recovery now refuses automatic `Force` and returns a manual-repair error.
- Self Review Checklist:
  - [x] tests assert sensitive substrings do not appear.
  - [x] production weak defaults fail closed.
  - [x] dirty migration no longer forces state.
  - [x] SSE query token redaction is covered.
- Acceptance Commands: backend full tests; focused log/config/migration tests; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./...`: passed.
  - Focused tests added under `internal/middleware`, `internal/database`, and `internal/database/migration`.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed.

### T07 - Send, Draft, Template, And Quota Schema Repair

- ID: T07
- Status: done
- Goal: Add versioned migrations to reconcile `sent_emails`, `send_queue`, `drafts`, `email_templates`, and `email_quotas`.
- Code To Inspect: `backend/internal/models/sent_email.go`, draft/template services, migration SQL.
- Allowed Changes: SQL migrations, GORM models, services, migration/CRUD tests.
- Implementation Notes: Draft semantics must use existing `drafts` / `models.Draft`; do not introduce a second `email_drafts` truth source.
- Implementation Notes:
  - Added versioned migration `000021_fix_send_template_quota_schema` to create `sent_emails` and `email_quotas`, add missing `email_templates` columns, and backfill `text_body`/`html_body` from legacy `body`.
  - Removed the stale `EmailDraft` model that targeted nonexistent `email_drafts`; `models.Draft` and the existing `drafts` table remain the only draft truth source.
  - Kept a hidden `EmailTemplate.Body` compatibility column with a save hook that mirrors canonical `text_body`/`html_body` into legacy `body`, because migration `000008` created `body TEXT NOT NULL`.
  - Added `send_schema_test.go` to apply SQL migrations without using `AutoMigrate` for send/template/quota tables, then exercise `SentEmail`, `EmailTemplate`, `EmailQuota`, and absence of `email_drafts`.
- Self Review Checklist:
  - [x] empty SQLite schema applies the relevant versioned migrations.
  - [x] key GORM CRUD paths work against migration-created send/template/quota tables.
  - [x] draft semantics stay on `drafts` / `models.Draft`.
  - [x] no new `email_drafts` table or model truth source remains.
- Acceptance Commands: backend full tests; empty SQLite migration test; key CRUD tests; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed.
  - `git diff --check`: passed.

### T08 - Extended Send Capability

- ID: T08
- Status: done
- Goal: Complete persistent send status, resend, bulk send race safety, event linkage, and route/OpenAPI exposure.
- Code To Inspect: `backend/internal/services/email_sender.go`, `backend/internal/handlers/email_send_handler.go`, route setup, OpenAPI.
- Allowed Changes: send services/handlers/routes/OpenAPI/generated files/tests.
- Implementation Notes:
  - `StandardEmailSender.SendEmail` now creates a `send_queue` record before async send, updates queue status through `pending` / `sending` / `sent` / `failed`, and writes `sent_emails` as the final sent record.
  - `GetSendStatus` restores state from `send_queue` or final `sent_emails`; restart recovery is covered by a focused service test.
  - `ResendEmail` reloads the persisted composed email payload from `send_queue` and calls `SendEmail`, producing a new `send_id` instead of overwriting historical send records.
  - `SendBulkEmails` now writes results by index into a preallocated slice, avoiding concurrent append; returned result pointers are no longer mutated by the async sender goroutine.
  - Extended routes for bulk/status/resend/draft/template are registered through `Handler.RegisterExtendedEmailSendRoutes`; `POST /emails/send` remains on the legacy stable handler to avoid duplicate route registration.
  - OpenAPI now exposes the completed bulk/status/resend/draft/template routes, and generated backend/server adapter plus frontend facade helpers were regenerated and mapped.
  - Send status and resend handlers validate `send_id` ownership through `send_queue` or `sent_emails` before returning data or creating a resend.
- Self Review Checklist:
  - [x] no fake send-status success remains; status can be loaded from persisted tables.
  - [x] resend creates a new `send_id` from persisted payload.
  - [x] bulk result collection is race-safe.
  - [x] generated server adapter implements the expanded OpenAPI interface.
  - [x] frontend compatibility facade uses generated helpers for newly stable routes.
- Acceptance Commands: backend full tests; focused race test; send persistence tests; OpenAPI/codegen checks; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./...`: passed.
  - `cd backend && go test -race ./internal/services -run 'TestStandardEmailSender'`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed.
  - `git diff --check`: passed.

### T09 - Scheduled Send And Template Composition

- ID: T09
- Status: done
- Goal: Fix scheduled/retry status machine and inject template service into composer.
- Code To Inspect: `backend/internal/services/scheduled_email_service.go`, `backend/internal/services/email_composer.go`, template service, send queue models.
- Allowed Changes: service wiring/state machine/tests/OpenAPI updates if routes become public.
- Implementation Notes:
  - `StandardEmailComposer` now accepts injected `EmailTemplateService`; `Handler.New` wires the real template service into the shared composer used by send and scheduler paths.
  - Template sends can omit explicit subject/body when `template_id` is present; rendered template subject/text/html become the composed email content.
  - Template execution now uses `missingkey=error`, so missing template data fails clearly instead of silently rendering `<no value>`.
  - Scheduler now processes both due `scheduled` records and due `retry` records.
  - Scheduler no longer marks an async queued send as final `sent`; pending/sending results become truthful `queued`, while actual final state is carried by the generated send record.
- Self Review Checklist:
  - [x] retry tasks are selected and processed when `next_attempt` is due.
  - [x] missing template variables fail the compose path.
  - [x] scheduled queue acceptance is not misrepresented as final sent status.
  - [x] template service is wired into production composer construction.
- Acceptance Commands: backend full tests; scheduler/template focused tests; frontend type-check; OpenAPI/codegen if public contract changes.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/services -run 'TestStandardEmailComposer|TestScheduledEmailService'`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed.
  - `git diff --check`: passed.

### T10 - Mail Consistency Fixes

- ID: T10
- Status: done
- Goal: Repair delete semantics, OAuth refresh callbacks for move/archive, and folder sync path identity.
- Code To Inspect: `backend/internal/services/email_service.go`, provider factory/token callback code, sync service.
- Allowed Changes: service code and focused consistency tests.
- Implementation Notes:
  - `DeleteEmail` now fails before local `is_deleted` mutation when provider creation, IMAP connection, folder selection, or remote delete fails.
  - Delete uses `Folder.GetFullPath()` instead of raw `Path`, matching nested-folder path behavior elsewhere.
  - `MoveEmail` and archive-folder creation now use `CreateProviderForAccount` and install OAuth2 token update callbacks before connecting.
  - `MoveEmail` uses full source and target folder paths for IMAP select/move operations.
  - `SyncSpecificFolder` delegates with `folder.GetFullPath()` so duplicate leaf names in nested folders resolve to the intended folder.
- Self Review Checklist:
  - [x] remote delete failure leaves local email visible and does not publish delete event.
  - [x] OAuth token refresh callback is invoked and persisted on move/archive provider paths.
  - [x] nested folder sync uses full path, not ambiguous leaf name.
- Acceptance Commands: backend full tests; focused delete/move/archive/folder tests; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/services -run 'TestDeleteEmailRemoteFailure|TestMoveEmailRefreshes|TestSyncSpecificFolderUsesFullPath|TestDeleteUnreadEmail|TestMoveEmailPublishes'`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed.
  - `git diff --check`: passed.

### T11 - Deduplication Completion

- ID: T11
- Status: done
- Goal: Implement deduplication account permissions, schedule semantics, and public OpenAPI routes.
- Code To Inspect: `backend/internal/handlers/deduplication_handler.go`, `backend/internal/services/deduplication_manager.go`, account ownership queries.
- Allowed Changes: dedup services/handlers/routes/OpenAPI/generated files/tests.
- Implementation Notes:
  - `DeduplicationHandler.validateAccountAccess` now checks `email_accounts.id` and `user_id` instead of allowing all access.
  - Deduplication routes are registered through `Handler.RegisterDeduplicationRoutes`.
  - `StandardDeduplicationManager` now stores scheduled deduplication entries in a guarded in-memory schedule map and computes validated `next_run`; cancel removes the stored schedule.
  - OpenAPI exposes deduplicate account/user, report, schedule, cancel schedule, and stats endpoints; generated backend adapter forwards them without editing generated files.
- Self Review Checklist:
  - [x] cross-user account access is denied before manager methods run.
  - [x] schedule validates frequency/time and records `next_run`.
  - [x] cancel removes the scheduled entry.
  - [x] public OpenAPI route drift check covers deduplication routes.
- Acceptance Commands: backend full tests; dedup focused tests; OpenAPI/codegen; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/handlers ./internal/services -run 'TestDeduplication'`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed.
  - `git diff --check`: passed.

### T12 - Attachments, Provider Capability, And RFC Encoding

- ID: T12
- Status: done
- Goal: Make attachment preview/capability behavior truthful, align storage config/model, stop SMTP capability fake success, and fix Chinese header/filename encoding.
- Code To Inspect: attachment services/handlers, provider capability code, SMTP sender, parser/encoding utilities.
- Allowed Changes: backend services/handlers/providers/parser tests and OpenAPI if response contracts change.
- Implementation Notes:
  - `AttachmentService.PreviewAttachment` now returns typed preview errors with `ATTACHMENT_PREVIEW_UNSUPPORTED` or `ATTACHMENT_PREVIEW_FAILED` instead of embedding failures inside a successful response.
  - `AttachmentHandler.PreviewAttachment` maps unsupported previews to HTTP `415` with the v1 `ErrorResponse` envelope; OpenAPI documents the `415` response.
  - PDF preview no longer returns placeholder text as if the feature were implemented; unsupported and unknown types now use the same stable unsupported code path.
  - `StandardCapabilityDetector.testSMTPConnection` now requires configured SMTP host/port, obtains an SMTP client, connects with password or OAuth2 credentials, and returns false on unknown/not-tested/failing cases instead of unconditional success.
  - `StandardEmailComposer` now RFC 2047-encodes non-ASCII subject/display names and emits both RFC 2047 `filename` and RFC 5987 `filename*` parameters for non-ASCII attachment names.
  - `LocalFileStorage` no longer defaults to claiming text compression; checksum calculation is driven by `ChecksumType` with supported `md5` and `sha256` values and explicit rejection for unsupported algorithms.
- Self Review Checklist:
  - [x] unsupported preview is not returned as `success: true`.
  - [x] capability unknown/not-tested is explicit.
  - [x] RFC encoding tests cover non-ASCII subject, sender/recipient display names, and attachment filenames.
  - [x] storage checksum config is truthful and tested.
- Acceptance Commands: backend full tests; focused attachment/capability/encoding tests; frontend type-check; OpenAPI/codegen if needed.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/services -run 'TestAttachment|TestLocalFileStorage|TestStandardEmailComposerEncodes'`: passed.
  - `cd backend && go test ./internal/providers -run 'TestCapabilityDetectorSMTP'`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed; Redocly still reports the accepted 9 ambiguous v1 path warnings recorded in F011.
  - `git diff --check`: passed.

### T13 - Dead Management Interface Disposition

- ID: T13
- Status: done
- Goal: Complete and expose or remove auth refresh/change-password/profile, backup management, and soft-delete management interfaces.
- Code To Inspect: auth handlers/services, backup service/handler, soft-delete handler/service, admin permission checks.
- Allowed Changes: management routes/OpenAPI/generated files/tests.
- Implementation Notes:
  - Registered `POST /api/v1/auth/refresh`, `POST /api/v1/auth/change-password`, and `PUT /api/v1/auth/profile` behind authenticated v1 auth routes.
  - Registered admin-only backup routes under `/api/v1/admin/backups` for list/create/delete/restore/validate/cleanup.
  - Registered admin-only soft-delete routes under `/api/v1/admin/soft-deletes` for stats, cleanup, restore, and permanent delete.
  - OpenAPI now documents all T13 management routes, request bodies, admin security via bearer auth, and route drift tooling expects 94 stable routes.
  - Generated server adapter forwards management operations without modifying generated code.
  - Added handler tests proving non-admin admin access is denied, admin backup access is allowed, and auth management routes are registered.
- Self Review Checklist:
  - [x] non-admin denial covered.
  - [x] admin access covered.
  - [x] dead code is productized through registered routes.
  - [x] OpenAPI covers complete routes.
- Acceptance Commands: backend full tests; admin permission tests; OpenAPI/codegen; frontend type-check.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/handlers -run 'TestManagementRoutesRequireAdminRole|TestAuthManagementRoutesAreRegistered'`: passed after E011 and E013 fixes.
  - `cd backend && go test ./internal/api`: passed after E012 adapter signature fix.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed; route drift check now covers 94 routes and Redocly still reports the accepted 9 ambiguous v1 path warnings recorded in F011.
  - `git diff --check`: passed.

### T14 - Cache, Search, And Maintainability Fixes

- ID: T14
- Status: done
- Goal: Remove search request mutation, isolate cache invalidation by user, simplify reply subject logic, and document/implement HTML sanitize policy.
- Code To Inspect: search handlers/services, cache manager, reply/forward code, HTML rendering/sanitization code.
- Allowed Changes: backend/frontend focused logic and tests.
- Implementation Notes:
  - `SearchEmails` now works on a local request copy, so parsed `from:/to:/subject:/body:` tokens no longer mutate the caller's request object.
  - Email-list cache keys now include a clear `emails:user:{id}:` prefix before the request hash, allowing invalidation to delete only the current user's entries.
  - `EmailServiceImpl` and `SyncService` invalidation paths both use the same user-scoped prefix.
  - Reply and reply-all subject generation now share `replySubjectFor`, which adds one `Re:` prefix only when needed.
  - Compose-time HTML policy is explicit and tested: supplied HTML is escaped wholesale until a reviewed allowlist sanitizer is introduced.
- Self Review Checklist:
  - [x] no request side effects.
  - [x] cache invalidation cannot leak across users.
  - [x] reply subject logic is centralized.
  - [x] HTML policy is testable.
- Acceptance Commands: backend full tests; frontend type-check; focused side-effect/cache/reply/sanitize tests.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/services -run 'TestSearchEmailsDoesNotMutateRequest|TestEmailListCacheInvalidationIsScopedToUser|TestReplySubjectForAddsSinglePrefix|TestComposerHTMLPolicyEscapesMarkup'`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed; Redocly still reports the accepted 9 ambiguous v1 path warnings recorded in F011.
  - `git diff --check`: passed.

### T15 - Final Migration Validation And Cleanup

- ID: T15
- Status: done
- Goal: Remove or downgrade duplicate handwritten DTOs, update developer docs, and enforce OpenAPI-driven SDK/server/DTO/route consistency.
- Code To Inspect: generated artifacts, handwritten DTOs, README/developer docs, CI/build scripts.
- Allowed Changes: cleanup/docs/CI/check scripts/generated artifacts.
- Implementation Notes:
  - README now documents the OpenAPI-first developer workflow, generated directories, adapter/facade boundaries, and the accepted v1 Redocly ambiguous-path warnings.
  - `make check-api-generated` is the final combined gate for OpenAPI lint, route drift, SDK facade drift, backend codegen, and frontend SDK generation.
  - Generated code remains under `backend/internal/api/generated` and `frontend/src/api/generated`; handwritten business logic remains in adapters, handlers, services, and compatibility facade.
  - No duplicate generated DTO cleanup was performed beyond documenting the boundary because current v1 handlers still depend on compatibility request/response structs during gradual migration.
- Self Review Checklist:
  - [x] README explains generation.
  - [x] no generated files contain business logic.
  - [x] OpenAPI route and SDK facade drift checks pass.
  - [x] strict final gates pass.
- Acceptance Commands: backend full tests; frontend type-check; OpenAPI lint; backend codegen diff; frontend SDK codegen diff; route drift; SDK drift; race package tests; empty DB migration/CRUD tests.
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `cd backend && go test -race ./internal/services -run 'TestStandardEmailSender'`: passed.
  - `cd backend && go test ./internal/database ./internal/database/migration -run 'TestSend|TestSchema|TestMigration|TestProduction'`: passed.
  - `make check-api-generated`: passed; OpenAPI route drift covers 94 routes and SDK facade drift covers 62 generated helper mappings. Redocly still reports the accepted 9 ambiguous v1 path warnings recorded in F011.
  - `git diff --exit-code -- backend/internal/api/generated frontend/src/api/generated`: passed for tracked generated drift after regeneration.
  - `git diff --check`: passed.

### T16 - E2E Investigation Baseline And Remediation Task Chain

- ID: T16
- Status: done
- Goal: Track the source-level E2E investigation document, correct task-file consistency, and append the full E2E remediation chain as the next canonical work items.
- Code To Inspect: `docs/e2e-issue-investigation.md`, `OPENAPI_MIGRATION_TASKS.md`, `/tmp/firemailplus-e2e-artifacts/E2E_REPORT.md`, `/tmp/firemailplus-e2e-artifacts/backend-curl-report.json`, `/tmp/firemailplus-e2e-artifacts/frontend.har`.
- Allowed Changes: `OPENAPI_MIGRATION_TASKS.md`, `docs/e2e-issue-investigation.md`.
- Implementation Notes:
  - The E2E investigation document classifies 12 findings from the previous curl/jshook run and separates confirmed defects, contract mismatches, test-data issues, and external Docker registry risk.
  - The task index now appends T16 through T28, with one task per remediation/validation slice and a final clean E2E acceptance task.
  - Corrected the stale per-task T07 status from `in_progress` to `done` to match the task index and acceptance history.
  - Locked implementation defaults from planning: valid tokens roll refresh; user default groups can be renamed; batch account mark-read becomes asynchronous.
- Self Review Checklist:
  - [x] E2E investigation document is tracked.
  - [x] Task file has no contradictory T07 status.
  - [x] T16-T28 are present in the task index.
  - [x] Full baseline gates pass before commit.
- Acceptance Commands:
  - `cd backend && go test ./...`
  - `cd frontend && pnpm type-check`
  - `make check-api-generated`
  - `git diff --check`
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed; Redocly still reports the accepted 9 ambiguous v1 path warnings recorded in F011.
  - `git diff --check`: passed.

### T17 - Rolling Auth Refresh

- ID: T17
- Status: done
- Goal: Make `POST /api/v1/auth/refresh` roll any valid access token forward while preserving invalid/expired token rejection.
- Code To Inspect: `backend/internal/auth/jwt.go`, `backend/internal/auth/service.go`, `backend/internal/handlers/auth.go`, `backend/internal/handlers/management_routes_test.go`, `openapi/firemail.yaml`.
- Allowed Changes: auth implementation/tests, task file, OpenAPI only if wire contract changes.
- Implementation Notes:
  - Removed the 30-minute near-expiry eligibility gate from `JWTManager.RefreshToken`; `ValidateToken` remains the validity boundary.
  - Added JWT tests covering fresh valid token refresh and expired token rejection.
  - No wire-shape change was needed, so OpenAPI was left unchanged after E014 to avoid generated JSDoc churn.
- Self Review Checklist:
  - [x] Fresh valid token refresh succeeds.
  - [x] Expired token refresh still fails.
  - [x] Existing handler route tests still pass.
  - [x] Generated API artifacts remain synchronized.
- Acceptance Commands:
  - `cd backend && go test ./internal/auth ./internal/handlers -run 'TestRefreshToken|TestAuthManagementRoutesAreRegistered'`
  - `cd backend && go test ./...`
  - `cd frontend && pnpm type-check`
  - `make check-api-generated`
  - `git diff --check`
- Exit Result: passed on 2026-04-30.
  - `cd backend && go test ./internal/auth ./internal/handlers -run 'TestRefreshToken|TestAuthManagementRoutesAreRegistered'`: passed.
  - `cd backend && go test ./...`: passed.
  - `cd frontend && pnpm type-check`: passed.
  - `make check-api-generated`: passed after E014; Redocly still reports the accepted 9 ambiguous v1 path warnings recorded in F011.
  - `git diff --check`: passed.

## Findings

- F001: Phase 1 stable route boundary should start from real registrations in `backend/cmd/firemail/main.go`, plus attachment routes registered through `AttachmentHandler.RegisterRoutes(api)`.
- F002: `frontend/src/lib/api.ts` is still the compatibility seam for current UI callers.
- F003: `backend/BACKEND_ISSUES_TODO_AUDIT.md` identifies blocked surfaces that must not be promoted into stable OpenAPI before implementation and tests.
- F004: `docs/openapi-first-migration.md` confirms a curated first contract is safer than blindly generating from all existing handlers.
- F005: Frontend drift remains for `getEmailStats()` calling `GET /api/v1/emails/stats`; `saveDraft()` is now registered and mapped through generated SDK helper `getSaveDraftUrl()`.
- F006: Attachment API is registered indirectly through `AttachmentHandler.RegisterRoutes(api)`, so route drift tooling must include both direct `setupRoutes` registrations and handler-level registration helpers.
- F007: `EmailSendHandler` and `DeduplicationHandler` expose meaningful candidate routes in code, but they are not registered and have known persistence/permission blockers. They are backlog APIs, not Phase 1 stable APIs.
- F008: Schema drift blocks send/template/quota publication: `sent_emails`, `email_drafts`, and `email_quotas` have model references without migration-created tables, while `email_templates` has column drift between model and migration.
- F009: Orval `8.9.0` treats the `schemas` output target as a schema directory in this configuration. The stable generated layout is `frontend/src/api/generated/firemail.ts` plus `frontend/src/api/generated/model/**`, not a single `firemail.schemas.ts` file.
- F010: `make check-api-generated` is now the repo-level smoke command for T02 tooling, but because generated directories are newly untracked in this worktree, `git diff --exit-code` only validates tracked generated drift after the first commit/stage baseline exists.
- F011: T08 OpenAPI exposes legacy-compatible paths such as `/api/v1/emails/draft/{id}`. Redocly reports non-fatal `no-ambiguous-paths` warnings against `/api/v1/emails/{id}/...`; this reflects the existing v1 route shape and is accepted for compatibility until a future versioned route cleanup.
- F012: Attachment preview previously encoded runtime failures in a successful response body; the stable compatibility path is now explicit `ErrorResponse` with `ATTACHMENT_PREVIEW_UNSUPPORTED` for unpreviewable types.
- F013: Provider SMTP capability was a placeholder and could falsely report success; SMTP capability must be treated as unknown/failed unless a configured SMTP client connects successfully.
- F014: Auth refresh/change-password/profile, backup, and soft-delete handlers were live code but dead public surface; they are now registered and documented, with backup/soft-delete placed under admin-only routes.
- F015: Search token parsing was mutating `SearchEmailsRequest`; cache invalidation was global because email-list cache keys hid the user ID behind an MD5 hash. T14 fixed both with local request copies and user-visible cache key prefixes.
- F016: Final state is reproducible through `make check-api-generated`; generated server/SDK artifacts are still newly untracked in this worktree until the migration commit is staged/committed, so tracked generated drift is checked explicitly by path.

## Errors Encountered

- E014: T17 `make check-api-generated` failed after adding a non-wire auth refresh OpenAPI description because Orval regenerated only the `refreshToken` JSDoc. Different strategy applied: remove the description-only OpenAPI edit, keep the behavioral fix in code/tests, and rerun the generated check.
- E001: Initial Redocly lint failed because `SuccessResponse.data` used `nullable` without a sibling `type`, and `/health` lacked an explicit security declaration. Different strategy applied: define `data` as a nullable object and add `security: []` to the public health operation.
- E002: Initial Orval config generated schema files under a directory named `firemail.schemas.ts`, causing poor `from './.'` imports. Different strategy applied: use `frontend/src/api/generated/model` as the schema directory.
- E003: `pnpm type-check` failed because `orval.config.ts` used unsupported `output.prettier`. Different strategy applied: remove that field and rely on Orval's generated output formatting.
- E004: T03 Redocly lint failed when `SuccessResponse.data` used `nullable` with `oneOf`. Different strategy applied: make the envelope field a nullable object and rely on typed allOf overlays for route-specific data.
- E005: T03 backend tests failed because generated code imported `github.com/oapi-codegen/runtime` and `runtime/types` absent from `go.mod`. Different strategy applied: add `github.com/oapi-codegen/runtime`.
- E006: `github.com/oapi-codegen/runtime@v1.1.1` was incompatible with `oapi-codegen v2.6.0` generated parameter binding fields. Different strategy applied: upgrade to `github.com/oapi-codegen/runtime@v1.4.0`.
- E007: T07 schema CRUD test failed because `email_templates` lacked `deleted_at` while `models.EmailTemplate` embeds `BaseModel`. Different strategy applied: add `deleted_at` and its index in the versioned schema repair migration.
- E008: T07 schema CRUD test failed because legacy `email_templates.body` is `NOT NULL` but the canonical model writes `text_body`/`html_body`. Different strategy applied: keep a hidden GORM mapping for `body` and mirror canonical body fields into it on save.
- E009: T08 backend tests failed after OpenAPI expansion because the handwritten generated-server adapter did not implement new methods such as `CreateTemplate`. Different strategy applied: add adapter forwarding methods to the existing handler layer without editing generated files.
- E010: T08 focused bulk-send test initially failed on SQLite `database table is locked` under concurrent writes. Different strategy applied: keep production concurrency unchanged and make the test database deterministic with one connection and busy timeout.
- E011: T13 focused handler test initially failed because the test pre-hashed `models.User.Password`, then the GORM create hook hashed it again, causing login to reject the fixture. Different strategy applied: create fixture users with plaintext test passwords and let model hooks hash once.
- E012: T13 generated adapter compile check initially failed because OpenAPI enum path parameter `{table}` generated operation-specific typed parameters, while the handwritten adapter used raw `string`. Different strategy applied: update adapter signatures to use generated `RestoreSoftDeletedParamsTable` and `PermanentlyDeleteSoftDeletedParamsTable`.
- E013: T13 refresh route test initially expected a one-hour token to refresh, but `JWTManager.RefreshToken` only refreshes tokens within 30 minutes of expiry. Different strategy applied: use a short-lived test token so the route exercises the intended refresh-eligible path.

## Acceptance History

- T00 started on 2026-04-30.
- T00 passed on 2026-04-30: backend `go test ./...` and frontend `pnpm type-check`.
- T01 passed on 2026-04-30: route/frontend/schema inventory recorded and backend/frontend checks passed.
- T02 passed on 2026-04-30: OpenAPI tooling, backend codegen, frontend SDK codegen, backend tests, frontend type-check, and `make check-api-generated` passed.
- T03 passed on 2026-04-30: Phase 1 contract covers 62 stable routes, excludes blocked routes, and passes lint/codegen/drift/backend/frontend gates.
- T04 passed on 2026-04-30: frontend facade stable methods use generated URL helper mappings while preserving existing request behavior.
- T05 passed on 2026-04-30: generated Go server interface is implemented by a handwritten adapter and remains validation-covered.
- T06 passed on 2026-04-30: credential-bearing logs and unsafe production defaults were closed with focused tests.
- T07 passed on 2026-04-30: send/template/quota schema drift repaired with versioned SQL migration, draft truth source unified on `drafts`, and migration-created table CRUD validated.
- T08 passed on 2026-04-30: extended send/draft/template routes are registered and documented, send status is persisted and reloadable, resend creates new send records, and bulk send race/persistence tests pass.
- T09 passed on 2026-04-30: composer template injection, missing-variable failure, scheduled retry processing, and truthful queued status are covered by focused tests.
- T10 passed on 2026-04-30: remote delete failures are strong-consistency errors, move/archive paths refresh OAuth tokens through callbacks, and nested folder sync uses full folder path.
- T11 passed on 2026-04-30: deduplication routes are public and generated, cross-user access is denied, and schedule/cancel behavior is stateful and tested.
- T12 passed on 2026-04-30: attachment preview unsupported behavior, SMTP capability truthfulness, storage checksum config, and Chinese header/filename encoding are covered by focused tests and full gates.
- T13 passed on 2026-04-30: auth management and admin backup/soft-delete routes are registered, documented, generated, and permission-tested.
- T14 passed on 2026-04-30: search side effects, cache isolation, reply subject logic, and compose HTML policy are covered by focused tests and full gates.
- T15 passed on 2026-04-30: README generation docs were added and final backend, frontend, OpenAPI/codegen, route drift, SDK drift, race, migration/CRUD, generated drift, and diff checks passed.

## Deferred Decisions

- D001: Whether Phase 1 should use a single `openapi/firemail.yaml` initially or immediately split under `openapi/components/**`. Current default: start single-file unless generation becomes hard to maintain.
- D002: Exact route-drift tooling implementation. Current default: add a small backend test or script once OpenAPI paths exist.
- D003: Whether endpoint-by-endpoint responses remain enveloped forever or gradually move to direct resources. Current default: preserve the v1 envelope.
- D004: Redocly warning policy. Current default: warnings are allowed during T02 bootstrap, but T03 should decide whether to satisfy or explicitly disable `info-license-strict`, `operation-4xx-response`, and temporary unused envelope component warnings.
