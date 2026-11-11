package stepca

import (
	"context"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
)

// Opt-in smoke test against a running Compose CA, with exported credentials.
func TestStepCAIntegration(t *testing.T) {
	caURL := os.Getenv("SPARKLY_TEST_CA_URL")
	if caURL == "" {
		t.Skip("set SPARKLY_TEST_CA_URL and SPARKLY_TEST_CA_CREDENTIALS to test a live CA")
	}
	credentials := os.Getenv("SPARKLY_TEST_CA_CREDENTIALS")
	if credentials == "" {
		t.Fatal("SPARKLY_TEST_CA_CREDENTIALS is required")
	}
	issuer, err := New(Config{
		URL: caURL, Provisioner: "sparkly-devices",
		RootFile:  filepath.Join(credentials, "root_ca.crt"),
		KeyFile:   filepath.Join(credentials, "provisioner.key"),
		KeyIDFile: filepath.Join(credentials, "provisioner.kid"),
	})
	if err != nil {
		t.Fatal(err)
	}
	device := domain.Device{ID: uuid.New(), ParkID: uuid.New()}
	identity, err := device.Identity()
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewDeviceCertificateRequest(csrFor(t, newKey(t), &x509.CertificateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := issuer.Issue(context.Background(), identity, request)
	if err != nil {
		t.Fatal(err)
	}
	if certificate.ChainPEM == "" || certificate.ExpiresAt.IsZero() {
		t.Fatal("missing issued credentials")
	}
	// The Compose provisioner must apply the one-year default without the
	// application requesting a lifetime. Allow for CA backdating and clock skew.
	remaining := time.Until(certificate.ExpiresAt)
	wantLifetime := 365 * 24 * time.Hour
	if remaining < wantLifetime-5*time.Minute || remaining > wantLifetime+5*time.Minute {
		t.Fatalf("device certificate lifetime = %s, want approximately %s", remaining, wantLifetime)
	}
}
