#!/usr/bin/env node
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const specPath = resolve(process.cwd(), 'openapi/firemail.yaml');
const spec = readFileSync(specPath, 'utf8');

const expected = new Set([
  'GET /health',
  'POST /api/v1/auth/login',
  'POST /api/v1/auth/logout',
  'GET /api/v1/auth/me',
  'POST /api/v1/auth/refresh',
  'POST /api/v1/auth/change-password',
  'PUT /api/v1/auth/profile',
  'GET /api/v1/oauth/gmail',
  'GET /api/v1/oauth/outlook',
  'GET /api/v1/oauth/{provider}/callback',
  'POST /api/v1/oauth/create-account',
  'POST /api/v1/oauth/manual-config',
  'GET /api/v1/accounts',
  'POST /api/v1/accounts',
  'POST /api/v1/accounts/custom',
  'GET /api/v1/accounts/{id}',
  'PUT /api/v1/accounts/{id}',
  'DELETE /api/v1/accounts/{id}',
  'POST /api/v1/accounts/{id}/test',
  'POST /api/v1/accounts/{id}/sync',
  'PUT /api/v1/accounts/{id}/mark-read',
  'POST /api/v1/accounts/batch/delete',
  'POST /api/v1/accounts/batch/sync',
  'POST /api/v1/accounts/batch/mark-read',
  'GET /api/v1/providers',
  'GET /api/v1/providers/detect',
  'GET /api/v1/emails',
  'GET /api/v1/emails/search',
  'GET /api/v1/emails/{id}',
  'PATCH /api/v1/emails/{id}',
  'DELETE /api/v1/emails/{id}',
  'POST /api/v1/emails/send',
  'POST /api/v1/emails/send/bulk',
  'GET /api/v1/emails/send/{send_id}/status',
  'POST /api/v1/emails/send/{send_id}/resend',
  'POST /api/v1/emails/draft',
  'GET /api/v1/emails/draft/{id}',
  'PUT /api/v1/emails/draft/{id}',
  'DELETE /api/v1/emails/draft/{id}',
  'GET /api/v1/emails/drafts',
  'POST /api/v1/emails/template',
  'GET /api/v1/emails/template/{id}',
  'PUT /api/v1/emails/template/{id}',
  'DELETE /api/v1/emails/template/{id}',
  'GET /api/v1/emails/templates',
  'POST /api/v1/deduplication/accounts/{id}/deduplicate',
  'GET /api/v1/deduplication/accounts/{id}/report',
  'POST /api/v1/deduplication/accounts/{id}/schedule',
  'DELETE /api/v1/deduplication/accounts/{id}/schedule',
  'GET /api/v1/deduplication/accounts/{id}/stats',
  'POST /api/v1/deduplication/user/deduplicate',
  'GET /api/v1/admin/backups',
  'POST /api/v1/admin/backups',
  'DELETE /api/v1/admin/backups',
  'POST /api/v1/admin/backups/restore',
  'POST /api/v1/admin/backups/validate',
  'POST /api/v1/admin/backups/cleanup',
  'GET /api/v1/admin/soft-deletes/stats',
  'POST /api/v1/admin/soft-deletes/cleanup',
  'POST /api/v1/admin/soft-deletes/{table}/{id}/restore',
  'DELETE /api/v1/admin/soft-deletes/{table}/{id}',
  'PUT /api/v1/emails/{id}/read',
  'PUT /api/v1/emails/{id}/unread',
  'PUT /api/v1/emails/{id}/star',
  'PUT /api/v1/emails/{id}/move',
  'PUT /api/v1/emails/{id}/archive',
  'POST /api/v1/emails/{id}/reply',
  'POST /api/v1/emails/{id}/reply-all',
  'POST /api/v1/emails/{id}/forward',
  'POST /api/v1/emails/batch',
  'GET /api/v1/folders',
  'POST /api/v1/folders',
  'GET /api/v1/folders/{id}',
  'PUT /api/v1/folders/{id}',
  'DELETE /api/v1/folders/{id}',
  'PUT /api/v1/folders/{id}/mark-read',
  'PUT /api/v1/folders/{id}/sync',
  'GET /api/v1/groups',
  'POST /api/v1/groups',
  'PUT /api/v1/groups/reorder',
  'PUT /api/v1/groups/{id}',
  'DELETE /api/v1/groups/{id}',
  'PUT /api/v1/groups/{id}/default',
  'POST /api/v1/attachments/upload',
  'GET /api/v1/attachments/{id}/download',
  'POST /api/v1/attachments/{id}/download',
  'GET /api/v1/attachments/{id}/preview',
  'GET /api/v1/attachments/{id}/progress',
  'GET /api/v1/emails/{id}/attachments',
  'POST /api/v1/emails/{id}/attachments/download',
  'GET /api/v1/sse',
  'GET /api/v1/sse/events',
  'GET /api/v1/sse/stats',
  'POST /api/v1/sse/test',
]);

const forbiddenPathFragments = [
  '/api/v1/emails/stats',
];

for (const fragment of forbiddenPathFragments) {
  if (spec.includes(`  ${fragment}:`)) {
    console.error(`Forbidden unstable path is present in OpenAPI: ${fragment}`);
    process.exitCode = 1;
  }
}

const actual = new Set();
let currentPath = null;
for (const line of spec.split(/\r?\n/)) {
  const pathMatch = line.match(/^  (\/[^:]+):$/);
  if (pathMatch) {
    currentPath = pathMatch[1];
    continue;
  }
  const methodMatch = line.match(/^    (get|post|put|patch|delete):$/);
  if (currentPath && methodMatch) {
    actual.add(`${methodMatch[1].toUpperCase()} ${currentPath}`);
  }
}

const missing = [...expected].filter((route) => !actual.has(route)).sort();
const extra = [...actual].filter((route) => !expected.has(route)).sort();

if (missing.length > 0) {
  console.error('OpenAPI is missing stable registered routes:');
  for (const route of missing) console.error(`  - ${route}`);
  process.exitCode = 1;
}

if (extra.length > 0) {
  console.error('OpenAPI has routes outside the Phase 1 stable route set:');
  for (const route of extra) console.error(`  - ${route}`);
  process.exitCode = 1;
}

if (process.exitCode) {
  process.exit(process.exitCode);
}

console.log(`OpenAPI route drift check passed for ${actual.size} Phase 1 routes.`);
