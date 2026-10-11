package scrapbox

import (
	"context"
	"iter"
	"net/url"
	"strconv"
)

// PagesService はページに関する API を扱う。
type PagesService service

// MaxListLimit は 1 回のページ一覧取得で指定できる件数の上限である。
const MaxListLimit = 1000

// ListOptions はページ一覧取得のオプションである。ゼロ値はサーバーのデフォルトを使う。
type ListOptions struct {
	Limit int
	Skip  int
	// Sort はソートキーである (updated, created, accessed, linked, views, title など)。
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

// List はプロジェクトのページ一覧を取得する。
//
// GET /api/pages/:project
func (s *PagesService) List(ctx context.Context, project string, opts *ListOptions) (*PageList, error) {
	var out PageList
	if _, err := s.client.getJSON(ctx, "pages/"+escapeSegment(project), opts.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAll は opts.Skip から最後のページまで、ページングしながら全件を返す。
// opts.Limit は 1 リクエストあたりの件数として扱う。
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

// Get はページの本文とメタデータを取得する。
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

// Titles はプロジェクトの全ページのタイトルとリンクを返す。
// サーバーは X-Following-Id ヘッダで次のページを示すため、それに従ってページングする。
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

// Code はページ内のコードブロックの内容を取得する。filename はコードブロックの見出し (code:xxx の xxx) である。
//
// GET /api/code/:project/:title/:filename
func (s *PagesService) Code(ctx context.Context, project, title, filename string) (string, error) {
	path := "code/" + escapeSegment(project) + "/" + escapeTitle(title) + "/" + escapeSegment(filename)
	return s.client.getText(ctx, path, nil)
}

// Table はページ内のテーブル (table:xxx) を CSV として取得する。
//
// GET /api/table/:project/:title/:filename.csv
func (s *PagesService) Table(ctx context.Context, project, title, filename string) (string, error) {
	path := "table/" + escapeSegment(project) + "/" + escapeTitle(title) + "/" + escapeSegment(filename+".csv")
	return s.client.getText(ctx, path, nil)
}
