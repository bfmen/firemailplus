#!/usr/bin/env node
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { performance } from 'node:perf_hooks';
import { fileURLToPath } from 'node:url';

const DEFAULT_ARTIFACT_DIR = '/tmp/firemailplus-e2e-artifacts';
const DEFAULT_BASE_URL = 'http://localhost:3000';
const args = new Set(process.argv.slice(2));

const dryRun = args.has('--dry-run');
const backendOnly = args.has('--backend-only');
const frontendOnly = args.has('--frontend-only');
const cleanArtifacts = args.has('--clean');
const artifactDir = resolve(process.env.E2E_ARTIFACT_DIR || DEFAULT_ARTIFACT_DIR);
const baseUrl = stripTrailingSlash(process.env.E2E_BASE_URL || DEFAULT_BASE_URL);
const apiBaseUrl = stripTrailingSlash(process.env.E2E_API_BASE_URL || `${baseUrl}/api/v1`);
const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const adminUsername = process.env.E2E_ADMIN_USERNAME || process.env.ADMIN_USERNAME || '';
const adminPassword = process.env.E2E_ADMIN_PASSWORD || process.env.ADMIN_PASSWORD || '';
const searchPrefix = process.env.E2E_SEARCH_PREFIX || `FMP-E2E-${new Date().toISOString().slice(0, 10)}`;
const frontendEvidenceInput = getFlagValue('--frontend-evidence');

if (cleanArtifacts) {
  rmSync(artifactDir, { recursive: true, force: true });
}
mkdirSync(artifactDir, { recursive: true });
const writtenArtifacts = [];

const secretValues = [
  adminPassword,
  process.env.E2E_OUTLOOK_PASSWORD_1,
  process.env.E2E_OUTLOOK_PASSWORD_2,
  process.env.E2E_OUTLOOK_REFRESH_TOKEN_1,
  process.env.E2E_OUTLOOK_REFRESH_TOKEN_2,
  process.env.JWT_SECRET,
  process.env.ENCRYPTION_KEY,
].filter(Boolean);

const backendReport = {
  base_url: baseUrl,
  api_base_url: apiBaseUrl,
  dry_run: dryRun,
  started_at: new Date().toISOString(),
  checks: [],
  skipped: [],
};

function stripTrailingSlash(value) {
  return value.replace(/\/+$/, '');
}

function getFlagValue(flag) {
  const argv = process.argv.slice(2);
  const index = argv.indexOf(flag);
  if (index === -1) return '';
  return argv[index + 1] || '';
}

function redact(value) {
  if (value === undefined || value === null) return value;
  let text = typeof value === 'string' ? value : JSON.stringify(value, null, 2);

  for (const secret of secretValues) {
    if (secret.length >= 4) {
      text = text.split(secret).join('[REDACTED_SECRET]');
    }
  }

  return text
    .replace(/Bearer\s+[A-Za-z0-9._~+/=-]+/gi, 'Bearer [REDACTED_TOKEN]')
    .replace(/"token"\s*:\s*"[^"]+"/gi, '"token":"[REDACTED_TOKEN]"')
    .replace(/"refresh_token"\s*:\s*"[^"]+"/gi, '"refresh_token":"[REDACTED_TOKEN]"')
    .replace(/"access_token"\s*:\s*"[^"]+"/gi, '"access_token":"[REDACTED_TOKEN]"')
    .replace(/"password"\s*:\s*"[^"]*"/gi, '"password":"[REDACTED_PASSWORD]"')
    .replace(/[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{20,}/g, '[REDACTED_JWT]')
    .replace(/token=([^&\s"]+)/gi, 'token=[REDACTED_TOKEN]');
}

function writeArtifact(relativePath, content) {
  const path = resolve(artifactDir, relativePath);
  mkdirSync(dirname(path), { recursive: true });
  const text = typeof content === 'string' ? content : JSON.stringify(content, null, 2);
  writeFileSync(path, redact(text));
  writtenArtifacts.push(path);
  return path;
}

async function requestCheck(name, method, path, options = {}) {
  const started = performance.now();
  const url = path.startsWith('http') ? path : `${apiBaseUrl}${path}`;
  const headers = {
    Accept: 'application/json',
    ...(options.body ? { 'Content-Type': 'application/json' } : {}),
    ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
  };

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs || 30000);
  const record = {
    name,
    method,
    url: redact(url),
    expected_status: options.expectedStatus || [200],
    status: 0,
    ok: false,
    duration_ms: 0,
    summary: '',
  };

  try {
    const response = await fetch(url, {
      method,
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
      signal: controller.signal,
    });
    const text = await response.text();
    const expected = Array.isArray(options.expectedStatus)
      ? options.expectedStatus
      : [options.expectedStatus || 200];
    record.status = response.status;
    record.ok = expected.includes(response.status);
    record.summary = summarizeBody(text);
    record.duration_ms = Math.round(performance.now() - started);
    backendReport.checks.push(record);
    return { record, response, text };
  } catch (error) {
    record.summary = error instanceof Error ? error.message : String(error);
    record.duration_ms = Math.round(performance.now() - started);
    backendReport.checks.push(record);
    return { record, response: null, text: '' };
  } finally {
    clearTimeout(timeout);
  }
}

function summarizeBody(text) {
  if (!text) return '';
  const redacted = redact(text);
  return redacted.length > 600 ? `${redacted.slice(0, 600)}...` : redacted;
}

function parseJson(text) {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

async function runBackendChecks() {
  if (dryRun) {
    backendReport.skipped.push('dry-run: backend network checks not executed');
    return;
  }

  await requestCheck('health', 'GET', `${baseUrl}/api/v1/health`);

  if (!adminUsername || !adminPassword) {
    backendReport.skipped.push('auth checks skipped: E2E_ADMIN_USERNAME/E2E_ADMIN_PASSWORD not set');
    return;
  }

  const login = await requestCheck('auth login', 'POST', '/auth/login', {
    body: { username: adminUsername, password: adminPassword },
  });
  const loginJson = parseJson(login.text);
  const token = loginJson?.data?.token || loginJson?.token || '';
  if (!token) {
    backendReport.skipped.push('authenticated checks skipped: login did not return token');
    return;
  }

  await requestCheck('auth me', 'GET', '/auth/me', { token });
  await requestCheck('auth refresh', 'POST', '/auth/refresh', { token });
  await requestCheck('providers', 'GET', '/providers', { token });
  const accountsResult = await requestCheck('accounts list', 'GET', '/accounts', { token });
  await requestCheck('groups list', 'GET', '/groups', { token });
  await requestCheck('sse stats', 'GET', '/sse/stats', { token });
  await requestCheck('search known prefix', 'GET', `/emails/search?q=${encodeURIComponent(searchPrefix)}&page=1&page_size=20`, { token });

  const accountsJson = parseJson(accountsResult.text);
  const accounts = Array.isArray(accountsJson?.data) ? accountsJson.data : [];
  for (const account of accounts.slice(0, 4)) {
    if (typeof account.id === 'number') {
      await requestCheck(`folders account ${account.id}`, 'GET', `/folders?account_id=${account.id}`, { token });
    }
  }
}

function buildFrontendJshookPlan() {
  const standaloneAssetResult = prepareStandaloneAssets();
  const plan = {
    base_url: baseUrl,
    artifact_dir: artifactDir,
    standalone_assets: standaloneAssetResult,
    required_env: ['E2E_ADMIN_USERNAME', 'E2E_ADMIN_PASSWORD'],
    output_artifacts: [
      'frontend-jshook-report.json',
      'frontend-console-redacted.log',
      'frontend-network-redacted.json',
      'frontend-screenshots/',
    ],
    steps: [
      'Clear cookies, localStorage, and sessionStorage before login.',
      'Open the production base URL and wait for the login form.',
      'Log in with E2E_ADMIN_USERNAME/E2E_ADMIN_PASSWORD without recording raw password values.',
      'Open mailbox desktop route and wait for account/sidebar data.',
      'Keep the mailbox route open for at least 120 seconds and record whether heartbeat or SSE events are observed.',
      'Scan console messages for heartbeat timeout loops, raw token query strings, Bearer tokens, passwords, and JWT-like strings.',
      'Open search page, run a query for E2E_SEARCH_PREFIX, and verify the empty state or result state matches backend data.',
      'Assert network evidence has no GET /api/v1/folders request missing account_id.',
      'Write redacted console and network evidence under the artifact directory.',
    ],
    leak_patterns: [
      'Authorization',
      'Bearer ',
      'token=',
      'password',
      'JWT-like three-segment token',
    ],
  };
  writeArtifact('frontend-jshook-plan.json', plan);
}

function prepareStandaloneAssets() {
  const frontendDir = resolve(repoRoot, 'frontend');
  const standaloneDir = resolve(frontendDir, '.next/standalone');
  const sourceStatic = resolve(frontendDir, '.next/static');
  const targetStatic = resolve(standaloneDir, '.next/static');
  const sourcePublic = resolve(frontendDir, 'public');
  const targetPublic = resolve(standaloneDir, 'public');

  if (!existsSync(standaloneDir)) {
    return {
      status: 'skipped',
      reason: 'frontend/.next/standalone does not exist; run pnpm build first',
    };
  }

  const copied = [];
  if (existsSync(sourceStatic)) {
    rmSync(targetStatic, { recursive: true, force: true });
    mkdirSync(dirname(targetStatic), { recursive: true });
    cpSync(sourceStatic, targetStatic, { recursive: true });
    copied.push('.next/static');
  }

  if (existsSync(sourcePublic)) {
    rmSync(targetPublic, { recursive: true, force: true });
    cpSync(sourcePublic, targetPublic, { recursive: true });
    copied.push('public');
  }

  return {
    status: copied.length > 0 ? 'prepared' : 'skipped',
    copied,
    target: redact(standaloneDir),
  };
}

function ingestFrontendEvidence() {
  if (!frontendEvidenceInput) return;
  const raw = readFileSync(frontendEvidenceInput, 'utf8');
  const redacted = redact(raw);
  writeArtifact('frontend-jshook-redacted-evidence.txt', redacted);
}

function writeSummary() {
  const failed = backendReport.checks.filter((check) => !check.ok);
  const lines = [
    '# FireMailPlus Local Production E2E Report',
    '',
    `- Generated at: ${new Date().toISOString()}`,
    `- Base URL: ${baseUrl}`,
    `- API Base URL: ${apiBaseUrl}`,
    `- Dry run: ${dryRun}`,
    `- Backend checks: ${backendReport.checks.length}`,
    `- Backend failures: ${failed.length}`,
    `- Skipped: ${backendReport.skipped.length}`,
    '',
    '## Backend Checks',
    '',
    '| Name | Method | Status | OK | Duration ms | Summary |',
    '| --- | --- | --- | --- | --- | --- |',
    ...backendReport.checks.map((check) =>
      `| ${check.name} | ${check.method} | ${check.status} | ${check.ok ? 'yes' : 'no'} | ${check.duration_ms} | ${String(check.summary).replace(/\n/g, ' ').slice(0, 180)} |`
    ),
    '',
    '## Skipped',
    '',
    ...(backendReport.skipped.length ? backendReport.skipped.map((item) => `- ${item}`) : ['- none']),
    '',
    '## Frontend Jshook',
    '',
    'Use `frontend-jshook-plan.json` as the deterministic browser flow contract. Raw browser evidence must be redacted before it is saved or shared.',
  ];

  writeArtifact('backend-curl-report.json', backendReport);
  writeArtifact('E2E_REPORT.md', lines.join('\n'));
  writeArtifact('RUN_MANIFEST.json', {
    artifact_dir: artifactDir,
    clean_start: cleanArtifacts,
    dry_run: dryRun,
    files_written: writtenArtifacts.map((path) => path.replace(`${artifactDir}/`, '')),
  });
}

async function main() {
  if (!frontendOnly) {
    await runBackendChecks();
  }
  if (!backendOnly) {
    buildFrontendJshookPlan();
    ingestFrontendEvidence();
  }
  backendReport.finished_at = new Date().toISOString();
  writeSummary();

  const failed = backendReport.checks.filter((check) => !check.ok);
  if (!dryRun && failed.length > 0) {
    console.error(`E2E backend checks failed: ${failed.length}`);
    process.exit(1);
  }
  console.log(`E2E artifacts written to ${artifactDir}`);
}

main().catch((error) => {
  console.error(redact(error instanceof Error ? error.stack || error.message : String(error)));
  process.exit(1);
});
