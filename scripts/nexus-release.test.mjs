import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const config = JSON.parse(readFileSync(join(root, 'nexus-release.json'), 'utf8'));
const workflow = readFileSync(join(root, '.forgejo/workflows/nexus-release.yaml'), 'utf8');
const releaseId = '12345678-1234-1234-1234-123456789abc';
const source = 'a'.repeat(40);

test('CI and Docker share the pinned Node runtime', () => {
  const nodeVersion = readFileSync(join(root, '.node-version'), 'utf8').trim();
  const dockerfile = readFileSync(join(root, 'Dockerfile'), 'utf8');
  const ci = readFileSync(join(root, '.forgejo/workflows/ci.yml'), 'utf8');
  assert.match(nodeVersion, /^\d+\.\d+\.\d+$/);
  assert(dockerfile.includes(`node:${nodeVersion}-alpine AS ui-builder`));
  assert.match(workflow, /node-version-file: \.node-version/);
  assert.match(ci, /node-version-file: \.node-version/);
  assert.doesNotMatch(dockerfile, /pnpm@latest|\|\| pnpm install/);
});

function runBuild(t, { fail = '', tls = false, warmCache = false, version = '1.2.3', plan } = {}) {
  const dir = mkdtempSync(join(tmpdir(), 'waygates-release-test-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const bin = join(dir, 'bin');
  const log = join(dir, 'commands.jsonl');
  const cache = join(dir, 'cache');
  mkdirSync(bin);
  if (warmCache) {
    mkdirSync(cache);
    writeFileSync(join(cache, 'index.json'), '{}');
  }
  writeFileSync(join(dir, 'plan.json'), JSON.stringify(plan ?? { config, releaseId }));
  // Every command that could install, test, build, or publish is replaced. jq and
  // Bash still evaluate the real plan and script, including fail-fast behavior.
  const recorder = `#!${process.execPath}
const fs = require('node:fs');
const path = require('node:path');
const command = path.basename(process.argv[1]);
const args = process.argv.slice(2);
const entry = { command, args, cwd: process.cwd() };
const configFlag = args.indexOf('--buildkitd-config');
if (configFlag >= 0) entry.buildkit = fs.readFileSync(args[configFlag + 1], 'utf8');
fs.appendFileSync(process.env.RELEASE_TEST_LOG, JSON.stringify(entry) + '\\n');
if (process.env.RELEASE_TEST_FAIL === [command, ...args].join(' ')) process.exit(1);
if (command === 'docker' && args[0] === 'context' && args[1] === 'inspect') process.exit(1);
`;
  for (const command of ['pnpm', 'go', 'docker']) {
    writeFileSync(join(bin, command), recorder, { mode: 0o755 });
  }
  const result = spawnSync('bash', [join(root, 'scripts/nexus-release-build.sh')], {
    encoding: 'utf8',
    env: {
      ...process.env,
      PATH: `${bin}:${process.env.PATH}`,
      RELEASE_TEST_LOG: log,
      RELEASE_TEST_FAIL: fail,
      NEXUS_PLAN: join(dir, 'plan.json'),
      NEXUS_VERSION: version,
      NEXUS_SOURCE_SHA: source,
      WAYGATES_BUILDX_CACHE: cache,
      DOCKER_CERT_PATH: tls ? '/runner/certs/client' : '',
      DOCKER_HOST: tls ? 'tcp://docker:2376' : '',
    },
  });
  let commands = [];
  try {
    commands = readFileSync(log, 'utf8').trim().split('\n').filter(Boolean).map(JSON.parse);
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
  return { ...result, commands, cache };
}

test('release contract requires both image architectures and matching reviewed helper pins', () => {
  assert.equal(config.schema, 2);
  assert.equal(config.workflow, 'nexus-release.yaml');
  assert.deepEqual(config.artifacts, [
    {
      id: 'image',
      kind: 'oci',
      repository: 'registry.nexus.e412.in/aloks98/waygates',
      platforms: ['linux/amd64', 'linux/arm64'],
    },
  ]);
  assert.match(config.helperRevision, /^[0-9a-f]{40}$/);
  assert.equal(workflow.match(/HELPER_REVISION: ([0-9a-f]{40})/)[1], config.helperRevision);
  assert.match(workflow, /HELPER_TAG: helper-v\d+\.\d+\.\d+/);
  assert.match(workflow, /HELPER_SHA256: [0-9a-f]{64}/);
  assert.match(workflow, /sha256sum --check --status/);
  assert.match(workflow, /ref: \$\{\{ inputs.source_sha \}\}/);
  assert.match(workflow, /persist-credentials: false/);
  assert.match(workflow, /on:\n  workflow_dispatch:/);
  const triggers = workflow.match(/^on:\n([\s\S]*?)(?=^\S)/m)[1];
  assert.doesNotMatch(triggers, /^  (push|release|pull_request):/m);
  assert.throws(() => readFileSync(join(root, '.forgejo/workflows/release.yml')), {
    code: 'ENOENT',
  });
});

for (const warmCache of [false, true]) {
  test(`stage only the reviewed candidate after checks (cache ${warmCache ? 'hit' : 'miss'})`, (t) => {
    const result = runBuild(t, { warmCache });
    assert.equal(result.status, 0, result.stderr);
    const commands = result.commands;
    const builds = commands.filter(
      (c) => c.command === 'docker' && c.args[0] === 'buildx' && c.args[1] === 'build',
    );
    assert.equal(builds.length, 2);
    const builder = commands.find((c) => c.args[0] === 'buildx' && c.args[1] === 'create');
    const builderName = builder.args[builder.args.indexOf('--name') + 1];
    assert(
      builds.every((build) => build.args[build.args.indexOf('--builder') + 1] === builderName),
    );
    assert(builds[0].args.includes('--load'));
    assert.equal(builds[0].args[builds[0].args.indexOf('--tag') + 1], 'waygates-test:latest');
    const publish = builds[1];
    assert(publish.args.includes('--push'));
    assert.equal(
      publish.args[publish.args.indexOf('--tag') + 1],
      `registry.nexus.e412.in/aloks98/waygates:nexus-${releaseId}-image`,
    );
    assert.equal(publish.args[publish.args.indexOf('--platform') + 1], 'linux/amd64,linux/arm64');
    assert(publish.args.includes('VERSION=1.2.3'));
    assert(publish.args.includes(`org.opencontainers.image.revision=${source}`));
    assert(publish.args.includes('--provenance=false'));
    assert.equal(publish.args.includes('--cache-from'), warmCache);
    const checks = commands.filter((c) => c.command === 'pnpm');
    assert.deepEqual(
      checks.map((c) => c.args),
      [['install', '--frozen-lockfile'], ['check'], ['typecheck'], ['test', '--run']],
    );
    // Corepack must see ui/package.json before choosing a pnpm version.
    assert(checks.every((command) => command.cwd === join(root, 'ui')));
    const backend = commands.find((c) => c.command === 'go');
    assert.equal(backend.cwd, root.replace(/\/$/, ''));
    assert.deepEqual(backend.args, ['test', '-count=1', './backend/...']);
    assert(commands.indexOf(checks.at(-1)) < commands.indexOf(builds[0]));
    assert(commands.indexOf(builds[0]) < commands.indexOf(backend));
    assert(commands.indexOf(backend) < commands.indexOf(publish));
    assert.match(commands.find((c) => c.buildkit).buildkit, /docker-cache\.e412\.in/);
  });
}

test('preserves DinD TLS when creating the Buildx context', (t) => {
  const result = runBuild(t, { tls: true });
  assert.equal(result.status, 0, result.stderr);
  const create = result.commands.find((c) => c.args[0] === 'context' && c.args[1] === 'create');
  assert(
    create.args.includes(
      'host=tcp://docker:2376,ca=/runner/certs/client/ca.pem,cert=/runner/certs/client/cert.pem,key=/runner/certs/client/key.pem',
    ),
  );
  const builder = result.commands.find((c) => c.args[0] === 'buildx' && c.args[1] === 'create');
  assert.equal(builder.args.at(-1), 'waygates-release');
  assert.notEqual(builder.args[builder.args.indexOf('--name') + 1], create.args[2]);
});

test(
  'Docker accepts the release builder alongside its context',
  { skip: process.env.WAYGATES_TEST_DOCKER !== '1' },
  (t) => {
    const result = runBuild(t, { tls: true });
    assert.equal(result.status, 0, result.stderr);
    const context = result.commands.find((c) => c.args[0] === 'context' && c.args[1] === 'create');
    const builder = result.commands.find((c) => c.args[0] === 'buildx' && c.args[1] === 'create');
    const builderName = builder.args[builder.args.indexOf('--name') + 1];
    const configDir = mkdtempSync(join(tmpdir(), 'waygates-buildx-test-'));
    t.after(() => rmSync(configDir, { recursive: true, force: true }));
    // Isolate Docker/Buildx metadata and use the runner's daemon/TLS endpoint.
    // No bootstrap, image build, or publication: create only inspects the daemon.
    const env = { ...process.env, DOCKER_CONFIG: configDir };
    delete env.BUILDX_CONFIG;
    delete env.BUILDX_BUILDER;
    delete env.DOCKER_CONTEXT;
    const docker = (...args) => spawnSync('docker', args, { env, encoding: 'utf8' });
    const certPath = process.env.DOCKER_CERT_PATH;
    let endpoint = `host=${process.env.DOCKER_HOST || 'unix:///var/run/docker.sock'}`;
    if (certPath) {
      endpoint += `,ca=${certPath}/ca.pem,cert=${certPath}/cert.pem,key=${certPath}/key.pem`;
    }
    const createdContext = docker('context', 'create', context.args[2], '--docker', endpoint);
    assert.equal(createdContext.status, 0, createdContext.stderr);
    const createdBuilder = docker(
      'buildx',
      'create',
      '--name',
      builderName,
      '--driver',
      'docker-container',
      builder.args.at(-1),
    );
    assert.equal(createdBuilder.status, 0, createdBuilder.stderr);
  },
);

for (const fail of [
  'pnpm check',
  'pnpm typecheck',
  'pnpm test --run',
  'go test -count=1 ./backend/...',
]) {
  test(`does not publish after failure: ${fail}`, (t) => {
    const result = runBuild(t, { fail });
    assert.notEqual(result.status, 0);
    assert(!result.commands.some((c) => c.args.includes('--push')));
  });
}

for (const version of ['v1.2.3', '1.2.3-rc.1', 'latest']) {
  test(`rejects unsupported version ${version} before building`, (t) => {
    const result = runBuild(t, { version });
    assert.notEqual(result.status, 0);
    assert.deepEqual(result.commands, []);
  });
}

test('rejects an unexpected registry before running project checks', (t) => {
  const changed = structuredClone(config);
  changed.artifacts[0].repository = 'ghcr.io/aloks98/waygates';
  const result = runBuild(t, { plan: { config: changed, releaseId } });
  assert.notEqual(result.status, 0);
  assert.deepEqual(result.commands, []);
});
