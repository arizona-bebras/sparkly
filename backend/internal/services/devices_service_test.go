package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
)

type deviceRepositoryFunc func(context.Context, string, func(domain.Device) error) error

func (f deviceRepositoryFunc) CreateDevice(ctx context.Context, hash string, enroll func(domain.Device) error) error {
	return f(ctx, hash, enroll)
}

type certificateIssuerFunc func(context.Context, domain.DeviceIdentity, domain.DeviceCertificateRequest) (domain.DeviceCertificate, error)

func (f certificateIssuerFunc) Issue(ctx context.Context, identity domain.DeviceIdentity, request domain.DeviceCertificateRequest) (domain.DeviceCertificate, error) {
	return f(ctx, identity, request)
}

func enrollmentCSR(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}))
}

func TestDeviceEnrollmentUsesRegisteredIdentity(t *testing.T) {
	device := domain.Device{ID: uuid.New(), ParkID: uuid.New()}
	csrPEM := enrollmentCSR(t)
	expectedCertificate := domain.DeviceCertificate{ChainPEM: "issued chain", ExpiresAt: time.Now().Add(time.Hour)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := deviceRepositoryFunc(func(receivedContext context.Context, _ string, enroll func(domain.Device) error) error {
		if receivedContext != ctx {
			t.Error("repository did not receive request context")
		}
		return enroll(device)
	})
	issuer := certificateIssuerFunc(func(receivedContext context.Context, identity domain.DeviceIdentity, request domain.DeviceCertificateRequest) (domain.DeviceCertificate, error) {
		if receivedContext != ctx {
			t.Error("issuer did not receive request context")
		}
		if identity.DeviceID() != device.ID || identity.ParkID() != device.ParkID {
			t.Error("issuer did not receive database-assigned identity")
		}
		if request.IsZero() || request.PEM() != csrPEM {
			t.Error("issuer did not receive validated CSR")
		}
		return expectedCertificate, nil
	})
	enrollment, err := NewDeviceService(repo, issuer).RegDevice(ctx, "park-secret", csrPEM)
	if err != nil {
		t.Fatal(err)
	}
	if enrollment.Identity.DeviceID() != device.ID || enrollment.Identity.ParkID() != device.ParkID || enrollment.Certificate != expectedCertificate {
		t.Fatalf("unexpected enrollment result: %+v", enrollment)
	}
}

func TestDeviceEnrollmentPreservesIssuanceFailure(t *testing.T) {
	device := domain.Device{ID: uuid.New(), ParkID: uuid.New()}
	repo := deviceRepositoryFunc(func(_ context.Context, _ string, enroll func(domain.Device) error) error {
		return enroll(device)
	})
	issuer := certificateIssuerFunc(func(context.Context, domain.DeviceIdentity, domain.DeviceCertificateRequest) (domain.DeviceCertificate, error) {
		return domain.DeviceCertificate{}, fmt.Errorf("CA request canceled: %w", context.Canceled)
	})
	enrollment, err := NewDeviceService(repo, issuer).RegDevice(context.Background(), "park-secret", enrollmentCSR(t))
	if !errors.Is(err, domain.ErrCertificateIssuance) || !errors.Is(err, context.Canceled) {
		t.Fatalf("issuance classification and original cause must be preserved: %v", err)
	}
	if enrollment != (DeviceEnrollment{}) {
		t.Fatal("failed enrollment returned credentials")
	}
}

func TestDeviceEnrollmentDiscardsCertificateOnCommitFailure(t *testing.T) {
	device := domain.Device{ID: uuid.New(), ParkID: uuid.New()}
	commitError := errors.New("database commit failed")
	repo := deviceRepositoryFunc(func(_ context.Context, _ string, enroll func(domain.Device) error) error {
		if err := enroll(device); err != nil {
			return err
		}
		return commitError
	})
	issuer := certificateIssuerFunc(func(context.Context, domain.DeviceIdentity, domain.DeviceCertificateRequest) (domain.DeviceCertificate, error) {
		return domain.DeviceCertificate{ChainPEM: "must not escape a failed enrollment"}, nil
	})
	enrollment, err := NewDeviceService(repo, issuer).RegDevice(context.Background(), "park-secret", enrollmentCSR(t))
	if !errors.Is(err, commitError) {
		t.Fatalf("expected commit failure, got %v", err)
	}
	if enrollment != (DeviceEnrollment{}) {
		t.Fatal("failed commit returned issued credentials")
	}
}
