package scrapbox

import "net/http"

// Authenticator はリクエストに認証情報を付与する。
// 新しい認証方式を追加する場合は、このインターフェースを実装する。
type Authenticator interface {
	Apply(req *http.Request)
}

// NoAuth は認証情報を付与しない。Public Project の参照に使う。
type NoAuth struct{}

// Apply は何もしない。
func (NoAuth) Apply(*http.Request) {}

// SessionCookie は connect.sid Cookie で認証する。
type SessionCookie string

// Apply は connect.sid Cookie を付与する。
func (s SessionCookie) Apply(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "connect.sid", Value: string(s)})
}

// PersonalAccessToken は Personal Access Token で認証する。
// https://scrapbox.io/settings/personal-access-tokens で発行する。
type PersonalAccessToken string

// Apply は x-personal-access-token ヘッダを付与する。
func (t PersonalAccessToken) Apply(req *http.Request) {
	req.Header.Set("x-personal-access-token", string(t))
}

// ServiceAccountKey は Service Account (Business プラン) のアクセスキーで認証する。
type ServiceAccountKey string

// Apply は x-service-account-access-key ヘッダを付与する。
func (k ServiceAccountKey) Apply(req *http.Request) {
	req.Header.Set("x-service-account-access-key", string(k))
}
