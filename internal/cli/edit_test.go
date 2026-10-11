package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

// fakeEditServer serves page "T" (lines T, alpha, tpyo beta, gamma) and applies previewed changes.
type fakeEditServer struct {
	t         *testing.T
	changes   []scrapbox.Change
	submitted bool
}

var fakeLines = []scrapbox.Line{{ID: "l1", Text: "T"}, {ID: "l2", Text: "alpha"}, {ID: "l3", Text: "tpyo beta"}, {ID: "l4", Text: "gamma"}}

func (s *fakeEditServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/pages/proj/T":
		_ = json.NewEncoder(w).Encode(scrapbox.Page{PageSummary: scrapbox.PageSummary{ID: "p1", Title: "T"}, Persistent: true, Lines: fakeLines})
	case "/api/pages/v2/proj/page-edit-for-ai/preview":
		var body struct{ Changes []scrapbox.Change }
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.changes = body.Changes
		deleted := len(body.Changes) == 1 && body.Changes[0].Deleted
		_ = json.NewEncoder(w).Encode(scrapbox.EditPreview{PreviewID: "pv", PageDelete: deleted, Page: &scrapbox.PagePreview{Lines: applyChanges(fakeLines, body.Changes)}})
	case "/api/pages/v2/proj/page-edit-for-ai/submit":
		s.submitted = true
		fmt.Fprint(w, `{"commitId":"c1","page":{"title":"T"}}`)
	default:
		http.NotFound(w, r)
	}
}

func applyChanges(lines []scrapbox.Line, changes []scrapbox.Change) []scrapbox.PreviewLine {
	var out []scrapbox.PreviewLine
	for _, l := range lines {
		out = append(out, scrapbox.PreviewLine{ID: l.ID, Text: l.Text})
	}
	for _, c := range changes {
		i := slices.IndexFunc(out, func(l scrapbox.PreviewLine) bool { return l.ID == c.Insert+c.Update+c.Delete })
		switch {
		case c.Insert == scrapbox.End:
			out = append(out, scrapbox.PreviewLine{ID: c.Lines.ID, Text: c.Lines.Text})
		case c.Insert != "":
			out = slices.Insert(out, i, scrapbox.PreviewLine{ID: c.Lines.ID, Text: c.Lines.Text})
		case c.Update != "":
			out[i].Text = c.Lines.Text
		case c.Delete != "":
			out = slices.Delete(out, i, i+1)
		}
	}
	return out
}

func runEdit(t *testing.T, args ...string) (*fakeEditServer, string, error) {
	t.Helper()
	s := &fakeEditServer{t: t}
	out, err := runCmd(t, s, map[string]string{config.EnvProject: "proj"}, args...)
	return s, out, err
}

func TestPageEditCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string // the page after the edit, one line per entry
	}{
		{"append", []string{"page", "append", "T", "-t", "x\ny"}, "T alpha tpyo-beta gamma x y"},
		{"insert after", []string{"page", "insert", "T", "--after", "2", "-t", "x"}, "T alpha x tpyo-beta gamma"},
		{"insert after last", []string{"page", "insert", "T", "--after", "4", "-t", "x"}, "T alpha tpyo-beta gamma x"},
		{"insert before", []string{"page", "insert", "T", "--before", "2", "-t", "x"}, "T x alpha tpyo-beta gamma"},
		{"replace", []string{"page", "replace", "T", "tpyo", "typo"}, "T alpha typo-beta gamma"},
		{"delete lines", []string{"page", "delete-lines", "T", "2-3"}, "T gamma"},
		{"rename", []string{"page", "rename", "T", "U"}, "U alpha tpyo-beta gamma"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, out, err := runEdit(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, l := range applyChanges(fakeLines, s.changes) {
				got = append(got, strings.ReplaceAll(l.Text, " ", "-"))
			}
			if strings.Join(got, " ") != tt.want || !s.submitted || !strings.Contains(out, "/proj/T") {
				t.Errorf("page = %v, submitted = %v, out = %q", got, s.submitted, out)
			}
		})
	}
}

func TestPageEdit_DryRunShowsDiff(t *testing.T) {
	s, out, err := runEdit(t, "page", "replace", "T", "tpyo", "typo", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if s.submitted || out != "  T\n  alpha\n~ typo beta\n  gamma\n" {
		t.Errorf("submitted = %v, out = %q", s.submitted, out)
	}
	_, out, _ = runEdit(t, "page", "delete-lines", "T", "2", "--dry-run")
	if out != "  T\n- alpha\n  tpyo beta\n  gamma\n" {
		t.Errorf("out = %q", out)
	}
}

func TestPageEdit_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"ambiguous replace", []string{"page", "replace", "T", "a", "b"}, "occurs"},
		{"replace not found", []string{"page", "replace", "T", "zzz", "b"}, "not found"},
		{"delete title", []string{"page", "delete-lines", "T", "1"}, "title"},
		{"line out of range", []string{"page", "insert", "T", "--after", "9", "-t", "x"}, "out of range"},
		{"insert before title", []string{"page", "insert", "T", "--before", "1", "-t", "x"}, "title"},
		{"no position", []string{"page", "insert", "T", "-t", "x"}, "--after or --before"},
		{"no text", []string{"page", "append", "T"}, "no text"},
		{"missing page", []string{"page", "append", "nope", "-t", "x"}, "does not exist"},
		{"create existing", []string{"page", "create", "T"}, "already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _, err := runEdit(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) || s.submitted {
				t.Errorf("err = %v, want %q (submitted = %v)", err, tt.want, s.submitted)
			}
		})
	}
}

func TestPageCreate(t *testing.T) {
	s, out, err := runEdit(t, "page", "create", "New", "-t", "body")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.changes) != 2 || s.changes[0].Lines.Text != "New" || s.changes[1].Insert != scrapbox.End || !strings.HasPrefix(out, "Created ") {
		t.Errorf("changes = %+v, out = %q", s.changes, out)
	}
}

func TestPageDelete_RequiresYes(t *testing.T) {
	s, out, err := runEdit(t, "page", "delete", "T")
	if err != nil || s.submitted || !strings.Contains(out, "- alpha") {
		t.Errorf("without --yes: err = %v, submitted = %v, out = %q", err, s.submitted, out)
	}
	if s, _, err := runEdit(t, "page", "delete", "T", "--yes"); err != nil || !s.submitted || !s.changes[0].Deleted {
		t.Errorf("with --yes: err = %v, submitted = %v", err, s.submitted)
	}
}
