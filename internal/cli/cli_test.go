package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
)

// runCmd はテスト用 API サーバーと環境変数でコマンドを実行し、stdout を返す。
func runCmd(t *testing.T, handler http.Handler, env map[string]string, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	merged := map[string]string{config.EnvBaseURL: srv.URL + "/api"}
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
