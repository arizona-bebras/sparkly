# Backend — Control Plane

Device enrollment uses step-ca to sign a device-generated CSR. See [device certificate setup and enrollment](../step-ca/README.md) for required `STEP_CA_*` settings, the HTTP contract, and the Envoy trust boundary.

## Device PKI boundaries

- `internal/domain` owns device identity, CSR proof-of-possession and key policy, and issued certificate values. Validated identity and CSR values have private fields. These types do not depend on step-ca, HTTP, JWT, database models, or environment settings.
- `internal/services` orchestrates enrollment through repository and certificate issuer interfaces. It validates the CSR before accessing the repository and requests issuance inside the enrollment transaction. Issuance failures preserve both their application classification and their underlying cause.
- `internal/infrastructure/stepca` implements the issuer interface. Configuration loading, JWK authorization, HTTP transport, and certificate verification are separate concerns within the adapter.
- `internal/handlers` maps enrollment results to the HTTP response. JSON field names belong here rather than in domain types.

`cmd/backend` is the composition root: it resolves environment settings, constructs the step-ca adapter, and injects it into the service. Replacing the CA implementation does not require changing the enrollment workflow.

## Checks

Run checks from this directory with `go test ./...`.

To include real step-ca issuance after exporting the Compose credentials as described in the setup guide:

```sh
SPARKLY_TEST_CA_URL=https://localhost:9001 \
SPARKLY_TEST_CA_CREDENTIALS="$PWD/../step-ca/local" \
go test ./internal/infrastructure/stepca -run TestStepCAIntegration -count=1
```
