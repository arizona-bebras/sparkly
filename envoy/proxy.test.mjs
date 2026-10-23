import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';
import test from 'node:test';

const directory = fileURLToPath(new URL('.', import.meta.url));
const image = (await readFile(join(directory, 'Dockerfile'), 'utf8')).match(/^FROM (.+)$/m)[1];
const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
const jwk = { ...publicKey.export({ format: 'jwk' }), kid: 'test-key', alg: 'RS256', use: 'sig' };
const now = () => Math.floor(Date.now() / 1000);

function token(overrides = {}) {
  const header = Buffer.from(JSON.stringify({ alg: 'RS256', kid: jwk.kid })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    id: 'user-123', type: 'access', exp: now() + 300, ...overrides,
  })).toString('base64url');
  const input = `${header}.${payload}`;
  return `${input}.${sign('RSA-SHA256', Buffer.from(input), privateKey).toString('base64url')}`;
}

async function listen(server) {
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '0.0.0.0', resolve);
  });
  return server.address().port;
}

test('Envoy routing and cookie authentication', { timeout: 90000 }, async (t) => {
  const temporary = await mkdtemp(join(tmpdir(), 'sparkly-envoy-'));
  const fixtures = [];
  let jwksAvailable = false;
  let logs = '';
  let proxy;
  t.after(async () => {
    if (proxy && proxy.exitCode === null) {
      proxy.kill('SIGTERM');
      await new Promise((resolve) => proxy.once('exit', resolve));
    }
    for (const server of fixtures) {
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
    }
    await rm(temporary, { recursive: true, force: true });
  });

  let config = await readFile(join(directory, 'envoy.yaml'), 'utf8');
  const host = process.env.ENVOY_BIN ? '127.0.0.1' : 'host.docker.internal';
  for (const service of ['frontend', 'backend', 'analytics', 'ml']) {
    const server = createServer((req, res) => {
      if (service === 'backend' && req.url === '/.well-known/jwks.json') {
        res.writeHead(jwksAvailable ? 200 : 503, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ keys: jwksAvailable ? [jwk] : [] }));
        return;
      }
      res.writeHead(req.url === '/sparkly.backend.v1.TestService/Unauthorized' ? 401 : 200, {
        'content-type': 'application/json',
        'set-cookie': 'fixture=value; Path=/; HttpOnly',
      });
      res.end(JSON.stringify({ service, path: req.url, user: req.headers['x-user-id'] ?? null }));
    });
    fixtures.push(server);
    const port = await listen(server);
    config = config.replace(
      new RegExp(`address: ${service}, port_value: \\d+`),
      `address: ${host}, port_value: ${port}`,
    );
    if (service === 'backend') {
      config = config.replace('http://backend:8080/', `http://${host}:${port}/`);
    }
  }
  const reservation = createServer();
  const port = await listen(reservation);
  await new Promise((resolve) => reservation.close(resolve));
  config = config.replace('port_value: 10000', `port_value: ${port}`);
  await writeFile(join(temporary, 'envoy.yaml'), config);
  const args = ['-c', process.env.ENVOY_BIN ? join(temporary, 'envoy.yaml') : '/test/envoy.yaml',
    '--concurrency', '1', '--log-level', 'warn'];
  proxy = process.env.ENVOY_BIN
    ? spawn(process.env.ENVOY_BIN, args)
    : spawn('docker', ['run', '--rm', '--network', 'host',
      '--add-host', 'host.docker.internal:host-gateway',
      '-v', `${directory}:/etc/envoy:ro`, '-v', `${temporary}:/test:ro`, image, ...args]);
  proxy.stdout.on('data', (chunk) => { logs += chunk; });
  proxy.stderr.on('data', (chunk) => { logs += chunk; });
  let spawnError;
  proxy.on('error', (error) => { spawnError = error; });
  const request = (path, headers = {}) => fetch(`http://127.0.0.1:${port}${path}`, {
    headers, signal: AbortSignal.timeout(5000),
  });
  let ready = false;
  for (let attempt = 0; attempt < 150; attempt++) {
    if (spawnError || proxy.exitCode !== null) break;
    try {
      ready = (await request('/')).status === 200;
      if (ready) break;
    } catch { /* Envoy is still starting. */ }
    await delay(100);
  }
  assert.ok(ready, `Envoy failed to start. Install Docker (Linux host networking) or set ENVOY_BIN.\n${spawnError ?? ''}\n${logs}`);

  const auth = (value = token()) => ({ cookie: `access_token=${value}` });
  const rpc = '/sparkly.backend.v1.ParkService/GetAllParks';
  await t.test('JWKS failure returns 401', async () => {
    assert.equal((await request(rpc, auth())).status, 401);
    jwksAvailable = true;
  });
  for (const path of ['/', '/auth/signin', '/auth/signup', '/_app/immutable/app.js',
    '/sparkly.backend.v1.UserService/Reg', '/sparkly.backend.v1.UserService/GetToken',
    '/api/backend/health', '/api/analytics/health', '/api/ml/health']) {
    await t.test(`public ${path} ignores invalid cookies and strips spoofed identity`, async () => {
      const response = await request(path, { ...auth('invalid'), 'x-user-id': 'attacker' });
      assert.equal(response.status, 200);
      const body = await response.json();
      assert.equal(body.service, path.startsWith('/sparkly.') ? 'backend' : 'frontend');
      assert.equal(body.user, null);
      assert.match(response.headers.get('set-cookie'), /fixture=value/);
    });
  }
  const protectedRoutes = [
    [rpc, 'backend', rpc],
    ['/sparkly.backend.v1.UserService/CheckAuth', 'backend', '/sparkly.backend.v1.UserService/CheckAuth'],
    ['/sparkly.backend.v1.UserService/Reg/extra', 'backend', '/sparkly.backend.v1.UserService/Reg/extra'],
    [`${rpc}?probe=1`, 'backend', `${rpc}?probe=1`],
    ['/sparkly.analytics.v1.AnalyticsService/GetSummary', 'analytics', '/sparkly.analytics.v1.AnalyticsService/GetSummary'],
    ['/sparkly.ml.v1.FutureService/Call', 'ml', '/sparkly.ml.v1.FutureService/Call'],
  ];
  for (const [path, service, upstreamPath] of protectedRoutes) {
    await t.test(`protected ${path}`, async () => {
      assert.equal((await request(path, { 'x-user-id': 'attacker' })).status, 401);
      const response = await request(path, { ...auth(), 'x-user-id': 'attacker' });
      assert.equal(response.status, 200);
      assert.deepEqual(await response.json(), { service, path: upstreamPath, user: 'user-123' });
    });
  }
  const invalidCredentials = [
    ['malformed', 401, auth('invalid')],
    ['forged signature', 401, auth(`${token().split('.').slice(0, 2).join('.')}.AAAA`)],
    ['expired', 401, auth(token({ exp: now() - 10 }))],
    ['not yet valid', 401, auth(token({ nbf: now() + 300 }))],
    ['missing expiration', 401, auth(token({ exp: undefined }))],
    ['missing id', 403, auth(token({ id: undefined }))],
    ['empty id', 403, auth(token({ id: '' }))],
    ['non-string id', 403, auth(token({ id: 123 }))],
    ['unsafe id', 403, auth(token({ id: 'user\r\nspoof: yes' }))],
    ['refresh token in access cookie', 403, auth(token({ type: 'refresh' }))],
    ['refresh cookie only', 401, { cookie: `refresh_token=${token()}` }],
    ['Authorization header only', 401, { authorization: `Bearer ${token()}` }],
  ];
  for (const [label, status, headers] of invalidCredentials) {
    await t.test(`${label} returns ${status}`, async () => {
      assert.equal((await request(rpc, { ...headers, 'x-user-id': 'attacker' })).status, status);
    });
  }
  await t.test('query parameter token is not accepted', async () => {
    assert.equal((await request(`${rpc}?access_token=${token()}`)).status, 401);
  });
  await t.test('unknown RPC namespace requires auth then returns 404', async () => {
    for (const path of ['/sparkly.future.v1.Service/Call']) {
      assert.equal((await request(path)).status, 401);
      assert.equal((await request(path, auth())).status, 404);
    }
  });
  await t.test('upstream 401 is preserved', async () => {
    assert.equal((await request('/sparkly.backend.v1.TestService/Unauthorized', auth())).status, 401);
  });
});
