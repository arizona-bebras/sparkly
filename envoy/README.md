# Gateway authentication

Envoy terminates device TLS and forwards device identity to the allowed upstream endpoint. User JWT authentication remains on the existing application listener.

| Listener    | Default host port | Authentication                                               | Allowed device routes                                              |
| ----------- | ----------------- | ------------------------------------------------------------ | ------------------------------------------------------------------ |
| Application | 10000, HTTP       | Existing user JWT rules; public CA download                  | `GET/HEAD /.well-known/root-ca.crt`                                |
| Devices     | 10001, HTTPS      | Required device client certificate                           | `POST /ingest` → analytics                                         |
| Enrollment  | 10002, HTTPS      | Park enrollment token checked by backend; public CA download | `POST /devices/reg` → backend; `GET/HEAD /.well-known/root-ca.crt` |

Set `ENVOY_PORT`, `ENVOY_DEVICE_PORT`, or `ENVOY_ENROLLMENT_PORT` to change host ports. The TLS listeners accept HTTP/1.1 and HTTP/2 and require TLS 1.2 or newer. Paths and methods are explicitly allowed; other device/enrollment requests return 404. Plaintext application ingress also returns 404 for `/ingest` and `/devices/reg`.

## Root CA download

Envoy serves `GET /.well-known/root-ca.crt` directly from `/run/envoy-pki/root_ca.crt`, with PEM content type and a download filename of `root_ca.crt`. `HEAD` is also supported; other methods return 405. It is available through both the application HTTP listener and enrollment HTTPS listener.

```sh
curl --fail --output root_ca.crt http://localhost:10000/.well-known/root-ca.crt
openssl x509 -in root_ca.crt -noout -fingerprint -sha256
```

Compare that fingerprint with a trusted value supplied separately (for example, firmware configuration or an administrator) before trusting the downloaded certificate. Downloading a root certificate does not establish its authenticity. After verification, use `--cacert root_ca.crt` for local enrollment and device requests. In Kubernetes, the gateway server may use a different CA; clients must trust the issuer of its server certificate.

The direct-response body is loaded when Envoy loads its route configuration and supports a CA bundle up to 64 KiB. After changing the public root bundle, restart/reload Envoy to refresh this download; TLS SDS updates are handled independently. Only this exact file is exposed, not the mounted directory or any private keys.

## Device identity

The device listener requires a certificate chain trusted by `root_ca.crt`. BoringSSL validates the client certificate's chain, lifetime and TLS client usage. `device_identity.lua` then requires exactly one URI SAN of the form:

```text
spiffe://sparkly/parks/<park UUID>/devices/<device UUID>
```

UUIDs must be canonical lowercase, nonzero values. The filter rejects missing or malformed identities, extra URI SANs and other trust domains with HTTP 403. It reads identity directly from the verified TLS connection, independent of client-supplied HTTP headers or certificate common names.

Envoy removes inbound `x-device-id`, `x-park-id`, `x-user-id`, and `x-forwarded-client-cert`. On the device listener, it sets `x-device-id` and `x-park-id` from the certificate. The application and enrollment listeners strip device identity headers too; a user JWT does not grant access to the device listener.

TLS session resumption is disabled on the device listener because Envoy's Lua `peerCertificateValidated()` reports false on resumed sessions. See the [Envoy Lua TLS API](https://www.envoyproxy.io/docs/envoy/v1.40.0/configuration/http/http_filters/lua_filter#peercertificatevalidated).

The analytics ingest handler requires both forwarded identity headers and stores them in `raw_batches.device_id` and `raw_batches.park_id`. It rejects missing, invalid, or duplicate values before writing a batch. The gateway verifies device/park membership through the signed certificate; analytics relies on that trusted identity and must be protected from direct untrusted requests.

## Local Compose credentials

```sh
docker compose --profile backend --profile analytics up --build -d
mkdir -p step-ca/local
docker compose cp step-ca:/run/envoy-pki/root_ca.crt step-ca/local/root_ca.crt
```

The CA provisions a local server certificate for `localhost`, `envoy`, `host.docker.internal`, and `127.0.0.1`. The separate `envoy-pki` volume contains only:

- `server-chain.crt`: Envoy's server certificate and intermediate chain.
- `server.key`: Envoy's server private key.
- `root_ca.crt`: Device certificate trust bundle.

Envoy mounts this volume read-only. It never receives the backend's device provisioner key or the CA private keys. The local server certificate lasts 30 days and is reissued when step-ca starts; restart step-ca before its expiry for long-running local environments. Kubernetes should supply renewed server credentials through its certificate-management system.

Enroll through `https://localhost:10002/devices/reg`, trusting the copied root certificate. See [device enrollment](../step-ca/README.md) for CSR creation and response handling. With the returned device certificate chain and the private key retained by the device, send a binary parcel:

```sh
curl --fail-with-body --cacert step-ca/local/root_ca.crt \
  --cert device-chain.crt --key device.key \
  -H 'Content-Type: application/octet-stream' --data-binary @parcel.bin \
  https://localhost:10001/ingest
```

## Kubernetes deployment contract

Expose the TLS listeners through a TCP Service/load balancer so the device's TLS connection reaches Envoy. Keep enrollment available without an existing client certificate. Do not terminate device TLS at another ingress and pass identity in unauthenticated headers.

Mount Envoy configuration and `device_identity.lua` at their configured `/etc/envoy` paths. Supply a projected directory at `/run/envoy-pki` containing a server certificate/key (for example, a cert-manager-managed Secret) and the device CA trust bundle. Map Secret keys such as `tls.crt` and `tls.key` to `server-chain.crt` and `server.key`. The server certificate must cover the hostname used by devices; its issuer may differ from the device CA.

Both TLS certificate and device trust are loaded through filesystem SDS. The secret definitions watch `/run/envoy-pki` for atomic replacement, including projected-volume updates. Mount the directory rather than individual files with `subPath`, and ensure Envoy's user can read it. See [Envoy SDS rotation](https://www.envoyproxy.io/docs/envoy/v1.40.0/configuration/security/secret#key-rotation).

Upstreams must accept connections only from Envoy, enforced with Kubernetes NetworkPolicy and private Services or authenticated upstream TLS. Published Compose backend/analytics ports are for development and allow bypassing gateway authentication. Upstreams must use the trusted park identity when authorizing access to resources. Certificate chain validation does not check whether a device was deleted from the application database; immediate device disablement requires an authorization/revocation mechanism.

## Verification

Run `node --test envoy/proxy.test.mjs` with OpenSSL and Docker available (Linux host networking), or set `ENVOY_BIN` to a local Envoy binary. The suite uses disposable certificate authorities and upstream servers. It covers existing JWT rules, TLS authentication failures, identity validation and spoofing, route restrictions, and live SDS certificate replacement.
