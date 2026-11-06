package handlers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
)

type enrollmentRepo struct {
	device    domain.Device
	err       error
	called    bool
	committed bool
	t         *testing.T
}

func (r *enrollmentRepo) CreateDevice(_ context.Context, token string, issue func(domain.Device) error) error {
	r.called = true
	if token != "park-secret" {
		r.t.Error("repository received incorrect enrollment token")
	}
	if r.err != nil {
		return r.err
	}
	if err := issue(r.device); err != nil {
		return err
	}
	r.committed = true
	return nil
}

type enrollmentIssuer struct {
	err    error
	called bool
}

func (i *enrollmentIssuer) Issue(_ context.Context, _ domain.DeviceIdentity, _ domain.DeviceCertificateRequest) (domain.DeviceCertificate, error) {
	i.called = true
	if i.err != nil {
		return domain.DeviceCertificate{}, i.err
	}
	return domain.DeviceCertificate{ChainPEM: "test-chain"}, nil
}

func TestDeviceEnrollment(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		t.Fatal(err)
	}
	csrPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}))
	validBody, _ := json.Marshal(map[string]string{"park_token": "park-secret", "csr_pem": csrPEM})
	for _, test := range []struct {
		name, body               string
		repoError, issuerError   error
		status                   int
		repoCalled, issuerCalled bool
	}{
		{name: "enrolled", body: string(validBody), status: 201, repoCalled: true, issuerCalled: true},
		{name: "invalid token", body: string(validBody), repoError: domain.ErrTokenNotFound, status: 401, repoCalled: true},
		{name: "CA failure", body: string(validBody), issuerError: errors.New("private CA failure details"), status: 503, repoCalled: true, issuerCalled: true},
		{name: "database failure", body: string(validBody), repoError: errors.New("database failure"), status: 500, repoCalled: true},
		{name: "missing CSR", body: `{"park_token":"park-secret"}`, status: 400},
		{name: "missing token", body: `{"csr_pem":"invalid"}`, status: 400},
		{name: "invalid CSR", body: `{"park_token":"park-secret","csr_pem":"invalid"}`, status: 400},
		{name: "caller identity", body: `{"park_token":"park-secret","device_id":"victim"}`, status: 400},
		{name: "multiple objects", body: string(validBody) + ` {}`, status: 400},
		{name: "oversized", body: `{"park_token":"` + strings.Repeat("x", 33<<10) + `"}`, status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &enrollmentRepo{device: domain.Device{ID: uuid.New(), ParkID: uuid.New()}, err: test.repoError, t: t}
			issuer := &enrollmentIssuer{err: test.issuerError}
			handler := NewDeviceHandler(services.NewDeviceService(repo, issuer))
			recorder := httptest.NewRecorder()
			handler.RegDevice(recorder, httptest.NewRequest(http.MethodPost, "/devices/reg", strings.NewReader(test.body)))
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.status, recorder.Body)
			}
			if repo.called != test.repoCalled || issuer.called != test.issuerCalled {
				t.Fatalf("repo called %v, issuer called %v", repo.called, issuer.called)
			}
			if repo.committed != (test.status == 201) {
				t.Fatalf("unexpected commit: %v", repo.committed)
			}
			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Error("missing cache protection")
			}
			if test.status == 201 {
				var certificate deviceEnrollmentResponse
				if err := json.Unmarshal(recorder.Body.Bytes(), &certificate); err != nil {
					t.Fatal(err)
				}
				if certificate.DeviceID != repo.device.ID || certificate.ParkID != repo.device.ParkID {
					t.Error("wrong enrollment identity")
				}
				if strings.Contains(recorder.Body.String(), "device_token") {
					t.Error("returned legacy bearer token")
				}
			} else if strings.Contains(recorder.Body.String(), "private CA") {
				t.Error("leaked CA failure details")
			}
		})
	}
}
