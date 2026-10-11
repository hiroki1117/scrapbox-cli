package scrapbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPreviewAndSubmitEdit(t *testing.T) {
	var bodies []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("method = %s, content type = %q", r.Method, r.Header.Get("Content-Type"))
		}
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		switch r.URL.Path {
		case "/api/pages/v2/proj/page-edit-for-ai/preview":
			fmt.Fprint(w, `{"previewId":"pv1","pagePreview":{"title":"T","lines":[]}}`)
		case "/api/pages/v2/proj/page-edit-for-ai/submit":
			fmt.Fprint(w, `{"commitId":"c1","page":{"title":"T"}}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	ctx := context.Background()

	changes := []Change{InsertBefore("l1", "new"), Replace("l2", "fixed"), DeleteLine("l3")}
	p, err := c.Pages.PreviewEdit(ctx, "proj", "page1", changes)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		PageID  string           `json:"pageId"`
		Changes []map[string]any `json:"changes"`
	}
	if err := json.Unmarshal([]byte(bodies[0]), &got); err != nil {
		t.Fatal(err)
	}
	ins, upd, del := got.Changes[0], got.Changes[1], got.Changes[2]
	if got.PageID != "page1" || ins["_insert"] != "l1" || ins["lines"].(map[string]any)["id"] == "" ||
		upd["_update"] != "l2" || upd["lines"].(map[string]any)["text"] != "fixed" ||
		del["_delete"] != "l3" || len(del) != 1 {
		t.Errorf("preview body = %s", bodies[0])
	}

	res, err := c.Pages.SubmitEdit(ctx, "proj", p.PreviewID)
	if err != nil {
		t.Fatal(err)
	}
	if bodies[1] != `{"previewId":"pv1"}` || res.Page.Title != "T" {
		t.Errorf("submit body = %s, result = %+v", bodies[1], res)
	}
}

func TestPreviewEdit_NewPageAndDelete(t *testing.T) {
	var body string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		fmt.Fprint(w, `{"previewId":"pv"}`)
	}))
	ctx := context.Background()

	if _, err := c.Pages.PreviewEdit(ctx, "proj", "", []Change{InsertBefore(End, "Title")}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "pageId") || !strings.Contains(body, `"_insert":"_end"`) {
		t.Errorf("new page body = %s", body)
	}
	if _, err := c.Pages.PreviewEdit(ctx, "proj", "p", []Change{DeletePage()}); err != nil {
		t.Fatal(err)
	}
	if body != `{"pageId":"p","changes":[{"deleted":true}]}` {
		t.Errorf("delete body = %s", body)
	}
}

func TestWrite_NotRetriedAndConflict(t *testing.T) {
	var requests int
	status := http.StatusServiceUnavailable
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(status)
		fmt.Fprint(w, `{"error":"NotFastForward","latest":null}`)
	}), fastRetry)
	ctx := context.Background()

	if _, err := c.Pages.SubmitEdit(ctx, "proj", "pv"); err == nil || requests != 1 {
		t.Errorf("err = %v, requests = %d; writes must not be retried", err, requests)
	}

	status = http.StatusConflict
	_, err := c.Pages.SubmitEdit(ctx, "proj", "pv")
	var apiErr *APIError
	if !errors.Is(err, ErrConflict) || !errors.As(err, &apiErr) || apiErr.Name != "NotFastForward" {
		t.Errorf("err = %v, want ErrConflict with name NotFastForward", err)
	}
}

func TestReplaceLinks(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/api/pages/proj/replace/links" || string(b) != `{"from":"a","to":"b"}` {
			t.Errorf("path = %s, body = %s", r.URL.Path, b)
		}
		fmt.Fprint(w, `{"message":"1 pages have been successfully updated!"}`)
	}))
	msg, err := c.Pages.ReplaceLinks(context.Background(), "proj", "a", "b")
	if err != nil || !strings.HasPrefix(msg, "1 pages") {
		t.Errorf("msg = %q, err = %v", msg, err)
	}
}

func TestPageURL(t *testing.T) {
	c := newTestClient(t, http.NotFoundHandler())
	if got := c.PageURL("proj", "a b"); !strings.HasSuffix(got, "/proj/a_b") || strings.Contains(got, "/api/") {
		t.Errorf("PageURL = %q", got)
	}
}
