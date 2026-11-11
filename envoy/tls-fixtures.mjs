import { execFile } from 'node:child_process';
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { promisify } from 'node:util';

const run = promisify(execFile);
export const deviceID = '01900000-0000-7000-8000-000000000001';
export const parkID = '01900000-0000-7000-8000-000000000002';
const identity = `spiffe://sparkly/parks/${parkID}/devices/${deviceID}`;

// All keys here are disposable test fixtures, never deployment credentials.
export async function createTLSFixtures(directory) {
  const path = (name, extension) => join(directory, `${name}.${extension}`);
  const openssl = (args) => run('openssl', args, { cwd: directory });
  for (const name of ['root', 'foreign-root']) {
    await openssl(['req', '-x509', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:P-256',
      '-nodes', '-keyout', path(name, 'key'), '-out', path(name, 'crt'),
      '-days', '1', '-subj', `/CN=${name}`, '-addext', 'basicConstraints=critical,CA:TRUE',
      '-addext', 'keyUsage=critical,keyCertSign,cRLSign']);
  }
  let serial = 10;
  async function issue(name, { sans, usage = 'clientAuth', issuer = 'root', days = '1' } = {}) {
    await openssl(['req', '-new', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:P-256',
      '-nodes', '-keyout', path(name, 'key'), '-out', path(name, 'csr'), '-subj', `/CN=${name}`]);
    const extensions = ['basicConstraints=critical,CA:FALSE', 'keyUsage=critical,digitalSignature',
      `extendedKeyUsage=${usage}`, ...(sans ? [`subjectAltName=${sans}`] : [])];
    await writeFile(path(name, 'ext'), extensions.join('\n'));
    await openssl(['x509', '-req', '-in', path(name, 'csr'), '-out', path(name, 'crt'),
      '-CA', path(issuer, 'crt'), '-CAkey', path(issuer, 'key'), '-set_serial', String(serial++),
      '-days', days, '-extfile', path(name, 'ext')]);
    return { cert: await readFile(path(name, 'crt')), key: await readFile(path(name, 'key')) };
  }
  const server = await issue('server', { sans: 'DNS:localhost,IP:127.0.0.1', usage: 'serverAuth' });
  const rotatedServer = await issue('server-rotated', { sans: 'DNS:localhost,IP:127.0.0.1', usage: 'serverAuth' });
  const valid = await issue('device', { sans: `URI:${identity}` });
  const foreign = await issue('foreign', { sans: `URI:${identity}`, issuer: 'foreign-root' });
  const expired = await issue('expired', { sans: `URI:${identity}`, days: '0' });
  const wrongUsage = await issue('wrong-usage', { sans: `URI:${identity}`, usage: 'serverAuth' });
  const wrongDomain = await issue('wrong-domain', { sans: `URI:${identity.replace('sparkly/', 'foreign/')}` });
  const missingSAN = await issue('missing-san');
  const multipleSANs = await issue('multiple-sans', { sans: `URI:${identity},URI:${identity}/extra` });
  const malformedID = await issue('malformed-id', { sans: `URI:${identity.replace(parkID, 'attacker')}` });
  const nilID = await issue('nil-id', { sans: `URI:${identity.replace(deviceID, '00000000-0000-0000-0000-000000000000')}` });
  return { ca: await readFile(path('root', 'crt')), server, rotatedServer, valid, foreign, expired,
    wrongUsage, wrongDomain, missingSAN, multipleSANs, malformedID, nilID };
}
