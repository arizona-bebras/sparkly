package stepca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const signingTimeout = 10 * time.Second

// Config describes step-ca connection and credential files. Environment variables
// are resolved by the composition root, not by the adapter or application service.
type Config struct {
	URL         string
	RootFile    string
	Provisioner string
	KeyID       string
	KeyIDFile   string // Used only when KeyID is empty.
	KeyFile     string // PEM-encoded ECDSA P-256 provisioner key, never the CA key.
}

func New(config Config) (*Client, error) {
	endpoint, err := signingEndpoint(config.URL)
	if err != nil {
		return nil, err
	}
	credentials, err := loadProvisioner(config)
	if err != nil {
		return nil, err
	}
	roots, err := loadRoots(config.RootFile)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	return &Client{
		endpoint:    endpoint,
		provisioner: credentials,
		roots:       roots,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   signingTimeout,
			// Never forward a CSR or authorization token to a redirect target.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func signingEndpoint(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", errors.New("CA URL must be an HTTPS origin")
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("CA URL must be an HTTPS origin")
	}
	if u.Path != "" && u.Path != "/" {
		return "", errors.New("CA URL must not contain a path")
	}
	return strings.TrimRight(origin, "/") + "/1.0/sign", nil
}

func loadProvisioner(config Config) (provisioner, error) {
	keyID := config.KeyID
	if keyID == "" && config.KeyIDFile != "" {
		data, err := os.ReadFile(config.KeyIDFile)
		if err != nil {
			return provisioner{}, fmt.Errorf("read provisioner key ID: %w", err)
		}
		keyID = strings.TrimSpace(string(data))
	}
	if config.Provisioner == "" || keyID == "" {
		return provisioner{}, errors.New("provisioner name and key ID are required")
	}
	key, err := loadProvisionerKey(config.KeyFile)
	if err != nil {
		return provisioner{}, err
	}
	return provisioner{name: config.Provisioner, keyID: keyID, key: key}, nil
}

func loadProvisionerKey(path string) (*ecdsa.PrivateKey, error) {
	keyPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provisioner key: %w", err)
	}
	key, err := jwt.ParseECPrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse provisioner key: %w", err)
	}
	if key.Curve != elliptic.P256() {
		return nil, errors.New("provisioner key must use P-256 (ES256)")
	}
	return key, nil
}

func loadRoots(path string) (*x509.CertPool, error) {
	rootPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CA root: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		return nil, errors.New("invalid CA root PEM")
	}
	return roots, nil
}
