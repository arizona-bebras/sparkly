# Sparkly Helm chart

Deploys Envoy, frontend, backend, and analytics. PostgreSQL, ClickHouse, RabbitMQ,
step-ca, ingress controllers, and certificate controllers are supplied by the
cluster. The chart does not create databases, broker vhosts, operator resources,
CA signing keys, or credential Secrets.

| Public endpoint (443) | Routing | TLS termination |
| --- | --- | --- |
| `sparkly.run` | Traefik HTTP Ingress → Envoy 10000 | Traefik, web TLS Secret |
| `ingest.devices.sparkly.run` | Traefik SNI TCP route → Envoy 10001 | Envoy, required device mTLS |
| `enroll.devices.sparkly.run` | Traefik SNI TCP route → Envoy 10002 | Envoy, server TLS only |

The device and enrollment listeners share a server certificate with both DNS
SANs. They have separate ports and authentication policies. TLS passes through
Traefik unchanged, preserving the client's certificate for Envoy's URI SAN
validation. No HTTP identity forwarding from Traefik is used. Existing JWT,
identity header stripping, device route restrictions, and public root download
behavior are preserved from the repository gateway configuration.

## Cluster prerequisites

- Kubernetes 1.26+, Helm 3 or 4, and a NetworkPolicy-capable CNI.
- Traefik with its Kubernetes Ingress and CRD providers, `traefik.io/v1alpha1`
  CRDs installed, and the `websecure` entrypoint exposed on TCP 443. Set
  `routing.ingressClassName` to its configured class and `routing.entryPoint` to
  its TLS entrypoint. The HTTP and TCP routes use the same entrypoint.
- DNS for all three names pointing to the Traefik load balancer.
- cert-manager, Smallstep step-issuer, and a ready `StepClusterIssuer` named
  `sparkly-step-ca`. Override `certificates.issuerRef` for another issuer (a
  namespaced `StepIssuer` must be in the release namespace). Configure request
  approval for the external `certmanager.step.sm` signer in your installation.
- An existing step-ca with a dedicated TLS provisioner for step-issuer, authorized
  to issue the three DNS names, ECDSA keys, and 24-hour server certificates.
  The backend needs a **separate** `sparkly-devices` JWK provisioner using
  [the device template and issuance policy](../../step-ca/README.md). Do not use
  the device-only provisioner for server certificate requests. This chart uses
  your existing CA; it does not run the Compose development bootstrap.
- Operator-provided databases/users and a RabbitMQ vhost with permissions to
  declare the application's exchanges and queues. Backend and analytics perform
  their schema migrations at startup; grant the required DDL privileges (including
  PostgreSQL `pg_trgm`). Database names and broker vhosts must exist first.

Traefik passthrough configuration follows the
[IngressRouteTCP reference](https://doc.traefik.io/traefik/reference/routing-configuration/kubernetes/crd/tcp/ingressroutetcp/).
For issuer provisioning and the `Certificate` → Secret workflow, see the
[Smallstep step-issuer instructions](https://github.com/smallstep/step-issuer).

## Required Secrets

Create these in the release namespace or override their names/keys in
[values.yaml](values.yaml). Secrets are referenced, never generated or stored in
Helm values. Operator-generated Secrets can be used directly if they expose the
needed keys. For example, CloudNativePG application credentials commonly have
`username`/`password`; configure the actual operator-generated service and Secret
names. A RabbitMQ default-user Secret's separate credentials are insufficient:
provide a Secret containing a complete URI.

| Default Secret | Required keys | Consumer |
| --- | --- | --- |
| `sparkly-postgres-app` | `username`, `password` | backend |
| `sparkly-clickhouse-app` | `username`, `password` | analytics |
| `sparkly-rabbitmq-app` | `uri` | backend, analytics, optional symbolicator |
| `sparkly-backend` | `JWKS_MASTER_KEY` | backend |
| `sparkly-device-provisioner` | `provisioner.key`, `provisioner.kid` | backend only |
| `sparkly-device-trust` | `root_ca.crt` | backend and Envoy |

`uri` is a complete `amqp://user:password@service.namespace.svc:5672/vhost` (or
`amqps://...`) connection string; percent-encode credentials and the vhost.
`JWKS_MASTER_KEY` must be a stable, high-entropy value: changing it can prevent
reading existing signing keys. `provisioner.key` must be an unencrypted P-256 PEM
private key; `provisioner.kid` is its provisioner key ID. `root_ca.crt` is the
trusted PEM root bundle for the **device CA**, also used by backend to connect to
step-ca and validate issued device chains. It is independent of the server TLS
issuer's root and is not inferred from a server Secret's `ca.crt`.

For example, import provisioner and trust files obtained securely from your CA
administrator (the chart never needs the root/intermediate private keys):

```sh
kubectl create namespace sparkly
kubectl -n sparkly create secret generic sparkly-device-provisioner \
  --from-file=provisioner.key=/secure/device-provisioner.key \
  --from-file=provisioner.kid=/secure/device-provisioner.kid
kubectl -n sparkly create secret generic sparkly-device-trust \
  --from-file=root_ca.crt=/secure/device-root-ca.crt
```

cert-manager creates and renews `sparkly-web-tls` and `sparkly-devices-tls`, each
with `tls.crt`/`tls.key`, using the configured step issuer. The chart requests
24-hour certificates with renewal 8 hours before expiry and fresh private keys.
With `certificates.enabled=false`, provision both Secrets through your external
certificate controller instead. Server clients and browsers must already trust
the step-ca TLS issuer's root; private step-ca certificates are not publicly
trusted by default.

## Build and install

The [release workflow](../../.github/workflows/release.yml) builds and publishes
images and an OCI Helm chart to GHCR. Pushes to `main` append
`-main.<workflow run number>` to the version in `Chart.yaml`, and stable
`vMAJOR.MINOR.PATCH` tags publish production releases. Both use the same OCI Helm chart repository
and image repositories. The workflow publishes all supported images first,
then packages the chart with `version` and `appVersion` matching the image tags.
Empty component image tags inherit `appVersion`, so selecting a chart release
selects its matching images.

For example, Git tag `v0.2.0` publishes:

- Images: `ghcr.io/arizona-bebras/sparkly/{envoy,frontend,backend,analytics,ml}:0.2.0`
- Chart: `oci://ghcr.io/arizona-bebras/sparkly/charts/sparkly:0.2.0`

The default registry prefix is `global.imageRegistry`; the release workflow sets
it from the GitHub repository name in a temporary packaging directory.
Component `image.repository` values are relative to that prefix.
Explicit component tags remain available for special overrides.
The Envoy image contains its configuration, Lua filter, and SDS definitions;
the chart only mounts certificate Secrets. Argo CD Applications and environment
values live in the separate GitOps repository.

Services use the fixed namespace-local names `envoy`, `frontend`, `backend`,
`analytics`, and (when enabled) `ml`. Envoy uses these hostnames directly.
Install one Sparkly release per namespace; use separate namespaces for multiple
installations. `fullnameOverride` changes workload names without changing these
Service names.

```yaml
# production-values.yaml (no credentials)
postgres:
  host: sparkly-db-rw.databases.svc.cluster.local
  credentialsSecret: my-postgres-app
clickhouse:
  host: clickhouse-sparkly.databases.svc.cluster.local
  credentialsSecret: my-clickhouse-app
rabbitmq:
  connectionSecret: my-rabbitmq-uri
stepCA:
  url: https://step-ca.pki.svc.cluster.local:9000
networkPolicy:
  ingressNamespaceSelector:
    kubernetes.io/metadata.name: traefik
  ingressPodSelector:
    app.kubernetes.io/name: traefik
```

Adjust namespace/pod selectors to your actual Traefik installation; they are
combined with AND. Also allow these application pods in any operator/database/CA
network policies. The chart restricts application ingress to Envoy pods in the
same release and namespace. Gateway ingress is restricted to the selected
Traefik pods. Egress remains unrestricted for DNS and external dependencies.
Additional policies are additive: a broad allow policy elsewhere can weaken the
gateway boundary.

```sh
helm lint charts/sparkly --strict
# Use a version already published by the release workflow.
helm template sparkly oci://ghcr.io/arizona-bebras/sparkly/charts/sparkly \
  --version 0.2.0 -n sparkly -f production-values.yaml
helm upgrade --install sparkly oci://ghcr.io/arizona-bebras/sparkly/charts/sparkly \
  --version 0.2.0 -n sparkly -f production-values.yaml --wait --timeout 10m
kubectl -n sparkly get certificates,ingress,ingressroutetcp,pods
kubectl -n sparkly wait --for=condition=Ready certificate --all --timeout=180s
```

Use `--create-namespace` if a secret controller will populate credentials after
namespace creation. Secrets and certificates must be available before pods can
start; temporary database/broker unavailability results in container restarts.
Probes use TCP listeners because the services do not expose dedicated readiness
endpoints; they do not measure dependency health after startup.

## Rotation and optional workloads

Envoy mounts the whole projected Secret directory without `subPath` and watches
it via filesystem SDS, allowing TLS certificate/key and client trust updates
without restarting the gateway. Gateway configuration changes require rebuilding
its image and publishing a new release.
The public root download is loaded at Envoy configuration load time, so after
changing the device root bundle, restart Envoy to refresh the downloadable copy:

```sh
kubectl -n sparkly rollout restart deployment/sparkly-sparkly-envoy
```

The release name is `sparkly` in this example; override the Deployment name if
using another release or `fullnameOverride`. Backend provisioner/credential
changes require a backend restart; env-based database/broker credentials require
restarting their consuming workloads. Preserve overlapping CA trust during a
planned root rotation.

ML is disabled because its current entrypoint is a stub; the existing ML gateway
route will return an upstream error until ML is implemented and enabled with
`components.ml.enabled=true`. The symbolicator is also disabled by default. To
enable it, provide an image whose ENTRYPOINT is the `sparkly-symbolicator` binary,
set `components.symbolicator.enabled=true`, and set
`symbolicator.firmwareClaimName` to an existing PVC containing the finalized,
unstripped `firmware.elf` at its root. The chart passes
`worker /app/fixtures/firmware.elf`; ensure PVC access modes support the requested
replica count. There is no symbolicator Dockerfile in the current repository, so the release
workflow does not publish its image. Build/publish it separately and set its
relative `image.repository` and explicit `image.tag` before enabling it.

Backend and analytics default to one replica with Recreate upgrades because
migrations run on startup. Do not scale them until concurrent migration behavior
has been addressed. PostgreSQL currently hardcodes `sslmode=disable` and
ClickHouse uses plaintext native TCP; configure operator endpoints accordingly.
Encrypted database connections need application changes. Device certificate
renewal and immediate revocation remain application limitations described in the
[device enrollment documentation](../../step-ca/README.md).

## Chart checks

```sh
python3 -m pip install PyYAML==6.0.3
python3 -m unittest discover -s charts/sparkly/tests -v
helm lint charts/sparkly --strict
```

The tests cover SNI routing, mTLS separation, Secret isolation, certificate SANs,
rotation mounts, network policy boundaries, namespace-local Service names, image
configuration, and invalid values. The gateway configuration in `envoy/` is the
single source included by `envoy/Dockerfile`. Existing `envoy/proxy.test.mjs`
tests the gateway's runtime authentication behavior.
