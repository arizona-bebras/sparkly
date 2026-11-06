package stepca

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxSignResponseBytes = 128 << 10

type signRequest struct {
	CSR   string `json:"csr"`
	Token string `json:"ott"`
}

type signResponse struct {
	Certificate string   `json:"crt"`
	CA          string   `json:"ca"`
	Chain       []string `json:"certChain"`
}

func (r signResponse) certificatePEMs() []string {
	if len(r.Chain) > 0 {
		return r.Chain
	}
	return []string{r.Certificate, r.CA}
}

func (c *Client) sign(ctx context.Context, csrPEM, token string) (signResponse, error) {
	payload, err := json.Marshal(signRequest{CSR: csrPEM, Token: token})
	if err != nil {
		return signResponse{}, fmt.Errorf("encode step-ca sign request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return signResponse{}, fmt.Errorf("create step-ca sign request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return signResponse{}, fmt.Errorf("step-ca sign request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		// CA error bodies may contain credentials. Log only the status, never the body.
		return signResponse{}, fmt.Errorf("step-ca sign returned HTTP %d", response.StatusCode)
	}
	var signed signResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxSignResponseBytes)).Decode(&signed); err != nil {
		return signResponse{}, fmt.Errorf("decode step-ca response: %w", err)
	}
	return signed, nil
}
