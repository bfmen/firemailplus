import { readFileSync } from 'node:fs';
import { readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

const repoRoot = resolve(process.cwd(), '..');
const frontendSrc = resolve(process.cwd(), 'src');
const apiFacadePath = resolve(frontendSrc, 'lib/api.ts');
const searchPagePath = resolve(frontendSrc, 'components/mailbox/search-results-page.tsx');
const openapiPath = resolve(repoRoot, 'openapi/firemail.yaml');

function listSourceFiles(dir) {
  return readdirSync(dir).flatMap((entry) => {
    const path = join(dir, entry);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      if (path.includes(`${join('src', 'api', 'generated')}`)) return [];
      return listSourceFiles(path);
    }
    return /\.(ts|tsx)$/.test(entry) ? [path] : [];
  });
}

const noArgFolderCalls = [];
for (const path of listSourceFiles(frontendSrc)) {
  const source = readFileSync(path, 'utf8');
  if (/apiClient\.getFolders\(\s*\)/.test(source)) {
    noArgFolderCalls.push(relative(process.cwd(), path));
  }
}

if (noArgFolderCalls.length > 0) {
  console.error('apiClient.getFolders() must always receive accountId. Offenders:');
  for (const path of noArgFolderCalls) {
    console.error(`- ${path}`);
  }
  process.exit(1);
}

const apiFacade = readFileSync(apiFacadePath, 'utf8');
if (!/async getFolders\(accountId: number\)/.test(apiFacade)) {
  console.error('ApiClient.getFolders must require accountId: number.');
  process.exit(1);
}

const openapi = readFileSync(openapiPath, 'utf8');
if (!/\/api\/v1\/folders:[\s\S]*?\$ref: '#\/components\/parameters\/RequiredAccountIdQuery'/.test(openapi)) {
  console.error('OpenAPI listFolders must use RequiredAccountIdQuery.');
  process.exit(1);
}

const searchPage = readFileSync(searchPagePath, 'utf8');
if (!/router\.replace\(`\/mailbox\/search\?q=\$\{encodeURIComponent\(trimmedQuery\)\}`\)/.test(searchPage)) {
  console.error('SearchResultsPage must synchronize typed searches into the URL.');
  process.exit(1);
}

if (!/const currentQuery = activeSearchParams\.q \|\| searchParams\.get\('q'\) \|\| '';/.test(searchPage)) {
  console.error('SearchResultsPage empty state must use active search params before URL fallback.');
  process.exit(1);
}

console.log('Search contract checks passed.');
