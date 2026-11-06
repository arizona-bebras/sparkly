package stepca

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
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

func TestClientIssue(t *testing.T) {
	provisionerKey := newKey(t)
	deviceKey := newKey(t)
	caKey := newKey(t)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test root"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	device := domain.Device{ID: uuid.New(), ParkID: uuid.New()}
	csrPEM := csrFor(t, deviceKey, &x509.CertificateRequest{})
	csr, err := domain.NewDeviceCertificateRequest(csrPEM)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := csr.Fingerprint()
	identity, err := device.Identity()
	if err != nil {
		t.Fatal(err)
	}
	var mode string
	var endpoint string
	seenIDs := map[string]bool{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/1.0/sign" {
			t.Errorf("unexpected CA request: %s %s", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		var request struct {
			CSR string `json:"csr"`
			OTT string `json:"ott"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		token, err := jwt.Parse(request.OTT,
			func(token *jwt.Token) (any, error) { return &provisionerKey.PublicKey, nil },
			jwt.WithValidMethods([]string{"ES256"}),
			jwt.WithAudience(endpoint),
			jwt.WithIssuer("sparkly-devices"),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			t.Error(err)
			w.WriteHeader(401)
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		metadata := claims["sparkly"].(map[string]any)
		confirmation := claims["cnf"].(map[string]any)
		if token.Header["kid"] != "test-kid" {
			t.Error("incorrect provisioner key ID")
		}
		if claims["sub"] != identity.DeviceID().String() || metadata["device_id"] != identity.DeviceID().String() {
			t.Error("incorrect device claim")
		}
		if metadata["park_id"] != identity.ParkID().String() || metadata["uri"] != identity.URI() {
			t.Error("incorrect park identity claim")
		}
		if confirmation["x5rt#S256"] != base64.RawURLEncoding.EncodeToString(fingerprint[:]) {
			t.Error("incorrect CSR fingerprint claim")
		}
		if request.CSR != csrPEM {
			t.Error("incorrect CSR")
		}
		jti := claims["jti"].(string)
		if seenIDs[jti] {
			t.Error("reused provisioner token ID")
		}
		seenIDs[jti] = true
		if mode == "unavailable" {
			w.WriteHeader(503)
			return
		}
		identityURI, _ := url.Parse(identity.URI())
		leaf := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject:      pkix.Name{CommonName: identity.DeviceID().String()},
			URIs:         []*url.URL{identityURI},
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			NotBefore:    time.Now().Add(-time.Minute),
			NotAfter:     time.Now().Add(30 * time.Minute),
		}
		publicKey := &deviceKey.PublicKey
		switch mode {
		case "wrong park":
			wrongIdentity, _ := domain.NewDeviceIdentity(identity.DeviceID(), uuid.New())
			leaf.URIs[0], _ = url.Parse(wrongIdentity.URI())
		case "wrong key":
			publicKey = &provisionerKey.PublicKey
		case "wrong key usage":
			leaf.KeyUsage = x509.KeyUsageCertSign
		case "server profile":
			leaf.ExtKeyUsage = append(leaf.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
		case "extra SAN":
			leaf.DNSNames = []string{"unexpected"}
		case "expired":
			leaf.NotAfter = time.Now().Add(-time.Second)
		}
		signer := caKey
		if mode == "untrusted signer" {
			signer = provisionerKey
		}
		parent := *ca
		parent.PublicKey = &signer.PublicKey
		der, err := x509.CreateCertificate(rand.Reader, leaf, &parent, publicKey, signer)
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"certChain": []string{certPEM, caPEM}})
	}))
	defer server.Close()
	endpoint = server.URL + "/1.0/sign"
	dir := t.TempDir()
	roots := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	roots = append(roots, []byte(caPEM)...)
	rootPath := filepath.Join(dir, "roots.pem")
	keyPath := filepath.Join(dir, "key.pem")
	keyDER, err := x509.MarshalECPrivateKey(provisionerKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootPath, roots, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	keyIDPath := filepath.Join(dir, "provisioner.kid")
	if err := os.WriteFile(keyIDPath, []byte("test-kid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := New(Config{URL: server.URL, RootFile: rootPath, Provisioner: "sparkly-devices", KeyIDFile: keyIDPath, KeyFile: keyPath})
	if err != nil {
		t.Fatal(err)
	}
	for _, testMode := range []string{"success", "wrong park", "wrong key", "wrong key usage", "server profile", "extra SAN", "expired", "untrusted signer", "unavailable"} {
		t.Run(testMode, func(t *testing.T) {
			mode = testMode
			certificate, err := client.Issue(context.Background(), identity, csr)
			if mode == "success" {
				if err != nil {
					t.Fatal(err)
				}
				if certificate.ExpiresAt.IsZero() || strings.Count(certificate.ChainPEM, "BEGIN CERTIFICATE") != 2 {
					t.Fatalf("unexpected enrollment response: %+v", certificate)
				}
			} else if err == nil {
				t.Fatal("accepted invalid CA response")
			}
		})
	}
}
