import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { spawn } from 'node:child_process';
import { chmod, mkdtemp, readFile, rename, rm, writeFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { request as httpsRequest } from 'node:https';
import { createTLSFixtures, deviceID, parkID } from './tls-fixtures.mjs';
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
const tokenID = '01930231-1234-7000-8000-000000000001';

function token(overrides = {}) {
  const header = Buffer.from(JSON.stringify({ alg: 'RS256', kid: jwk.kid })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    id: 'user-123', type: 'access', exp: now() + 300, ...overrides,
  })).toString('base64url');
  const input = `${header}.${payload}`;
  return `${input}.${sign('RSA-SHA256', Buffer.from(input), privateKey).toString('base64url')}`;
}

const refreshToken = (overrides = {}) => token({
  id: undefined, type: 'refresh', token_id: tokenID, user_id: 'user-123', ...overrides,
});

async function listen(server) {
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '0.0.0.0', resolve);
  });
  return server.address().port;
}

test('Envoy routing, JWT authentication, and device mTLS', { timeout: 90000 }, async (t) => {
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

  const tls = await createTLSFixtures(temporary);
  const runtimeDirectory = process.env.ENVOY_BIN ? temporary : '/test';
  await chmod(temporary, 0o755);
  for (const [name, content] of Object.entries({
    'server-chain.crt': tls.server.cert,
    'server.key': tls.server.key,
    'root_ca.crt': tls.ca,
    'device_identity.lua': await readFile(join(directory, 'device_identity.lua')),
  })) {
    await writeFile(join(temporary, name), content, { mode: 0o644 });
    await chmod(join(temporary, name), 0o644);
  }
  for (const name of ['server-secret.yaml', 'device-trust-secret.yaml']) {
    const secret = (await readFile(join(directory, name), 'utf8')).replaceAll('/run/envoy-pki', runtimeDirectory);
    await writeFile(join(temporary, name), secret, { mode: 0o644 });
  }
  let config = (await readFile(join(directory, 'envoy.yaml'), 'utf8'))
    .replaceAll('/etc/envoy/', `${runtimeDirectory}/`)
    .replaceAll('/run/envoy-pki', runtimeDirectory);
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
      res.end(JSON.stringify({ service, path: req.url, user: req.headers['x-user-id'] ?? null,
        token: req.headers['x-token-id'] ?? null,
        device: req.headers['x-device-id'] ?? null, park: req.headers['x-park-id'] ?? null,
        forwardedCertificate: req.headers['x-forwarded-client-cert'] ?? null }));
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
  async function availablePort() {
    const reservation = createServer();
    const port = await listen(reservation);
    await new Promise((resolve) => reservation.close(resolve));
    return port;
  }
  const devicePort = await availablePort();
  const enrollmentPort = await availablePort();
  config = config.replace('port_value: 10001', `port_value: ${devicePort}`)
    .replace('port_value: 10002', `port_value: ${enrollmentPort}`);
  const configPath = join(temporary, 'envoy.yaml');
  await writeFile(configPath, config);
  await chmod(configPath, 0o644);
  const args = ['-c', process.env.ENVOY_BIN ? configPath : '/test/envoy.yaml',
    '--concurrency', '1', '--log-level', 'warn'];
  proxy = process.env.ENVOY_BIN
    ? spawn(process.env.ENVOY_BIN, args)
    : spawn('docker', ['run', '--rm', '--network', 'host',
      '--add-host', 'host.docker.internal:host-gateway',
      '-v', `${temporary}:/test:ro`, image, ...args]);
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
  await t.test('root CA download is public even when JWKS is unavailable', async () => {
    const response = await request('/.well-known/root-ca.crt', {
      cookie: 'access_token=invalid; refresh_token=invalid',
    });
    assert.equal(response.status, 200);
    assert.equal(response.headers.get('content-type'), 'application/x-pem-file');
    assert.equal(await response.text(), tls.ca.toString());
    assert.equal(response.headers.get('content-disposition'), 'attachment; filename="root_ca.crt"');
    const head = await fetch(`http://127.0.0.1:${port}/.well-known/root-ca.crt`, { method: 'HEAD' });
    assert.equal(head.status, 200);
    assert.equal(await head.text(), '');
    const post = await fetch(`http://127.0.0.1:${port}/.well-known/root-ca.crt`, { method: 'POST' });
    assert.equal(post.status, 405);
    assert.equal(post.headers.get('allow'), 'GET, HEAD');
  });
  await t.test('JWKS failure denies backend access', async () => {
    assert.equal((await request(rpc, auth())).status, 401);
    assert.equal((await request(rpc, { cookie: `refresh_token=${refreshToken()}` })).status, 401);
    jwksAvailable = true;
  });
  for (const path of ['/', '/auth/signin', '/auth/signup', '/_app/immutable/app.js',
    '/sparkly.backend.v1.UserService/Reg', '/sparkly.backend.v1.UserService/GetToken',
    '/api/backend/health', '/api/analytics/health', '/api/ml/health']) {
    for (const cookie of ['access_token=invalid; refresh_token=invalid',
      `access_token=${token()}; refresh_token=${refreshToken()}`]) {
      await t.test(`public ${path} ignores cookies and strips spoofed identity`, async () => {
        const response = await request(path, {
          cookie, 'x-user-id': 'attacker', 'x-token-id': 'attacker',
          'x-device-id': 'attacker', 'x-park-id': 'attacker', 'x-forwarded-client-cert': 'attacker',
        });
        assert.equal(response.status, 200);
        const body = await response.json();
        assert.equal(body.service, path.startsWith('/sparkly.') ? 'backend' : 'frontend');
        assert.equal(body.user, null);
        assert.equal(body.token, null);
        assert.equal(body.device, null);
        assert.equal(body.park, null);
        assert.equal(body.forwardedCertificate, null);
        assert.match(response.headers.get('set-cookie'), /fixture=value/);
      });
    }
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
      assert.equal((await request(path, { 'x-user-id': 'attacker', 'x-token-id': 'attacker' })).status, 401);
      const response = await request(path, { ...auth(), 'x-user-id': 'attacker', 'x-token-id': 'attacker',
        'x-device-id': 'attacker', 'x-park-id': 'attacker', 'x-forwarded-client-cert': 'attacker' });
      assert.equal(response.status, 200);
      assert.deepEqual(await response.json(), { service, path: upstreamPath, user: 'user-123', token: null, device: null, park: null, forwardedCertificate: null });
    });
  }
  const invalidCredentials = [
    ['malformed', auth('invalid')],
    ['both cookies malformed', { cookie: 'access_token=invalid; refresh_token=invalid' }],
    ['forged signature', auth(`${token().split('.').slice(0, 2).join('.')}.AAAA`)],
    ['expired', auth(token({ exp: now() - 10 }))],
    ['not yet valid', auth(token({ nbf: now() + 300 }))],
    ['missing expiration', auth(token({ exp: undefined }))],
    ['missing id', auth(token({ id: undefined }))],
    ['empty id', auth(token({ id: '' }))],
    ['non-string id', auth(token({ id: 123 }))],
    ['unsafe id', auth(token({ id: 'user\r\nspoof: yes' }))],
    ['refresh token in access cookie', auth(token({ type: 'refresh' }))],
    ['access token in refresh cookie', { cookie: `refresh_token=${token()}` }],
    ['Authorization header only', { authorization: `Bearer ${token()}` }],
  ];
  for (const [label, headers] of invalidCredentials) {
    await t.test(`${label} returns 401`, async () => {
      const response = await request(rpc, {
        ...headers, 'x-user-id': 'attacker', 'x-token-id': 'attacker',
      });
      assert.equal(response.status, 401);
      assert.equal(response.headers.get('www-authenticate'), 'Bearer');
    });
  }
  await t.test('query parameter token is not accepted', async () => {
    assert.equal((await request(`${rpc}?access_token=${token()}`)).status, 401);
  });
  await t.test('gRPC receives Unauthenticated for a missing backend session', async () => {
    const response = await fetch(`http://127.0.0.1:${port}${rpc}`, {
      method: 'POST',
      headers: { 'content-type': 'application/grpc' },
      signal: AbortSignal.timeout(5000),
    });
    assert.equal(response.status, 200);
    assert.equal(response.headers.get('grpc-status'), '16');
  });
  for (const [label, access] of [
    ['absent', undefined], ['expired', token({ exp: now() - 10 })], ['malformed', 'invalid'],
    ['wrong type', token({ type: 'refresh' })], ['empty id', token({ id: '' })],
    ['non-string id', token({ id: 123 })], ['unsafe id', token({ id: 'user\r\nspoof: yes' })],
  ]) {
    await t.test(`refresh fallback with ${label} access cookie`, async () => {
      const response = await request(rpc, {
        cookie: `${access === undefined ? '' : `access_token=${access}; `}refresh_token=${refreshToken()}`,
        'x-user-id': 'attacker', 'x-token-id': 'attacker',
      });
      assert.equal(response.status, 200);
      assert.deepEqual(await response.json(), { service: 'backend', path: rpc, user: null, token: tokenID, device: null, park: null, forwardedCertificate: null });
      assert.match(response.headers.get('set-cookie'), /fixture=value/);
    });
  }
  await t.test('both valid cookies forward user and session for revocation', async () => {
    const path = '/sparkly.backend.v1.UserService/RevokeToken';
    const response = await request(path, {
      cookie: `access_token=${token()}; refresh_token=${refreshToken()}`,
      'x-user-id': 'attacker', 'x-token-id': 'attacker',
    });
    assert.equal(response.status, 200);
    assert.deepEqual(await response.json(), { service: 'backend', path, user: 'user-123', token: tokenID, device: null, park: null, forwardedCertificate: null });
  });
  for (const invalid of ['invalid', `${refreshToken().split('.').slice(0, 2).join('.')}.AAAA`,
    refreshToken({ exp: now() - 10 }), refreshToken({ nbf: now() + 300 }),
    refreshToken({ exp: undefined }), refreshToken({ type: 'access' }),
    refreshToken({ token_id: undefined }), refreshToken({ token_id: '' }),
    refreshToken({ token_id: 123 }), refreshToken({ token_id: 'session\r\nspoof: yes' })]) {
    await t.test('invalid refresh cannot authorize or supply a session header', async () => {
      const cookie = `refresh_token=${invalid}`;
      assert.equal((await request(rpc, {
        cookie: `access_token=${token({ exp: now() - 10 })}; ${cookie}`,
        'x-user-id': 'attacker', 'x-token-id': 'attacker',
      })).status, 401);
      const response = await request(rpc, { cookie: `access_token=${token()}; ${cookie}` });
      assert.equal(response.status, 200);
      assert.equal((await response.json()).token, null);
    });
  }
  for (const path of ['/sparkly.analytics.v1.AnalyticsService/GetSummary', '/sparkly.ml.v1.FutureService/Call']) {
    await t.test(`refresh cookie cannot authenticate ${path}`, async () => {
      const response = await request(path, {
        cookie: `access_token=${token({ exp: now() - 10 })}; refresh_token=${refreshToken()}`,
      });
      assert.equal(response.status, 401);
    });
  }
  await t.test('unknown RPC namespace requires auth then returns 404', async () => {
    for (const path of ['/sparkly.future.v1.Service/Call']) {
      assert.equal((await request(path)).status, 401);
      assert.equal((await request(path, auth())).status, 404);
    }
  });
  await t.test('upstream 401 is preserved', async () => {
    assert.equal((await request('/sparkly.backend.v1.TestService/Unauthorized', auth())).status, 401);
  });

  function secureRequest(port, path, { credentials, method = 'POST', headers = {} } = {}) {
    return new Promise((resolve, reject) => {
      const req = httpsRequest({ hostname: '127.0.0.1', servername: 'localhost', port, path, method,
        ca: tls.ca, cert: credentials?.cert, key: credentials?.key, headers, agent: false }, (res) => {
        const serial = res.socket.getPeerCertificate().serialNumber;
        let body = '';
        res.setEncoding('utf8');
        res.on('data', (chunk) => { body += chunk; });
        res.on('end', () => resolve({ status: res.statusCode, body, serial }));
      });
      req.setTimeout(5000, () => req.destroy(new Error('HTTPS request timed out')));
      req.on('error', reject);
      req.end();
    });
  }
  await t.test('root CA download is available on enrollment HTTPS without a client certificate', async () => {
    const response = await secureRequest(enrollmentPort, '/.well-known/root-ca.crt', { method: 'GET' });
    assert.equal(response.status, 200);
    assert.equal(response.body, tls.ca.toString());
    assert.equal((await secureRequest(enrollmentPort, '/.well-known/root-ca.crt', { method: 'HEAD' })).body, '');
    assert.equal((await secureRequest(enrollmentPort, '/.well-known/root-ca.crt')).status, 405);
  });
  await t.test('enrollment uses HTTPS without an existing device certificate', async () => {
    const response = await secureRequest(enrollmentPort, '/devices/reg', { headers: {
      'x-device-id': 'attacker', 'x-park-id': 'attacker', 'x-user-id': 'attacker',
      'x-forwarded-client-cert': 'attacker',
    } });
    assert.equal(response.status, 200);
    assert.deepEqual(JSON.parse(response.body), { service: 'backend', path: '/devices/reg',
      user: null, token: null, device: null, park: null, forwardedCertificate: null });
    assert.equal((await secureRequest(enrollmentPort, '/ingest')).status, 404);
    assert.equal((await secureRequest(enrollmentPort, '/devices/reg', { method: 'GET' })).status, 404);
  });
  await t.test('device endpoints are not available through the plaintext listener', async () => {
    assert.equal((await request('/devices/reg')).status, 404);
    assert.equal((await request('/ingest', { 'x-device-id': deviceID, 'x-park-id': parkID })).status, 404);
  });
  await t.test('mTLS forwards only certificate-derived identity', async () => {
    const response = await secureRequest(devicePort, '/ingest', { credentials: tls.valid, headers: {
      cookie: `access_token=${token()}`, 'x-device-id': 'attacker', 'x-park-id': 'attacker',
      'x-user-id': 'attacker', 'x-forwarded-client-cert': 'attacker',
    } });
    assert.equal(response.status, 200);
    assert.deepEqual(JSON.parse(response.body), { service: 'analytics', path: '/ingest',
      device: deviceID, park: parkID, user: null, token: null, forwardedCertificate: null });
  });
  for (const name of ['foreign', 'expired', 'wrongUsage']) {
    await t.test(`TLS rejects ${name} client certificate`, async () => {
      await assert.rejects(secureRequest(devicePort, '/ingest', { credentials: tls[name] }));
    });
  }
  await t.test('a user token cannot replace the device certificate', async () => {
    await assert.rejects(secureRequest(devicePort, '/ingest', { headers: auth() }));
  });
  for (const name of ['wrongDomain', 'missingSAN', 'multipleSANs', 'malformedID', 'nilID']) {
    await t.test(`identity validation rejects ${name}`, async () => {
      assert.equal((await secureRequest(devicePort, '/ingest', { credentials: tls[name] })).status, 403);
    });
  }
  await t.test('device credentials do not grant user or enrollment access', async () => {
    for (const path of [rpc, '/devices/reg', '/unknown']) {
      assert.equal((await secureRequest(devicePort, path, { credentials: tls.valid })).status, 404);
    }
    assert.equal((await secureRequest(devicePort, '/ingest', { credentials: tls.valid, method: 'GET' })).status, 404);
  });
  await t.test('SDS reloads rotated server credentials without restarting Envoy', async () => {
    const before = await secureRequest(devicePort, '/ingest', { credentials: tls.valid });
    for (const [name, contents] of [['server.key', tls.rotatedServer.key], ['server-chain.crt', tls.rotatedServer.cert]]) {
      await writeFile(join(temporary, `${name}.next`), contents, { mode: 0o644 });
      await rename(join(temporary, `${name}.next`), join(temporary, name));
    }
    let after;
    for (let attempt = 0; attempt < 50; attempt++) {
      after = await secureRequest(devicePort, '/ingest', { credentials: tls.valid });
      if (after.serial !== before.serial) break;
      await delay(100);
    }
    assert.equal(after.status, 200);
    assert.notEqual(after.serial, before.serial, 'Envoy kept serving the old certificate');
  });
});
