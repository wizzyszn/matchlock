package txline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const guestAuthTimeout = 15 * time.Second

type guestStartResponse struct {
	Token string `json:"token"`
}

// Client talks to the TxLINE REST and SSE APIs with dual-header auth.
type Client struct {
	baseURL    string
	guestURL   string
	httpClient *http.Client
	apiToken   string

	mu  sync.RWMutex
	jwt string
}

// NewClient creates a TxLINE API client. Call EnsureGuestJWT before data requests.
func NewClient(baseURL, guestURL, apiToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 0}
	}
	return &Client{
		baseURL:    baseURL,
		guestURL:   guestURL,
		httpClient: httpClient,
		apiToken:   apiToken,
	}
}

// APIToken returns the activated API token (never log this value).
func (c *Client) APIToken() string {
	return c.apiToken
}

// JWT returns the current guest JWT, if any.
func (c *Client) JWT() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwt
}

// AuthHeaders returns headers required on all TxLINE data/SSE requests.
func (c *Client) AuthHeaders() (http.Header, error) {
	jwt := c.JWT()
	if jwt == "" {
		return nil, fmt.Errorf("guest JWT not set; call EnsureGuestJWT first")
	}
	h := make(http.Header)
	h.Set("Authorization", "Bearer "+jwt)
	h.Set("X-Api-Token", c.apiToken)
	return h, nil
}

// EnsureGuestJWT fetches a guest JWT when missing or force is true.
func (c *Client) EnsureGuestJWT(ctx context.Context, force bool) error {
	if !force && c.JWT() != "" {
		return nil
	}
	return c.refreshGuestJWT(ctx)
}

func (c *Client) refreshGuestJWT(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, guestAuthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.guestURL, nil)
	if err != nil {
		return fmt.Errorf("build guest auth request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("guest auth request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read guest auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("guest auth failed: status=%d body=%s", resp.StatusCode, truncate(body, 256))
	}

	var parsed guestStartResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("decode guest auth response: %w", err)
	}
	if parsed.Token == "" {
		return fmt.Errorf("guest auth response missing token")
	}

	c.mu.Lock()
	c.jwt = parsed.Token
	c.mu.Unlock()
	return nil
}

// DoAuthenticated performs an HTTP request with TxLINE auth headers.
// On 401 it refreshes the guest JWT once and retries.
func (c *Client) DoAuthenticated(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.EnsureGuestJWT(ctx, false); err != nil {
		return nil, err
	}

	resp, err := c.doWithAuth(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	resp.Body.Close()

	if err := c.EnsureGuestJWT(ctx, true); err != nil {
		return nil, err
	}
	return c.doWithAuth(ctx, req)
}

func (c *Client) doWithAuth(ctx context.Context, req *http.Request) (*http.Response, error) {
	headers, err := c.AuthHeaders()
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}
	return c.httpClient.Do(req)
}

// Ping verifies TxLINE auth endpoints are reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.EnsureGuestJWT(ctx, true)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}