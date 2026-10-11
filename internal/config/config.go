// Package config は CLI の設定を環境変数から読み込む。
// フラグによる上書きは cli パッケージが行う。
package config

import (
	"fmt"
	"time"

	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

// 環境変数名である。COSENSE_PROJECT_NAME / COSENSE_SID / SCRAPBOX_API_URL は scrapbox_mcp と共通である。
const (
	EnvProject           = "COSENSE_PROJECT_NAME"
	EnvPersonalToken     = "COSENSE_PAT"
	EnvServiceAccountKey = "COSENSE_SERVICE_ACCOUNT_KEY"
	EnvSessionID         = "COSENSE_SID"
	EnvBaseURL           = "SCRAPBOX_API_URL"
	EnvTimeout           = "SCRAPBOX_TIMEOUT"
)

// Config は CLI の設定である。
type Config struct {
	// Project はデフォルトのプロジェクト名である。
	Project string
	BaseURL string
	Timeout time.Duration
	// Credentials はシェル履歴やプロセス一覧に残らないよう、環境変数からのみ読み込む。
	Credentials Credentials
}

// Credentials は認証情報である。複数設定されている場合は
// Personal Access Token > Service Account Key > connect.sid の順に優先する。
type Credentials struct {
	PersonalAccessToken string
	ServiceAccountKey   string
	SessionID           string
}

// Authenticator は設定された認証情報に対応する Authenticator を返す。
func (c Credentials) Authenticator() scrapbox.Authenticator {
	switch {
	case c.PersonalAccessToken != "":
		return scrapbox.PersonalAccessToken(c.PersonalAccessToken)
	case c.ServiceAccountKey != "":
		return scrapbox.ServiceAccountKey(c.ServiceAccountKey)
	case c.SessionID != "":
		return scrapbox.SessionCookie(c.SessionID)
	default:
		return scrapbox.NoAuth{}
	}
}

// Load はデフォルト値に環境変数を重ねた設定を返す。getenv には通常 os.Getenv を渡す。
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Project: getenv(EnvProject),
		BaseURL: scrapbox.DefaultBaseURL,
		Timeout: scrapbox.DefaultTimeout,
		Credentials: Credentials{
			PersonalAccessToken: getenv(EnvPersonalToken),
			ServiceAccountKey:   getenv(EnvServiceAccountKey),
			SessionID:           getenv(EnvSessionID),
		},
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
	return cfg, nil
}
