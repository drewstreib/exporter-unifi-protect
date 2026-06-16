package internal

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	loginPath       = "/api/auth/login"
	sensorsAPIPath  = "/proxy/protect/api/sensors"
	tokenCookieName = "TOKEN"
	csrfTokenHeader = "X-CSRF-Token" //nolint:gosec // header name, not a credential
)

// Client is a minimal UniFi Protect API client. It authenticates with a local
// account, caches the session token and CSRF token until the token's JWT
// expires, and issues authenticated requests against the Protect API.
type Client struct {
	baseURL *url.URL
	http    *http.Client

	username string
	password string

	mu      sync.Mutex
	token   string
	csrf    string
	expires time.Time
}

// NewClient creates a Client for the UniFi console at baseURL. The console
// serves a self-signed certificate, so TLS verification is disabled.
func NewClient(baseURL *url.URL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed UniFi console certificate
			},
		},
	}
}

// ListSensors fetches every adopted sensor from the Protect API.
func (c *Client) ListSensors(ctx context.Context) ([]Sensor, error) {
	var sensors []Sensor
	if err := c.get(ctx, sensorsAPIPath, &sensors); err != nil {
		return nil, err
	}

	return sensors, nil
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ensureSession logs in when there is no cached token or it has expired.
func (c *Client) ensureSession(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expires) {
		return nil
	}

	body, err := json.Marshal(loginRequest{Username: c.username, Password: c.password})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.JoinPath(loginPath).String(), bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("login failed: unexpected status %d: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	_, _ = io.Copy(io.Discard, res.Body)

	c.csrf = res.Header.Get(csrfTokenHeader)
	c.token = ""

	for _, cookie := range res.Cookies() {
		if cookie.Name == tokenCookieName {
			c.token = cookie.Value
			c.expires = tokenExpiry(cookie.Value)
		}
	}

	if c.token == "" {
		return fmt.Errorf("login succeeded but no %s cookie was returned", tokenCookieName)
	}

	return nil
}

// tokenExpiry reads the exp claim from the session JWT so we know when to
// re-authenticate. The token is never verified locally; it is only replayed as
// a cookie. A token whose expiry cannot be read is treated as already expired,
// forcing a fresh login on the next request.
func tokenExpiry(raw string) time.Time {
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser().ParseUnverified(raw, &claims); err != nil {
		return time.Time{}
	}

	if claims.ExpiresAt == nil {
		return time.Time{}
	}

	return claims.ExpiresAt.Time
}

// get performs an authenticated GET of path and decodes the JSON body into v.
func (c *Client) get(ctx context.Context, path string, v any) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL.JoinPath(path).String(), nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	req.Header.Set(csrfTokenHeader, c.csrf)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: c.token})
	c.mu.Unlock()

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	return json.NewDecoder(res.Body).Decode(v)
}
