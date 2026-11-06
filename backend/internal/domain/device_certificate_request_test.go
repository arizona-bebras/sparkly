package domain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"testing"
)

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func csrFor(t *testing.T, key *ecdsa.PrivateKey, template *x509.CertificateRequest) string {
	t.Helper()
	der, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestDeviceCertificateRequest(t *testing.T) {
	key := newKey(t)
	valid := csrFor(t, key, &x509.CertificateRequest{})
	block, _ := pem.Decode([]byte(valid))
	block.Bytes[len(block.Bytes)-1] ^= 1
	tampered := string(pem.EncodeToMemory(block))
	for _, test := range []struct {
		name, input string
		valid       bool
	}{
		{"valid", valid, true},
		{"invalid PEM", "not a CSR", false},
		{"multiple PEM blocks", valid + valid, false},
		{"invalid proof of possession", tampered, false},
		{"caller common name", csrFor(t, key, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "victim"}}), false},
		{"caller SAN", csrFor(t, key, &x509.CertificateRequest{DNSNames: []string{"victim"}}), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := NewDeviceCertificateRequest(test.input)
			if (err == nil) != test.valid {
				t.Fatalf("NewDeviceCertificateRequest error = %v, valid = %v", err, test.valid)
			}
			if err != nil {
				if !errors.Is(err, ErrInvalidCertificateRequest) {
					t.Errorf("missing invalid CSR classification: %v", err)
				}
				return
			}
			block, _ := pem.Decode([]byte(test.input))
			expectedKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				t.Fatal(err)
			}
			if request.Fingerprint() != sha256.Sum256(block.Bytes) {
				t.Error("incorrect CSR fingerprint")
			}
			if !request.MatchesPublicKey(expectedKey) {
				t.Error("CSR public key does not match original key")
			}
			if request.PEM() != test.input {
				t.Error("CSR encoding was not preserved")
			}

		})
	}
}

func TestZeroDeviceCertificateRequest(t *testing.T) {
	var request DeviceCertificateRequest
	if !request.IsZero() || request.MatchesPublicKey(nil) {
		t.Fatal("zero request must not match any public key")
	}
}
