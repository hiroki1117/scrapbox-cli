package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

// runCmd runs a command against a test API server with env, and returns stdout.
func runCmd(t *testing.T, handler http.Handler, env map[string]string, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// Isolate the config file in a temp dir so the real one is never read.
	merged := map[string]string{
		config.EnvBaseURL:    srv.URL + "/api",
		config.EnvConfigFile: filepath.Join(t.TempDir(), "config.json"),
	}
	maps.Copy(merged, env)
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
		fmt.Fprint(w, `{"title":"T","persistent":true,"lines":[{"text":"T"},{"text":"line2"}],"links":["A","B"]}`)
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
	never := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
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

// runWithConfig runs a command with the given config file path and stdin.
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

func TestSearchCommands(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/api/pages/p/search/query":
			if r.URL.Query().Get("q") != "go cli" || r.URL.Query().Get("op") != "or" {
				t.Errorf("text query = %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"count":1,"pages":[{"title":"Go","lines":["Go","  go cli tool"]}]}`)
		case "/api/pages/p/search/vector/titles":
			fmt.Fprint(w, `{"pages":[{"title":"A","score":0.9,"linked":1,"exists":true},{"title":"B","score":0.5,"exists":false}]}`)
		case "/api/pages/v2/p/Start/links2hop":
			if r.URL.Query().Get("search") != "api token" {
				t.Errorf("related query = %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"links2hop":[{"title":"R","linked":2}],"pagination":{"total":1}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.EscapedPath())
		}
	})
	env := map[string]string{config.EnvProject: "p"}

	tests := []struct {
		args []string
		want string
	}{
		{[]string{"search", "text", "go", "cli", "--or"}, "Go\n    go cli tool\n"},
		{[]string{"search", "vector", "x", "--limit", "1"}, "SCORE  LINKED  TITLE\n0.900  1       A\n"},
		{[]string{"search", "related", "Start", "api", "token", "--hops", "2"}, "R\n"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args[:2], " "), func(t *testing.T) {
			got, err := runCmd(t, api, env, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("output = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestPageGet_MissingPage(t *testing.T) {
	// With authentication the API returns a missing page as a non-persistent empty page.
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"title":"X","persistent":false,"lines":[{"text":"X"}]}`)
	})
	_, err := runCmd(t, api, map[string]string{config.EnvProject: "proj"}, "page", "get", "X")
	if !errors.Is(err, scrapbox.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
