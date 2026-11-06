# Device certificate enrollment

`POST /devices/reg` now accepts `park_token` and `csr_pem`, and returns a client certificate chain instead of `device_token`. This is a breaking change for enrollment clients. The device creates and retains its private key. The backend hashes the park enrollment token, looks up the park, assigns a device UUID, and authorizes step-ca issuance using a dedicated ES256 JWK provisioner.

The issued URI SAN is:

```text
spiffe://sparkly/parks/<park UUID>/devices/<device UUID>
```

The common name is the device UUID. URI SANs carry the authorization identity; the common name is only a label. Certificates have `digitalSignature` and `clientAuth` usages and cannot act as CAs. The template reads only backend-signed JWT claims and never copies CSR identities or extensions. The JWT is short lived, single use, and bound to the CSR fingerprint. The backend validates the returned certificate chain, public key, identity and client profile before committing registration.

## Run with Docker Compose

From the repository root:

```sh
docker compose --profile backend up --build -d
```

Compose initializes step-ca automatically on first boot, installs the device provisioner and template, and waits for CA health before starting the backend. No host `step` installation or manual key ID configuration is needed. The CA has only the `sparkly-devices` online provisioner; the generic initialization provisioner is removed before the server starts.

| Setting                                          | Default                                                        |
| ------------------------------------------------ | -------------------------------------------------------------- |
| Host HTTPS endpoint                              | `https://localhost:9001` (`STEP_CA_PORT`)                      |
| Backend HTTPS endpoint                           | `https://step-ca:9000` over Compose networking                 |
| CA name                                          | `Sparkly Device CA` (`STEP_CA_NAME`)                           |
| CA DNS names                                     | `step-ca,localhost,host.docker.internal` (`STEP_CA_DNS_NAMES`) |
| Device certificate lifetime                      | 365 days (`8760h`)                                             |
| Root CA lifetime on first initialization         | 10 × 365 days (`87600h`)                                       |
| Intermediate CA lifetime on first initialization | 5 × 365 days (`43800h`)                                        |

Host port 9001 avoids ClickHouse's native port 9000. Include `step-ca` and `localhost` when overriding CA DNS names. CA name and DNS names are initialization settings; editing them does not replace an existing CA's identity.

The `step-ca-data` named volume holds the CA certificates, encrypted signing keys, randomly generated unlock password, issuance database, and persistent device provisioner key. Back it up securely. The separate `device-pki` volume exports only `root_ca.crt`, `provisioner.key`, and `provisioner.kid`; the backend mounts it read-only. Exported files are readable by the backend's nonroot user. They contain a provisioner secret, so mount this volume only into services authorized to issue device certificates. The separate `envoy-pki` volume contains the gateway server certificate/key and device trust root; Envoy never mounts `device-pki` or the CA storage. The CA unlock password is generated once and is not printed or placed in Compose environment variables.

Ordinary restarts preserve the root and provisioner. Rebuilding/restarting step-ca applies template changes. `docker compose down` preserves volumes; `docker compose down -v` destroys this CA and its credentials along with the other project volumes. Certificates from a destroyed CA will not be trusted by a replacement CA.

Rebuilding/restarting step-ca also applies the one-year device issuance policy to an existing CA. Previously issued certificates retain their original expiry. Existing CA certificates are never replaced automatically: the explicit root and intermediate lifetimes apply only to a fresh initialization. Preserve an existing deployment's trust root and use a planned intermediate rotation to change its CA lifecycle.

Begin rotating a five-year intermediate before its final year, allowing operational margin for one-year device certificates. Keep the previous issuer's certificate and CRL-signing capability until its outstanding certificates expire. Production deployments should provision the root offline and import the intermediate; this development bootstrap keeps both CA keys in its persistent volume.

To run the backend on the host, export its three files after CA startup:

```sh
mkdir -p step-ca/local
chmod 700 step-ca/local
docker compose cp step-ca:/run/device-pki/root_ca.crt step-ca/local/root_ca.crt
docker compose cp step-ca:/run/device-pki/provisioner.key step-ca/local/provisioner.key
docker compose cp step-ca:/run/device-pki/provisioner.kid step-ca/local/provisioner.kid
chmod 600 step-ca/local/provisioner.key
export STEP_CA_URL=https://localhost:9001
export STEP_CA_PROVISIONER=sparkly-devices
export STEP_CA_ROOT_FILE="$PWD/step-ca/local/root_ca.crt"
export STEP_CA_KEY_FILE="$PWD/step-ca/local/provisioner.key"
export STEP_CA_KEY_ID_FILE="$PWD/step-ca/local/provisioner.kid"
```

`step-ca/local/` is ignored by Git. Export `STEP_CA_KEY_ID` directly instead of `STEP_CA_KEY_ID_FILE` if preferred; the direct value takes precedence. Host processes can use a separately managed CA with the same `STEP_CA_*` settings. The backend requires an HTTPS origin, a trusted root bundle, a provisioner name and key ID, and an unencrypted P-256 provisioner private key in PEM format.

For local Envoy HTTPS, startup also signs a 30-day server certificate directly with the intermediate CA and exports it to `envoy-pki`. It is reissued on CA startup; restart step-ca before its expiry in long-running local environments. See [Envoy TLS setup](../envoy/README.md).

The container uses the official [Smallstep image](https://smallstep.com/docs/tutorials/docker-tls-certificate-authority/), with a custom bootstrap script and [JWK provisioner template](https://smallstep.com/docs/step-ca/templates/). It runs as the image's `step` user. Compose is a development deployment: the CA keys and their unlock password share persistent storage; use a separate offline root/key-management setup for production.

## Download the public root certificate

The public root CA bundle is available from Envoy at `http://localhost:10000/.well-known/root-ca.crt` and `https://localhost:10002/.well-known/root-ca.crt`. No enrollment token or device certificate is required. See [root CA download and fingerprint verification](../envoy/README.md#root-ca-download) before using it as a trust anchor.

## Enroll a device

The CSR must have an empty common name and no SANs: the device does not yet know its assigned identity. Supported keys are ECDSA P-256/P-384/P-521, Ed25519, and RSA >= 2048 bits. For a local smoke test:

```sh
openssl ecparam -name prime256v1 -genkey -noout -out device.key
openssl req -new -key device.key -subj / -out device.csr
jq -n --arg park_token "$PARK_TOKEN" --rawfile csr_pem device.csr \
  '{park_token: $park_token, csr_pem: $csr_pem}' > enrollment.json
curl --fail-with-body --cacert step-ca/local/root_ca.crt \
  -H 'Content-Type: application/json' --data-binary @enrollment.json \
  https://localhost:10002/devices/reg > enrollment-response.json
jq -r .certificate_chain_pem enrollment-response.json > device-chain.crt
```

Enrollment uses Envoy's server-authenticated HTTPS listener on port 10002 without requiring an existing client certificate. The plaintext application listener does not expose enrollment. Provision the enrollment server's trust anchor on the device beforehand; never obtain trust by accepting an unverified response. Remove the request file containing the enrollment token after the test.

A successful response is HTTP 201:

```json
{
  "device_id": "01900000-0000-7000-8000-000000000001",
  "park_id": "01900000-0000-7000-8000-000000000002",
  "identity_uri": "spiffe://sparkly/parks/01900000-0000-7000-8000-000000000002/devices/01900000-0000-7000-8000-000000000001",
  "certificate_chain_pem": "-----BEGIN CERTIFICATE-----\n...",
  "expires_at": "2027-11-06T00:00:00Z"
}
```

Malformed requests return 400, an invalid enrollment token returns 401, and CA failure returns 503. Request bodies are limited to 32 KiB. The existing park token remains reusable until rotated. This change does not make it a one-time token or authenticate previously enrolled devices automatically; existing devices must enroll again. The legacy `devices.token_hash` column remains for schema compatibility, with empty values for new registrations, and is never used for authentication.

## Envoy mTLS termination

Envoy's device listener on port 10001 requires a trusted client certificate, validates its URI SAN, and forwards `x-device-id` and `x-park-id` to analytics for `POST /ingest`. A separate HTTPS listener on port 10002 exposes initial enrollment. Both listeners strip client-supplied identity headers. See [gateway authentication and Kubernetes deployment requirements](../envoy/README.md) for the trust boundary, certificate mounts, rotation, and allowed routes.

## Lifecycle and failure handling

Certificate renewal and revocation are not implemented here. Add renewal before deploying long-running devices: authenticate with the current valid device certificate, retain the same device/park identity, and support key rotation. Never use the enrollment token as the renewal credential or register a fresh device on every renewal. Devices should renew well before the one-year default expiry, with enough time for their expected offline periods.

Deleting a database row or rotating a park token does not revoke an issued certificate. An Envoy-only chain check will continue accepting it until expiry unless a revocation/authorization mechanism is added. With one-year certificates, implement CRL enforcement or online device authorization before relying on device disablement; changing database state alone does not invalidate an issued certificate.

Signing occurs inside the database transaction. CA failure rolls registration back. A CA-issued certificate cannot be rolled back if the database commit or HTTP response later fails. Clients must treat a failed enrollment as incomplete; retries currently create a new device ID. Before production, add enrollment idempotency and reconciliation/audit of orphaned issuance if this failure window matters for your deployment.
