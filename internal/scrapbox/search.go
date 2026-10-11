package scrapbox

import (
	"context"
	"net/url"
	"strconv"
)

// SearchOptions are options for full-text search.
type SearchOptions struct {
	// Limit is the maximum number of pages. The server may ignore it, so results are also truncated here.
	Limit int
	// Or matches pages containing any of the words. The default is AND.
	Or bool
}

// Search runs a full-text search in a project.
//
// GET /api/pages/:project/search/query?q=
func (s *PagesService) Search(ctx context.Context, project, query string, opts *SearchOptions) (*SearchResult, error) {
	q := url.Values{"q": {query}}
	if opts != nil {
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Or {
			q.Set("op", "or")
		}
	}
	var out SearchResult
	if _, err := s.client.getJSON(ctx, "pages/"+escapeSegment(project)+"/search/query", q, &out); err != nil {
		return nil, err
	}
	if opts != nil && opts.Limit > 0 && len(out.Pages) > opts.Limit {
		out.Pages = out.Pages[:opts.Limit]
	}
	return &out, nil
}

// SearchVector finds pages by meaning, ordered by similarity.
// Only page titles and links are searched, not plain body text.
// Empty pages that exist only as links are included (Exists is false).
//
// GET /api/pages/:project/search/vector/titles?q=
func (s *PagesService) SearchVector(ctx context.Context, project, query string) (*VectorSearchResult, error) {
	var out VectorSearchResult
	path := "pages/" + escapeSegment(project) + "/search/vector/titles"
	if _, err := s.client.getJSON(ctx, path, url.Values{"q": {query}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RelatedOptions are options for fetching related pages.
type RelatedOptions struct {
	// Search keeps only related pages containing it (2 hop search).
	Search string
	// Or matches pages containing any of the words in Search. The default is AND.
	Or bool
}

// Related fetches the related pages (1-hop or 2-hop links) of a page.
// It returns results even if the page does not exist, as long as something links to it.
//
// GET /api/pages/v2/:project/:title/links1hop
// GET /api/pages/v2/:project/:title/links2hop
func (s *PagesService) Related(ctx context.Context, project, title string, hops Hops, opts *RelatedOptions) (*RelatedResult, error) {
	if err := hops.Validate(); err != nil {
		return nil, err
	}
	q := url.Values{}
	if opts != nil {
		if opts.Search != "" {
			q.Set("search", opts.Search)
		}
		if opts.Or {
			q.Set("op", "or")
		}
	}
	// The response key is links1hop or links2hop depending on hops.
	var raw struct {
		Links1Hop  []RelatedPage `json:"links1hop"`
		Links2Hop  []RelatedPage `json:"links2hop"`
		Pagination Pagination    `json:"pagination"`
	}
	path := "pages/v2/" + escapeSegment(project) + "/" + escapeTitle(title) + "/links" + strconv.Itoa(int(hops)) + "hop"
	if _, err := s.client.getJSON(ctx, path, q, &raw); err != nil {
		return nil, err
	}
	pages := raw.Links1Hop
	if hops == TwoHop {
		pages = raw.Links2Hop
	}
	if pages == nil {
		pages = []RelatedPage{}
	}
	return &RelatedResult{Hops: hops, Pages: pages, Pagination: raw.Pagination}, nil
}
