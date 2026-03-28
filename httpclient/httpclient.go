package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/interpolate"
)

// Client wraps net/http with interpolation and optional TLS skip.
type Client struct {
	http *http.Client
}

// New creates a Client. Pass insecure=true to skip TLS certificate verification.
func New(insecure bool) *Client {
	transport := &http.Transport{}

	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// Do executes the request after applying {{ VAR }} interpolation from vars.
func (c *Client) Do(ctx context.Context, req *config.Request, vars map[string]string) (*Response, error) {
	method := strings.ToUpper(interpolate.Apply(req.Method, vars))
	url := interpolate.Apply(req.URL, vars)
	headers := interpolate.ApplyToMap(req.Headers, vars)

	var bodyBytes []byte

	if req.Body != nil {
		var err error

		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	if len(bodyBytes) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.http.Do(httpReq) //nolint:bodyclose // closed inside newResponse
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return newResponse(resp)
}
