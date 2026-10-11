// Package config は CLI の設定を設定ファイルと環境変数から読み込む。
// 優先順位は フラグ > 環境変数 > 設定ファイル > デフォルト値 である。フラグによる上書きは cli パッケージが行う。
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
	// EnvConfigFile は設定ファイルのパスを上書きする。
	EnvConfigFile = "SB_CONFIG"
)

// Source は設定値の取得元である。
type Source string

const (
	SourceNone Source = ""
	SourceEnv  Source = "env"
	SourceFile Source = "file"
)

// Config は CLI の設定である。
type Config struct {
	// Project はデフォルトのプロジェクト名である。
	Project string
	BaseURL string
	Timeout time.Duration
	// Credentials は認証情報である。シェル履歴に残らないよう、フラグでは受け付けない。
	Credentials Credentials
	// CredentialSource は Credentials の取得元である。
	CredentialSource Source
	// FilePath は読み込んだ (または保存先となる) 設定ファイルのパスである。
	FilePath string
}

// AuthType は認証方式である。
type AuthType string

const (
	AuthPersonalAccessToken AuthType = "pat"
	AuthServiceAccount      AuthType = "service-account"
	AuthSessionID           AuthType = "sid"
)

// AuthTypes は指定可能な認証方式の一覧である。
var AuthTypes = []AuthType{AuthPersonalAccessToken, AuthServiceAccount, AuthSessionID}

// ParseAuthType は文字列を AuthType に変換する。
func ParseAuthType(s string) (AuthType, error) {
	for _, t := range AuthTypes {
		if AuthType(s) == t {
			return t, nil
		}
	}
	return "", fmt.Errorf("unknown auth type %q (available: %v)", s, AuthTypes)
}

// Credentials は認証情報である。複数設定されている場合は
// Personal Access Token > Service Account Key > connect.sid の順に優先する。
type Credentials struct {
	PersonalAccessToken string `json:"personalAccessToken,omitempty"`
	ServiceAccountKey   string `json:"serviceAccountKey,omitempty"`
	SessionID           string `json:"sessionId,omitempty"`
}

// IsZero は認証情報が 1 つも設定されていないかを返す。
func (c Credentials) IsZero() bool { return c == Credentials{} }

// Active は優先順位に従って実際に使う認証方式と値を返す。未設定なら空文字を返す。
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

// Authenticator は設定された認証情報に対応する Authenticator を返す。
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

// NewCredentials は認証方式と値から Credentials を生成する。
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

// Load は デフォルト値 < 設定ファイル < 環境変数 の順に重ねた設定を返す。getenv には通常 os.Getenv を渡す。
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
		Project:  file.Project,
		BaseURL:  scrapbox.DefaultBaseURL,
		Timeout:  scrapbox.DefaultTimeout,
		FilePath: path,
	}
	if !file.Credentials.IsZero() {
		cfg.Credentials, cfg.CredentialSource = file.Credentials, SourceFile
	}

	if v := getenv(EnvProject); v != "" {
		cfg.Project = v
	}
	// 環境変数の認証情報が 1 つでもあれば、設定ファイルの認証情報とは混ぜずに環境変数だけを使う。
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
	return cfg, nil
}
