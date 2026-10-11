package scrapbox

import (
	"context"
	"net/url"
	"strconv"
)

// SearchOptions は全文検索のオプションである。
type SearchOptions struct {
	// Limit は返すページ数の上限である。サーバーが limit を無視する場合があるため、クライアント側でも切り詰める。
	Limit int
	// Or が true なら、いずれかの語を含むページを返す。既定は AND 検索である。
	Or bool
}

// Search はプロジェクト内を全文検索する。
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

// SearchVector は意味の近さでページを検索する。結果は類似度の高い順に並ぶ。
// 検索対象はページタイトルと本文中のリンク記法だけで、本文の通常テキストは対象外である。
// リンクだけが存在する空ページもヒットする (Exists が false)。
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

// RelatedOptions は関連ページ取得のオプションである。
type RelatedOptions struct {
	// Search を指定すると、本文に Search を含む関連ページだけに絞り込む (2 hop search)。
	Search string
	// Or が true なら、Search のいずれかの語を含むページを返す。既定は AND である。
	Or bool
}

// Related はページの関連ページリスト (1 hop / 2 hop link) を取得する。
// 起点ページが存在しなくても、被リンクがあれば結果を返す。
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
	// レスポンスのキーが hops によって links1hop / links2hop と異なるため、両方で受けてまとめる。
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
