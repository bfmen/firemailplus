import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const sourcePath = resolve(process.cwd(), 'src/lib/sse-client.ts');
const source = readFileSync(sourcePath, 'utf8');

const consoleCalls = [...source.matchAll(/console\.(?:debug|error|info|log|warn)\(([\s\S]*?)\);/g)];
const unsafeCalls = consoleCalls
  .map((match) => match[0])
  .filter((call) =>
    /\burl\b|token=|this\.config\.token|buildEventSourceUrl\(/.test(call)
  );

if (unsafeCalls.length > 0) {
  console.error('SSE client console output may expose a credential-bearing URL or token:');
  for (const call of unsafeCalls) {
    console.error(call);
  }
  process.exit(1);
}

if (!source.includes('getSafeConnectionInfo()')) {
  console.error('SSE client must log sanitized connection metadata via getSafeConnectionInfo().');
  process.exit(1);
}

if (!source.includes("closeEventSourceForReconnect('heartbeat-timeout')")) {
  console.error('SSE heartbeat timeout must close the active EventSource before reconnecting.');
  process.exit(1);
}

console.log('SSE client redaction/static reconnect checks passed.');
