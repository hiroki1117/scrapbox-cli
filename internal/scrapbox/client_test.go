package scrapbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// newTestClient は handler を API サーバーとする Client を返す。
func newTestClient(t *testing.T, handler http.Handler, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient(append([]Option{WithBaseURL(srv.URL + "/api")}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPagesGet_EscapesTitle(t *testing.T) {
	var gotPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		fmt.Fprint(w, `{"title":"a b/c","lines":[{"text":"a b/c"},{"text":"body"}]}`)
	}))

	page, err := c.Pages.Get(context.Background(), "proj", "a b/c")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/api/pages/proj/a_b%2Fc"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if len(page.Lines) != 2 || page.Lines[1].Text != "body" {
		t.Errorf("unexpected lines: %+v", page.Lines)
	}
}

func TestAuthenticators(t *testing.T) {
	tests := []struct {
		name  string
		auth  Authenticator
		check func(r *http.Request) bool
	}{
		{"pat", PersonalAccessToken("tok"), func(r *http.Request) bool { return r.Header.Get("x-personal-access-token") == "tok" }},
		{"service account", ServiceAccountKey("key"), func(r *http.Request) bool { return r.Header.Get("x-service-account-access-key") == "key" }},
		{"cookie", SessionCookie("sid"), func(r *http.Request) bool {
			ck, err := r.Cookie("connect.sid")
			return err == nil && ck.Value == "sid"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := false
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ok = tt.check(r)
				fmt.Fprint(w, `{"isGuest":false,"name":"me"}`)
			}), WithAuthenticator(tt.auth))
			if _, err := c.Users.Me(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Error("credential was not sent")
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	tests := []struct {
		status int
		target error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				fmt.Fprint(w, `{"name":"SomeError","message":"boom"}`)
			}))
			_, err := c.Projects.Get(context.Background(), "proj")
			if !errors.Is(err, tt.target) {
				t.Fatalf("errors.Is(%v, %v) = false", err, tt.target)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Name != "SomeError" || apiErr.Message != "boom" {
				t.Errorf("unexpected APIError: %+v", apiErr)
			}
		})
	}
}

func TestPagesListAll_Paginates(t *testing.T) {
	const total = 5
	var requests int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		fmt.Fprintf(w, `{"count":%d,"pages":[`, total)
		for i := skip; i < min(skip+limit, total); i++ {
			if i > skip {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"title":"p%d"}`, i)
		}
		fmt.Fprint(w, `]}`)
	}))

	var titles []string
	for p, err := range c.Pages.ListAll(context.Background(), "proj", &ListOptions{Limit: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		titles = append(titles, p.Title)
	}
	if len(titles) != total {
		t.Errorf("got %d pages, want %d: %v", len(titles), total, titles)
	}
	if requests != 3 {
		t.Errorf("requests = %d, want 3", requests)
	}
}

func TestPagesTitles_FollowsHeader(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("followingId") {
		case "":
			w.Header().Set("X-Following-Id", "id2")
			fmt.Fprint(w, `[{"id":"id1","title":"a"},{"id":"id2","title":"b"}]`)
		case "id2":
			fmt.Fprint(w, `[{"id":"id3","title":"c"}]`)
		default:
			t.Errorf("unexpected followingId %q", r.URL.Query().Get("followingId"))
		}
	}))

	var titles []string
	for e, err := range c.Pages.Titles(context.Background(), "proj") {
		if err != nil {
			t.Fatal(err)
		}
		titles = append(titles, e.Title)
	}
	if fmt.Sprint(titles) != "[a b c]" {
		t.Errorf("titles = %v", titles)
	}
}

func TestPagesSearch_ClampsLimit(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "go cli" {
			t.Errorf("q = %q", got)
		}
		fmt.Fprint(w, `{"count":3,"pages":[{"title":"a"},{"title":"b"},{"title":"c"}]}`)
	}))
	res, err := c.Pages.Search(context.Background(), "proj", "go cli", &SearchOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pages) != 2 {
		t.Errorf("len(pages) = %d, want 2", len(res.Pages))
	}
}

func TestSmartContextExport(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/api/smart-context/export-2hop-links/proj.txt"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		fmt.Fprint(w, "<PageList>"+r.URL.Query().Get("title"))
	}))
	text, err := c.SmartContext.Export(context.Background(), "proj", "a b", TwoHop)
	if err != nil {
		t.Fatal(err)
	}
	if text != "<PageList>a b" {
		t.Errorf("text = %q", text)
	}
	if _, err := c.SmartContext.Export(context.Background(), "proj", "a", Hops(3)); err == nil {
		t.Error("expected error for invalid hops")
	}
}

func TestParsePageURL(t *testing.T) {
	tests := []struct {
		in      string
		want    PageRef
		wantErr bool
	}{
		{in: "https://scrapbox.io/proj/Title", want: PageRef{"proj", "Title"}},
		{in: "https://scrapbox.io/proj/a%2Fb", want: PageRef{"proj", "a/b"}},
		{in: "https://scrapbox.io/proj/%E6%97%A5%E6%9C%AC", want: PageRef{"proj", "日本"}},
		{in: "https://scrapbox.io/proj/", wantErr: true},
		{in: "ftp://scrapbox.io/proj/a", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParsePageURL(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
