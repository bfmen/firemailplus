## Summary

- 

## Validation

- [ ] `cd backend && go test ./...`
- [ ] `cd frontend && pnpm type-check`
- [ ] `make check-api-generated`
- [ ] `git diff --check`

## OpenAPI And API Contract

- [ ] Public API changes are reflected in `openapi/firemail.yaml`.
- [ ] Generated backend and frontend artifacts are synchronized.
- [ ] Existing `/api/v1` compatibility and response envelopes are preserved, or the intentional exception is documented.

## Deployment And E2E Impact

- [ ] Docker/build/deployment impact has been considered.
- [ ] Database migration impact has been considered.
- [ ] E2E or manual smoke coverage is documented when user-facing behavior changes.

## Security Checklist

- [ ] No passwords, JWTs, OAuth refresh tokens, cookies, raw HARs, local databases, or runtime credentials are committed.
- [ ] Logs and screenshots included in the PR are sanitized.
