# Local Production E2E Harness

This repository keeps a reproducible local production E2E entrypoint at:

```bash
node scripts/e2e-local-production.mjs
```

The harness writes redacted artifacts under `/tmp/firemailplus-e2e-artifacts` by default:

- `backend-curl-report.json`: machine-readable backend API checks.
- `E2E_REPORT.md`: human-readable summary.
- `frontend-jshook-plan.json`: deterministic browser flow contract for jshook/manual Playwright execution.
- `frontend-jshook-redacted-evidence.txt`: optional redacted frontend evidence when `--frontend-evidence` is used.

## Environment

Use environment variables instead of committing credentials:

```bash
export E2E_BASE_URL=http://localhost:3000
export E2E_API_BASE_URL=http://localhost:3000/api/v1
export E2E_ADMIN_USERNAME=admin
export E2E_ADMIN_PASSWORD='replace-with-test-password'
export E2E_SEARCH_PREFIX=FMP-E2E-20260430
```

The harness redacts known secret values, bearer tokens, JWT-like tokens, password fields, and SSE `token=` query values before writing artifacts.

## Dry Run

Use dry-run mode to verify report generation without a running instance:

```bash
node scripts/e2e-local-production.mjs --dry-run --clean
```

## Backend Curl Checks

The backend portion uses Node `fetch` as a curl-style runner. It checks health, providers, login, auth refresh, account/group lists, SSE stats, search, and per-account folder listing when credentials and account data are available.

Run backend only:

```bash
node scripts/e2e-local-production.mjs --backend-only --clean
```

## Frontend Jshook Flow

The script writes `frontend-jshook-plan.json`; use it as the browser flow contract in jshook:

1. Clear cookies, localStorage, and sessionStorage.
2. Log in with `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD`.
3. Open the mailbox route and wait for sidebar/account data.
4. Keep the page open for at least 120 seconds and record SSE heartbeat or event observations.
5. Open the search page, run `E2E_SEARCH_PREFIX`, and verify result or empty-state text.
6. Assert network evidence has no `GET /api/v1/folders` request missing `account_id`.
7. Save only redacted console/network artifacts under `/tmp/firemailplus-e2e-artifacts`.

If jshook exports raw evidence to a local file, redact it through the harness:

```bash
node scripts/e2e-local-production.mjs --frontend-only --frontend-evidence /tmp/raw-frontend-evidence.txt
```

Do not commit `/tmp/firemailplus-e2e-artifacts` or any raw credentials.
