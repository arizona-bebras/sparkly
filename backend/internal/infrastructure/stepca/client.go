// Package stepca implements device certificate issuance through a step-ca JWK provisioner.
package stepca

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"

	"sparkly/backend/internal/domain"
)

type Client struct {
	endpoint    string
	provisioner provisioner
	roots       *x509.CertPool
	httpClient  *http.Client
}

// Issue accepts validated domain values; CSR parsing is not repeated in the adapter.
func (c *Client) Issue(ctx context.Context, identity domain.DeviceIdentity, request domain.DeviceCertificateRequest) (domain.DeviceCertificate, error) {
	if err := identity.Validate(); err != nil {
		return domain.DeviceCertificate{}, err
	}
	if request.IsZero() {
		return domain.DeviceCertificate{}, domain.ErrInvalidCertificateRequest
	}

	token, err := c.provisioner.authorizationToken(c.endpoint, identity, request)
	if err != nil {
		return domain.DeviceCertificate{}, fmt.Errorf("authorize certificate issuance: %w", err)
	}
	signed, err := c.sign(ctx, request.PEM(), token)
	if err != nil {
		return domain.DeviceCertificate{}, err
	}
	chain, err := parseCertificateChain(signed.certificatePEMs())
	if err != nil {
		return domain.DeviceCertificate{}, err
	}
	if err := c.verifyCertificate(chain, identity, request); err != nil {
		return domain.DeviceCertificate{}, err
	}

	return domain.DeviceCertificate{
		ChainPEM:  chain.pem,
		ExpiresAt: chain.leaf().NotAfter,
	}, nil
}
