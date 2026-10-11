// Package scrapbox is a client for the Scrapbox (Cosense) REST API.
//
// Endpoints are grouped into services by resource (Pages, Projects, Users, SmartContext).
// To add an API, add a method to a service, or define a new service and register it in Client.
package scrapbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the base URL of the Scrapbox REST API.
	DefaultBaseURL = "https://scrapbox.io/api"
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	defaultUserAgent = "scrapbox-cli"
)

// Client is a Scrapbox REST API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       Authenticator
	userAgent  string
	retry      RetryPolicy

	Pages        *PagesService
	Projects     *ProjectsService
	Users        *UsersService
	SmartContext *SmartContextService
}

// service is the Client reference shared by all services.
type service struct {
	client *Client
}

// Option configures a Client.
type Option func(*Client) error

// WithBaseURL sets the API base URL, for tests and self-hosted servers.
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid base URL %q: %w", raw, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("invalid base URL %q: scheme must be http or https", raw)
		}
		c.baseURL = strings.TrimRight(raw, "/")
		return nil
	}
}

// WithHTTPClient sets the *http.Client to use.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = hc
		return nil
	}
}

// WithAuthenticator sets the auth method.
func WithAuthenticator(a Authenticator) Option {
	return func(c *Client) error {
		c.auth = a
		return nil
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
}

// NewClient creates a Client.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
		auth:       NoAuth{},
		userAgent:  defaultUserAgent,
		retry:      DefaultRetryPolicy,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	common := service{client: c}
	c.Pages = (*PagesService)(&common)
	c.Projects = (*ProjectsService)(&common)
	c.Users = (*UsersService)(&common)
	c.SmartContext = (*SmartContextService)(&common)
	return c, nil
}

// newRequest builds a request for an API path (escaped, without a leading "/").
// A non-nil body is sent as JSON.
func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Request, error) {
	u := c.baseURL + "/" + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", c.userAgent)
	c.auth.Apply(req)
	return req, nil
}

// do sends req and converts non-2xx responses to *APIError.
// Transient failures of GET requests are retried by the RetryPolicy; writes are never retried
// because resending them may apply the change twice. On success, the caller must close resp.Body.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	maxRetries := c.retry.MaxRetries
	if req.Method != http.MethodGet {
		maxRetries = 0
	}
	for attempt := 0; ; attempt++ {
		resp, err := c.httpClient.Do(req)
		canRetry := attempt < maxRetries && req.Context().Err() == nil
		if err != nil {
			if !canRetry {
				return nil, fmt.Errorf("%s %s: %w", req.Method, req.URL.Redacted(), err)
			}
		} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		} else if !canRetry || !retryable(resp.StatusCode) {
			defer resp.Body.Close()
			return nil, newAPIError(req, resp)
		}

		wait := c.retry.delay(attempt, resp)
		if resp != nil {
			discard(resp)
		}
		if err := sleep(req.Context(), wait); err != nil {
			return nil, fmt.Errorf("%s %s: %w", req.Method, req.URL.Redacted(), err)
		}
	}
}

// getJSON sends a GET request and decodes the response body into v.
// It returns the response header for callers that need it, e.g. for paging.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, v any) (http.Header, error) {
	return c.doJSON(ctx, http.MethodGet, path, query, nil, v)
}

// postJSON sends body as JSON in a POST request and decodes the response body into v.
func (c *Client) postJSON(ctx context.Context, path string, body, v any) error {
	_, err := c.doJSON(ctx, http.MethodPost, path, nil, body, v)
	return err
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, v any) (http.Header, error) {
	req, err := c.newRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return nil, fmt.Errorf("decode response of %s: %w", req.URL.Redacted(), err)
	}
	return resp.Header, nil
}

// getText sends a GET request and returns the response body as a string.
func (c *Client) getText(ctx context.Context, path string, query url.Values) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response of %s: %w", req.URL.Redacted(), err)
	}
	return string(b), nil
}
