import { readFileSync } from 'node:fs';

const harness = readFileSync('scripts/e2e-local-production.mjs', 'utf8');

const checks = [
  {
    name: 'uses canonical artifact directory',
    ok: harness.includes("const DEFAULT_ARTIFACT_DIR = '/tmp/firemailplus-e2e-artifacts';"),
  },
  {
    name: 'supports dry-run mode',
    ok: harness.includes("const dryRun = args.has('--dry-run');"),
  },
  {
    name: 'supports clean artifact runs and manifest',
    ok:
      harness.includes("const cleanArtifacts = args.has('--clean');") &&
      harness.includes("writeArtifact('RUN_MANIFEST.json'"),
  },
  {
    name: 'runs backend curl-style checks',
    ok:
      harness.includes("requestCheck('auth login'") &&
      harness.includes("requestCheck('accounts list'") &&
      harness.includes("requestCheck('sse stats'"),
  },
  {
    name: 'writes frontend jshook plan',
    ok:
      harness.includes('frontend-jshook-plan.json') &&
      harness.includes('Keep the mailbox route open for at least 120 seconds'),
  },
  {
    name: 'redacts credential patterns',
    ok:
      harness.includes('REDACTED_TOKEN') &&
      harness.includes('REDACTED_PASSWORD') &&
      harness.includes('REDACTED_JWT') &&
      harness.includes('token=[REDACTED_TOKEN]'),
  },
  {
    name: 'writes machine and markdown reports',
    ok:
      harness.includes("writeArtifact('backend-curl-report.json'") &&
      harness.includes("writeArtifact('E2E_REPORT.md'"),
  },
  {
    name: 'does not embed the supplied Outlook accounts',
    ok:
      !harness.includes('@outlook.com') &&
      !harness.includes('M.C502') &&
      !harness.includes('M.C530'),
  },
];

const failed = checks.filter((check) => !check.ok);
if (failed.length > 0) {
  console.error('E2E harness validation failed:');
  for (const check of failed) {
    console.error(`- ${check.name}`);
  }
  process.exit(1);
}

console.log('E2E harness validation checks passed.');
