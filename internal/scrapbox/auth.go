package scrapbox

import "net/http"

// Authenticator adds credentials to requests.
// Implement it to add a new auth method.
type Authenticator interface {
	Apply(req *http.Request)
}

// NoAuth adds no credentials. Use it for public projects.
type NoAuth struct{}

// Apply does nothing.
func (NoAuth) Apply(*http.Request) {}

// SessionCookie authenticates with the connect.sid cookie.
type SessionCookie string

// Apply sets the connect.sid cookie.
func (s SessionCookie) Apply(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "connect.sid", Value: string(s)})
}

// PersonalAccessToken authenticates with a Personal Access Token,
// issued at https://scrapbox.io/settings/personal-access-tokens.
type PersonalAccessToken string

// Apply sets the x-personal-access-token header.
func (t PersonalAccessToken) Apply(req *http.Request) {
	req.Header.Set("x-personal-access-token", string(t))
}

// ServiceAccountKey authenticates with a Service Account access key (Business plan).
type ServiceAccountKey string

// Apply sets the x-service-account-access-key header.
func (k ServiceAccountKey) Apply(req *http.Request) {
	req.Header.Set("x-service-account-access-key", string(k))
}
