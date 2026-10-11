// Package config loads the CLI config from the config file and environment variables.
// Precedence: flags > env > config file > defaults. The cli package applies flags.
package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

// Environment variable names. COSENSE_PROJECT_NAME, COSENSE_SID and SCRAPBOX_API_URL are shared with scrapbox_mcp.
const (
	EnvProject           = "COSENSE_PROJECT_NAME"
	EnvPersonalToken     = "COSENSE_PAT"
	EnvServiceAccountKey = "COSENSE_SERVICE_ACCOUNT_KEY"
	EnvSessionID         = "COSENSE_SID"
	EnvBaseURL           = "SCRAPBOX_API_URL"
	EnvTimeout           = "SCRAPBOX_TIMEOUT"
	EnvMaxRetries        = "SCRAPBOX_MAX_RETRIES"
	// EnvConfigFile overrides the config file path.
	EnvConfigFile = "SB_CONFIG"
)

// Source is where a setting came from.
type Source string

const (
	SourceNone Source = ""
	SourceEnv  Source = "env"
	SourceFile Source = "file"
)

// Config is the CLI config.
type Config struct {
	// Project is the default project name.
	Project string
	BaseURL string
	Timeout time.Duration
	// MaxRetries is the maximum number of retries on transient failures (429 / 5xx / network errors).
	MaxRetries int
	// Credentials are never accepted as flags, to keep them out of shell history.
	Credentials Credentials
	// CredentialSource is where Credentials came from.
	CredentialSource Source
	// FilePath is the config file path to read from and write to.
	FilePath string
}

// AuthType is an auth method.
type AuthType string

const (
	AuthPersonalAccessToken AuthType = "pat"
	AuthServiceAccount      AuthType = "service-account"
	AuthSessionID           AuthType = "sid"
)

var authTypes = []AuthType{AuthPersonalAccessToken, AuthServiceAccount, AuthSessionID}

// ParseAuthType parses an AuthType.
func ParseAuthType(s string) (AuthType, error) {
	for _, t := range authTypes {
		if AuthType(s) == t {
			return t, nil
		}
	}
	return "", fmt.Errorf("unknown auth type %q (available: %v)", s, authTypes)
}

// Credentials are API credentials. If several are set, the precedence is
// Personal Access Token > Service Account Key > connect.sid.
type Credentials struct {
	PersonalAccessToken string `json:"personalAccessToken,omitempty"`
	ServiceAccountKey   string `json:"serviceAccountKey,omitempty"`
	SessionID           string `json:"sessionId,omitempty"`
}

// IsZero reports whether no credential is set.
func (c Credentials) IsZero() bool { return c == Credentials{} }

// Active returns the auth method and value to use by precedence, or empty strings if none is set.
func (c Credentials) Active() (AuthType, string) {
	switch {
	case c.PersonalAccessToken != "":
		return AuthPersonalAccessToken, c.PersonalAccessToken
	case c.ServiceAccountKey != "":
		return AuthServiceAccount, c.ServiceAccountKey
	case c.SessionID != "":
		return AuthSessionID, c.SessionID
	}
	return "", ""
}

// Authenticator returns the Authenticator for the credentials.
func (c Credentials) Authenticator() scrapbox.Authenticator {
	typ, v := c.Active()
	switch typ {
	case AuthPersonalAccessToken:
		return scrapbox.PersonalAccessToken(v)
	case AuthServiceAccount:
		return scrapbox.ServiceAccountKey(v)
	case AuthSessionID:
		return scrapbox.SessionCookie(v)
	}
	return scrapbox.NoAuth{}
}

// NewCredentials creates Credentials from an auth method and a value.
func NewCredentials(typ AuthType, value string) Credentials {
	switch typ {
	case AuthPersonalAccessToken:
		return Credentials{PersonalAccessToken: value}
	case AuthServiceAccount:
		return Credentials{ServiceAccountKey: value}
	case AuthSessionID:
		return Credentials{SessionID: value}
	}
	return Credentials{}
}

// Load returns defaults overlaid by the config file, then env. getenv is usually os.Getenv.
func Load(getenv func(string) string) (Config, error) {
	path, err := FilePath(getenv)
	if err != nil {
		return Config{}, err
	}
	file, err := ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Project:    file.Project,
		BaseURL:    scrapbox.DefaultBaseURL,
		Timeout:    scrapbox.DefaultTimeout,
		MaxRetries: scrapbox.DefaultRetryPolicy.MaxRetries,
		FilePath:   path,
	}
	if !file.Credentials.IsZero() {
		cfg.Credentials, cfg.CredentialSource = file.Credentials, SourceFile
	}

	if v := getenv(EnvProject); v != "" {
		cfg.Project = v
	}
	// If any credential is set in env, use only env credentials; never mix them with the file's.
	envCreds := Credentials{
		PersonalAccessToken: getenv(EnvPersonalToken),
		ServiceAccountKey:   getenv(EnvServiceAccountKey),
		SessionID:           getenv(EnvSessionID),
	}
	if !envCreds.IsZero() {
		cfg.Credentials, cfg.CredentialSource = envCreds, SourceEnv
	}
	if v := getenv(EnvBaseURL); v != "" {
		cfg.BaseURL = v
	}
	if v := getenv(EnvTimeout); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid %s %q: %w", EnvTimeout, v, err)
		}
		cfg.Timeout = d
	}
	if v := getenv(EnvMaxRetries); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("invalid %s %q: must be a non-negative integer", EnvMaxRetries, v)
		}
		cfg.MaxRetries = n
	}
	return cfg, nil
}
