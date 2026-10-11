// Package scrapbox は Scrapbox (Cosense) REST API のクライアントを提供する。
//
// エンドポイントはリソースごとにサービス (Pages, Projects, Users, SmartContext) へ分割している。
// 新しい API を追加する場合は、該当サービスにメソッドを追加するか、新しいサービスを定義して Client に登録する。
package scrapbox

import (
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
	// DefaultBaseURL は Scrapbox REST API のベース URL である。
	DefaultBaseURL = "https://scrapbox.io/api"
	// DefaultTimeout は HTTP リクエストのデフォルトタイムアウトである。
	DefaultTimeout = 30 * time.Second

	defaultUserAgent = "scrapbox-cli"
)

// Client は Scrapbox REST API のクライアントである。
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       Authenticator
	userAgent  string

	Pages        *PagesService
	Projects     *ProjectsService
	Users        *UsersService
	SmartContext *SmartContextService
}

// service は各サービスが共有する Client への参照である。
type service struct {
	client *Client
}

// Option は Client の設定を変更する。
type Option func(*Client) error

// WithBaseURL は API のベース URL を変更する。テストやセルフホスト環境向けである。
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

// WithHTTPClient は内部で使う *http.Client を差し替える。
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = hc
		return nil
	}
}

// WithAuthenticator は認証方式を設定する。
func WithAuthenticator(a Authenticator) Option {
	return func(c *Client) error {
		c.auth = a
		return nil
	}
}

// WithUserAgent は User-Agent ヘッダを設定する。
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
}

// NewClient は Client を生成する。
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
		auth:       NoAuth{},
		userAgent:  defaultUserAgent,
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

// newRequest は API のパス (エスケープ済み、先頭の "/" なし) から GET リクエストを組み立てる。
func (c *Client) newRequest(ctx context.Context, path string, query url.Values) (*http.Request, error) {
	u := c.baseURL + "/" + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	c.auth.Apply(req)
	return req, nil
}

// do はリクエストを送り、2xx 以外のレスポンスを *APIError に変換する。
// 成功時は呼び出し側が resp.Body を閉じる責務を持つ。
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", req.Method, req.URL.Redacted(), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, newAPIError(req, resp)
	}
	return resp, nil
}

// getJSON は GET リクエストを送り、レスポンスボディを v にデコードする。
// ページングなどでヘッダが必要な呼び出し側のために、レスポンスヘッダを返す。
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, v any) (http.Header, error) {
	req, err := c.newRequest(ctx, path, query)
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

// getText は GET リクエストを送り、レスポンスボディを文字列として返す。
func (c *Client) getText(ctx context.Context, path string, query url.Values) (string, error) {
	req, err := c.newRequest(ctx, path, query)
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
