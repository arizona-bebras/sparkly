package stepca

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"sparkly/backend/internal/domain"
)

type certificateChain struct {
	certificates []*x509.Certificate
	pem          string
}

func parseCertificateChain(encoded []string) (certificateChain, error) {
	if len(encoded) == 0 {
		return certificateChain{}, errors.New("empty certificate chain")
	}
	var chain certificateChain
	var bundle strings.Builder
	for _, certPEM := range encoded {
		block, rest := pem.Decode([]byte(certPEM))
		if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(rest)) != 0 {
			return certificateChain{}, errors.New("invalid certificate in step-ca response")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return certificateChain{}, fmt.Errorf("parse issued certificate: %w", err)
		}
		chain.certificates = append(chain.certificates, certificate)
		bundle.Write(pem.EncodeToMemory(block))
	}
	chain.pem = bundle.String()
	return chain, nil
}

func (c certificateChain) leaf() *x509.Certificate { return c.certificates[0] }

func (c *Client) verifyCertificate(chain certificateChain, identity domain.DeviceIdentity, request domain.DeviceCertificateRequest) error {
	leaf := chain.leaf()
	if !request.MatchesPublicKey(leaf.RawSubjectPublicKeyInfo) {
		return errors.New("issued certificate public key does not match the CSR")
	}
	if err := verifyDeviceIdentity(leaf, identity); err != nil {
		return err
	}
	if err := verifyClientProfile(leaf); err != nil {
		return err
	}

	intermediates := x509.NewCertPool()
	for _, certificate := range chain.certificates[1:] {
		intermediates.AddCert(certificate)
	}
	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         c.roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return fmt.Errorf("verify issued device certificate chain: %w", err)
	}
	return nil
}

func verifyDeviceIdentity(certificate *x509.Certificate, identity domain.DeviceIdentity) error {
	if certificate.Subject.CommonName != identity.DeviceID().String() {
		return errors.New("issued certificate common name does not match the device")
	}
	if len(certificate.URIs) != 1 || certificate.URIs[0].String() != identity.URI() {
		return errors.New("issued certificate URI SAN does not match the device and park")
	}
	if len(certificate.DNSNames) > 0 || len(certificate.IPAddresses) > 0 || len(certificate.EmailAddresses) > 0 {
		return errors.New("issued certificate contains unexpected SANs")
	}
	return nil
}

func verifyClientProfile(certificate *x509.Certificate) error {
	if certificate.IsCA {
		return errors.New("issued device certificate must not be a CA")
	}
	if certificate.KeyUsage != x509.KeyUsageDigitalSignature {
		return errors.New("issued device certificate must allow only digital signatures")
	}
	if len(certificate.ExtKeyUsage) != 1 || certificate.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth || len(certificate.UnknownExtKeyUsage) > 0 {
		return errors.New("issued device certificate must allow only TLS client authentication")
	}
	return nil
}
