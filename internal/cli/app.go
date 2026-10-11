// Package cli は sb コマンドのサブコマンド定義を提供する。
//
// 各サブコマンドは App から Client / Printer / 設定を受け取る。
// 新しいコマンドを追加する場合は newXxxCmd(app *App) *cobra.Command を定義し、親コマンドに AddCommand する。
package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/output"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

// IOStreams はコマンドの入出力先である。テストではバッファに差し替える。
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// BuildInfo はビルド時に埋め込むバージョン情報である。
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// App はコマンド間で共有する依存関係である。
// 設定と出力形式はルートコマンドの PersistentPreRunE で解決し、Printer と Client は初回利用時に生成する。
type App struct {
	IO        IOStreams
	Getenv    func(string) string
	BuildInfo BuildInfo
	// HTTPClient が nil の場合は scrapbox パッケージのデフォルトを使う。
	HTTPClient *http.Client

	cfg     config.Config
	format  output.Format
	printer *output.Printer
	client  *scrapbox.Client
}

// Printer は出力形式に応じた Printer を返す。
func (a *App) Printer() *output.Printer {
	if a.printer == nil {
		a.printer = output.NewPrinter(a.IO.Out, a.format)
	}
	return a.printer
}

// Client は設定された認証情報を使う Scrapbox API クライアントを返す。初回呼び出し時に生成する。
func (a *App) Client() (*scrapbox.Client, error) {
	if a.client != nil {
		return a.client, nil
	}
	c, err := a.NewClient(a.cfg.Credentials.Authenticator())
	if err != nil {
		return nil, err
	}
	a.client = c
	return c, nil
}

// NewClient は指定した認証方式で Scrapbox API クライアントを生成する。
// 保存前の認証情報を検証する auth login などで使う。
func (a *App) NewClient(auth scrapbox.Authenticator) (*scrapbox.Client, error) {
	// 注入された *http.Client を書き換えないようコピーしてからタイムアウトを設定する。
	var hc http.Client
	if a.HTTPClient != nil {
		hc = *a.HTTPClient
	}
	hc.Timeout = a.cfg.Timeout

	ua := "scrapbox-cli"
	if a.BuildInfo.Version != "" {
		ua += "/" + a.BuildInfo.Version
	}
	retry := scrapbox.DefaultRetryPolicy
	retry.MaxRetries = a.cfg.MaxRetries
	return scrapbox.NewClient(
		scrapbox.WithBaseURL(a.cfg.BaseURL),
		scrapbox.WithRetryPolicy(retry),
		scrapbox.WithHTTPClient(&hc),
		scrapbox.WithAuthenticator(auth),
		scrapbox.WithUserAgent(ua),
	)
}

// errNoProject はプロジェクトが特定できないときのエラーである。
var errNoProject = fmt.Errorf("project is not specified: use --project, set %s, or run 'sb config set project <name>'", config.EnvProject)

// Project は対象プロジェクト名を返す。未設定ならエラーにする。
func (a *App) Project() (string, error) {
	if a.cfg.Project == "" {
		return "", errNoProject
	}
	return a.cfg.Project, nil
}

// PageRef はコマンド引数をページ参照に変換する。
// 引数がページ URL ならその URL のプロジェクトを、そうでなければ設定されたプロジェクトを使う。
func (a *App) PageRef(arg string) (scrapbox.PageRef, error) {
	if scrapbox.IsURL(arg) {
		return scrapbox.ParsePageURL(arg)
	}
	project, err := a.Project()
	if err != nil {
		return scrapbox.PageRef{}, err
	}
	return scrapbox.PageRef{Project: project, Title: arg}, nil
}

// projectClient は対象プロジェクトと API クライアントを返す。
func (a *App) projectClient() (string, *scrapbox.Client, error) {
	project, err := a.Project()
	if err != nil {
		return "", nil, err
	}
	client, err := a.Client()
	return project, client, err
}

// pageClient はページ引数を解決し、API クライアントとともに返す。
func (a *App) pageClient(arg string) (scrapbox.PageRef, *scrapbox.Client, error) {
	ref, err := a.PageRef(arg)
	if err != nil {
		return ref, nil, err
	}
	client, err := a.Client()
	return ref, client, err
}

// ErrorHint はエラーに応じて利用者向けの対処方法を返す。対処方法がなければ空文字を返す。
func ErrorHint(err error) string {
	switch {
	case errors.Is(err, scrapbox.ErrUnauthorized), errors.Is(err, scrapbox.ErrForbidden):
		return fmt.Sprintf("hint: this request requires authentication (private projects, Smart Context, etc.). run 'sb auth login', or set %s, %s or %s.",
			config.EnvPersonalToken, config.EnvServiceAccountKey, config.EnvSessionID)
	case errors.Is(err, scrapbox.ErrNotFound):
		return "hint: check the project name and page title."
	}
	return ""
}
