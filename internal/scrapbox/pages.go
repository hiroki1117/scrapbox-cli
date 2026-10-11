package scrapbox

import (
	"context"
	"iter"
	"net/url"
	"strconv"
)

// PagesService handles page APIs.
type PagesService service

// MaxListLimit is the maximum number of pages per list request.
const MaxListLimit = 1000

// ListOptions are options for listing pages. Zero values use the server defaults.
type ListOptions struct {
	Limit int
	Skip  int
	// Sort is the sort key (updated, created, accessed, linked, views, title, ...).
	Sort string
}

func (o *ListOptions) values() url.Values {
	q := url.Values{}
	if o == nil {
		return q
	}
	if o.Limit > 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Skip > 0 {
		q.Set("skip", strconv.Itoa(o.Skip))
	}
	if o.Sort != "" {
		q.Set("sort", o.Sort)
	}
	return q
}

// List lists pages in a project.
//
// GET /api/pages/:project
func (s *PagesService) List(ctx context.Context, project string, opts *ListOptions) (*PageList, error) {
	var out PageList
	if _, err := s.client.getJSON(ctx, "pages/"+escapeSegment(project), opts.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAll yields all pages from opts.Skip to the end, paging automatically.
// opts.Limit is used as the number of pages per request.
func (s *PagesService) ListAll(ctx context.Context, project string, opts *ListOptions) iter.Seq2[PageSummary, error] {
	return func(yield func(PageSummary, error) bool) {
		o := ListOptions{Limit: MaxListLimit}
		if opts != nil {
			o = *opts
			if o.Limit <= 0 {
				o.Limit = MaxListLimit
			}
		}
		for {
			list, err := s.List(ctx, project, &o)
			if err != nil {
				yield(PageSummary{}, err)
				return
			}
			for _, p := range list.Pages {
				if !yield(p, nil) {
					return
				}
			}
			o.Skip += len(list.Pages)
			if len(list.Pages) == 0 || o.Skip >= list.Count {
				return
			}
		}
	}
}

// Get fetches the content and metadata of a page.
//
// GET /api/pages/:project/:title
func (s *PagesService) Get(ctx context.Context, project, title string) (*Page, error) {
	var out Page
	path := "pages/" + escapeSegment(project) + "/" + escapeTitle(title)
	if _, err := s.client.getJSON(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Titles yields the title and links of every page in a project.
// It pages by following the X-Following-Id response header.
//
// GET /api/pages/:project/search/titles
func (s *PagesService) Titles(ctx context.Context, project string) iter.Seq2[TitleEntry, error] {
	return func(yield func(TitleEntry, error) bool) {
		path := "pages/" + escapeSegment(project) + "/search/titles"
		followingID := ""
		for {
			q := url.Values{}
			if followingID != "" {
				q.Set("followingId", followingID)
			}
			var entries []TitleEntry
			header, err := s.client.getJSON(ctx, path, q, &entries)
			if err != nil {
				yield(TitleEntry{}, err)
				return
			}
			for _, e := range entries {
				if !yield(e, nil) {
					return
				}
			}
			next := header.Get("X-Following-Id")
			if next == "" || next == followingID || len(entries) == 0 {
				return
			}
			followingID = next
		}
	}
}

// Code fetches a code block in a page. filename is the xxx in code:xxx.
//
// GET /api/code/:project/:title/:filename
func (s *PagesService) Code(ctx context.Context, project, title, filename string) (string, error) {
	path := "code/" + escapeSegment(project) + "/" + escapeTitle(title) + "/" + escapeSegment(filename)
	return s.client.getText(ctx, path, nil)
}

// Table fetches a table (table:xxx) in a page as CSV.
//
// GET /api/table/:project/:title/:filename.csv
func (s *PagesService) Table(ctx context.Context, project, title, filename string) (string, error) {
	path := "table/" + escapeSegment(project) + "/" + escapeTitle(title) + "/" + escapeSegment(filename+".csv")
	return s.client.getText(ctx, path, nil)
}
