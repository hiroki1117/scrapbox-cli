package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
)

// runCmd はテスト用 API サーバーと環境変数でコマンドを実行し、stdout を返す。
func runCmd(t *testing.T, handler http.Handler, env map[string]string, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// 実環境の設定ファイルを読まないよう、設定ファイルを一時ディレクトリに隔離する。
	merged := map[string]string{
		config.EnvBaseURL:    srv.URL + "/api",
		config.EnvConfigFile: filepath.Join(t.TempDir(), "config.json"),
	}
	for k, v := range env {
		merged[k] = v
	}
	var out, errOut bytes.Buffer
	app := &App{
		IO:     IOStreams{In: strings.NewReader(""), Out: &out, ErrOut: &errOut},
		Getenv: func(k string) string { return merged[k] },
	}
	cmd := NewRootCmd(app)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func pageHandler(t *testing.T, wantPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != wantPath {
			t.Errorf("path = %q, want %q", r.URL.EscapedPath(), wantPath)
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"title":"T","lines":[{"text":"T"},{"text":"line2"}],"links":["A","B"]}`)
	})
}

func TestPageGet(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		args     []string
		wantPath string
		want     string
	}{
		{
			name:     "project from env",
			env:      map[string]string{config.EnvProject: "envproj"},
			args:     []string{"page", "get", "T"},
			wantPath: "/api/pages/envproj/T",
			want:     "T\nline2\n",
		},
		{
			name:     "flag overrides env",
			env:      map[string]string{config.EnvProject: "envproj"},
			args:     []string{"page", "get", "-p", "flagproj", "T", "-n"},
			wantPath: "/api/pages/flagproj/T",
			want:     "1  T\n2  line2\n",
		},
		{
			name:     "page url",
			args:     []string{"page", "get", "https://scrapbox.io/urlproj/T"},
			wantPath: "/api/pages/urlproj/T",
			want:     "T\nline2\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runCmd(t, pageHandler(t, tt.wantPath), tt.env, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPageLinks_JSON(t *testing.T) {
	got, err := runCmd(t, pageHandler(t, "/api/pages/p/T"), nil, "page", "links", "T", "-p", "p", "-o", "json")
	if err != nil {
		t.Fatal(err)
	}
	var links []string
	if err := json.Unmarshal([]byte(got), &links); err != nil {
		t.Fatalf("invalid json %q: %v", got, err)
	}
	if fmt.Sprint(links) != "[A B]" {
		t.Errorf("links = %v", links)
	}
}

func TestErrors(t *testing.T) {
	never := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s", r.URL)
	})
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no project", []string{"page", "get", "T"}, "project is not specified"},
		{"invalid output", []string{"page", "get", "T", "-p", "p", "-o", "xml"}, "unknown output format"},
		{"invalid hops", []string{"context", "T", "-p", "p", "--hops", "3"}, "hops must be 1 or 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runCmd(t, never, nil, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

// runWithConfig は設定ファイルのパスと標準入力を指定してコマンドを実行する。
func runWithConfig(t *testing.T, handler http.Handler, cfgPath, stdin string, args ...string) (string, string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	env := map[string]string{config.EnvBaseURL: srv.URL + "/api", config.EnvConfigFile: cfgPath}
	var out, errOut bytes.Buffer
	app := &App{
		IO:     IOStreams{In: strings.NewReader(stdin), Out: &out, ErrOut: &errOut},
		Getenv: func(k string) string { return env[k] },
	}
	cmd := NewRootCmd(app)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestAuthLoginStatusLogout(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-personal-access-token") == "goodtoken123" {
			fmt.Fprint(w, `{"isGuest":false,"name":"alice"}`)
			return
		}
		fmt.Fprint(w, `{"isGuest":true}`)
	})

	if _, _, err := runWithConfig(t, api, cfgPath, "badtoken\n", "auth", "login", "--with-token"); err == nil {
		t.Fatal("expected verification error for invalid token")
	}
	if f, _ := config.ReadFile(cfgPath); !f.Credentials.IsZero() {
		t.Fatal("invalid token must not be saved")
	}

	if _, _, err := runWithConfig(t, api, cfgPath, "goodtoken123\n", "auth", "login", "--with-token"); err != nil {
		t.Fatal(err)
	}
	if f, _ := config.ReadFile(cfgPath); f.Credentials.PersonalAccessToken != "goodtoken123" {
		t.Fatalf("credentials = %+v", f.Credentials)
	}

	out, _, err := runWithConfig(t, api, cfgPath, "", "auth", "status", "-o", "json")
	if err != nil {
		t.Fatal(err)
	}
	var st authStatus
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatal(err)
	}
	if st.User != "alice" || st.Source != config.SourceFile || strings.Contains(out, "goodtoken123") {
		t.Errorf("unexpected status (credential must be masked): %s", out)
	}

	if _, _, err := runWithConfig(t, api, cfgPath, "", "auth", "logout"); err != nil {
		t.Fatal(err)
	}
	if f, _ := config.ReadFile(cfgPath); !f.Credentials.IsZero() {
		t.Errorf("credentials remain after logout: %+v", f.Credentials)
	}
}

func TestConfigSetProject(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if _, _, err := runWithConfig(t, pageHandler(t, "/api/pages/cfgproj/T"), cfgPath, "", "config", "set", "project", "cfgproj"); err != nil {
		t.Fatal(err)
	}
	out, _, err := runWithConfig(t, pageHandler(t, "/api/pages/cfgproj/T"), cfgPath, "", "page", "get", "T")
	if err != nil {
		t.Fatal(err)
	}
	if out != "T\nline2\n" {
		t.Errorf("output = %q", out)
	}
	if _, _, err := runWithConfig(t, pageHandler(t, ""), cfgPath, "", "config", "set", "nokey", "v"); err == nil {
		t.Error("expected error for unknown key")
	}
}
