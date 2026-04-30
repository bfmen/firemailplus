import { readFileSync } from 'node:fs';

const dockerfile = readFileSync('Dockerfile', 'utf8');
const buildScript = readFileSync('scripts/docker-build.sh', 'utf8');
const compose = readFileSync('docker-compose.yml', 'utf8');
const workflow = readFileSync('.github/workflows/docker-build.yml', 'utf8');
const readme = readFileSync('README.md', 'utf8');

const checks = [
  {
    name: 'Dockerfile declares GO_BASE_IMAGE arg',
    ok: /ARG GO_BASE_IMAGE=golang:1\.24-alpine/.test(dockerfile),
  },
  {
    name: 'Dockerfile uses GO_BASE_IMAGE in backend FROM',
    ok: /FROM \$\{GO_BASE_IMAGE\} AS backend-builder/.test(dockerfile),
  },
  {
    name: 'Dockerfile declares NODE_BASE_IMAGE arg',
    ok: /ARG NODE_BASE_IMAGE=node:20-alpine/.test(dockerfile),
  },
  {
    name: 'Dockerfile uses NODE_BASE_IMAGE in frontend/runtime FROM',
    ok:
      /FROM \$\{NODE_BASE_IMAGE\} AS frontend-builder/.test(dockerfile) &&
      /FROM \$\{NODE_BASE_IMAGE\}\n/.test(dockerfile),
  },
  {
    name: 'docker-build script passes base image build args',
    ok:
      /--build-arg "GO_BASE_IMAGE=\$\{GO_BASE_IMAGE\}"/.test(buildScript) &&
      /--build-arg "NODE_BASE_IMAGE=\$\{NODE_BASE_IMAGE\}"/.test(buildScript),
  },
  {
    name: 'docker-build script retries transient failures',
    ok:
      /DOCKER_BUILD_RETRIES/.test(buildScript) &&
      /for attempt in \$\(seq 1 "\$\{DOCKER_BUILD_RETRIES\}"\)/.test(buildScript),
  },
  {
    name: 'docker-compose passes base image args',
    ok:
      /GO_BASE_IMAGE: \$\{GO_BASE_IMAGE:-golang:1\.24-alpine\}/.test(compose) &&
      /NODE_BASE_IMAGE: \$\{NODE_BASE_IMAGE:-node:20-alpine\}/.test(compose),
  },
  {
    name: 'GitHub workflow exposes base image inputs and build args',
    ok:
      /go_base_image:/.test(workflow) &&
      /node_base_image:/.test(workflow) &&
      /GO_BASE_IMAGE=\$\{\{ env\.GO_BASE_IMAGE \}\}/.test(workflow) &&
      /NODE_BASE_IMAGE=\$\{\{ env\.NODE_BASE_IMAGE \}\}/.test(workflow),
  },
  {
    name: 'README documents base image override usage',
    ok:
      /GO_BASE_IMAGE=registry\.example\.com\/library\/golang:1\.24-alpine/.test(readme) &&
      /DOCKER_BUILD_RETRIES=5/.test(readme),
  },
];

const failed = checks.filter((check) => !check.ok);
if (failed.length > 0) {
  console.error('Docker build resilience configuration check failed:');
  for (const check of failed) {
    console.error(`- ${check.name}`);
  }
  process.exit(1);
}

console.log('Docker build resilience configuration checks passed.');
