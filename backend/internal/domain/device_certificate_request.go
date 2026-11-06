package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// DeviceCertificateRequest is a validated proof of key possession.
// It carries no caller-selected identity: the park and device are assigned on enrollment.
type DeviceCertificateRequest struct {
	pem          string
	fingerprint  [sha256.Size]byte
	publicKeyDER []byte
}

func NewDeviceCertificateRequest(input string) (DeviceCertificateRequest, error) {
	csr, err := parseDeviceCSR(input)
	if err != nil {
		return DeviceCertificateRequest{}, fmt.Errorf("%w: %w", ErrInvalidCertificateRequest, err)
	}
	return DeviceCertificateRequest{
		pem:          input,
		fingerprint:  sha256.Sum256(csr.Raw),
		publicKeyDER: bytes.Clone(csr.RawSubjectPublicKeyInfo),
	}, nil
}

func (r DeviceCertificateRequest) PEM() string                    { return r.pem }
func (r DeviceCertificateRequest) Fingerprint() [sha256.Size]byte { return r.fingerprint }
func (r DeviceCertificateRequest) IsZero() bool                   { return r.pem == "" }

func (r DeviceCertificateRequest) MatchesPublicKey(publicKeyDER []byte) bool {
	return !r.IsZero() && bytes.Equal(r.publicKeyDER, publicKeyDER)
}

func parseDeviceCSR(input string) (*x509.CertificateRequest, error) {
	block, rest := pem.Decode([]byte(input))
	if block == nil || block.Type != "CERTIFICATE REQUEST" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("expected one PEM certificate request")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("verify CSR signature: %w", err)
	}
	if csr.Subject.CommonName != "" || hasRequestedSANs(csr) {
		return nil, errors.New("CSR must have an empty common name and no SANs; the backend assigns identity")
	}
	if err := validateDevicePublicKey(csr.PublicKey); err != nil {
		return nil, err
	}
	return csr, nil
}

func hasRequestedSANs(csr *x509.CertificateRequest) bool {
	return len(csr.DNSNames) > 0 || len(csr.IPAddresses) > 0 || len(csr.EmailAddresses) > 0 || len(csr.URIs) > 0
}

func validateDevicePublicKey(publicKey any) error {
	switch key := publicKey.(type) {
	case *ecdsa.PublicKey:
		switch key.Curve {
		case elliptic.P256(), elliptic.P384(), elliptic.P521():
			return nil
		default:
			return errors.New("unsupported EC curve")
		}
	case *rsa.PublicKey:
		if key.N.BitLen() < 2048 || key.E < 65537 {
			return errors.New("RSA key must be at least 2048 bits with exponent >= 65537")
		}
		return nil
	case ed25519.PublicKey:
		return nil
	default:
		return errors.New("unsupported CSR public key")
	}
}
