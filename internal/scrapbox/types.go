package scrapbox

import "fmt"

// Hops は関連ページの範囲である。関連ページリストと Smart Context で使う。
type Hops int

const (
	// OneHop は起点ページとリンクでつながるページである。
	OneHop Hops = 1
	// TwoHop は 1 hop のページとリンクを共有するページである。
	TwoHop Hops = 2
)

// Validate は Hops が API で扱える値かを検証する。
func (h Hops) Validate() error {
	if h != OneHop && h != TwoHop {
		return fmt.Errorf("hops must be 1 or 2, got %d", h)
	}
	return nil
}

// UserRef はページなどに埋め込まれるユーザー参照である。
type UserRef struct {
	ID string `json:"id"`
}

// PageSummary はページ一覧に含まれるページの概要である。
type PageSummary struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Image          string    `json:"image,omitempty"`
	Descriptions   []string  `json:"descriptions"`
	User           UserRef   `json:"user"`
	LastUpdateUser UserRef   `json:"lastUpdateUser"`
	Pin            int64     `json:"pin"`
	Views          int       `json:"views"`
	Linked         int       `json:"linked"`
	Created        Timestamp `json:"created"`
	Updated        Timestamp `json:"updated"`
	Accessed       Timestamp `json:"accessed"`
	LinesCount     int       `json:"linesCount"`
	CharsCount     int       `json:"charsCount"`
}

// Pinned はページがピン留めされているかを返す。
func (p PageSummary) Pinned() bool { return p.Pin != 0 }

// PageList は GET /api/pages/:project のレスポンスである。
type PageList struct {
	ProjectName string        `json:"projectName"`
	Skip        int           `json:"skip"`
	Limit       int           `json:"limit"`
	Count       int           `json:"count"`
	Pages       []PageSummary `json:"pages"`
}

// Line はページの 1 行である。先頭行はタイトルである。
type Line struct {
	ID      string    `json:"id"`
	Text    string    `json:"text"`
	UserID  string    `json:"userId"`
	Created Timestamp `json:"created"`
	Updated Timestamp `json:"updated"`
}

// Page は GET /api/pages/:project/:title のレスポンスである。
type Page struct {
	PageSummary
	CommitID     string       `json:"commitId"`
	PageRank     float64      `json:"pageRank"`
	Persistent   bool         `json:"persistent"`
	Lines        []Line       `json:"lines"`
	Links        []string     `json:"links"`
	ProjectLinks []string     `json:"projectLinks"`
	Icons        []string     `json:"icons"`
	Files        []string     `json:"files"`
	RelatedPages RelatedPages `json:"relatedPages"`
}

// RelatedPages はページの関連ページリストである。
type RelatedPages struct {
	Links1Hop           []RelatedPage `json:"links1hop"`
	Links2Hop           []RelatedPage `json:"links2hop"`
	ProjectLinks1Hop    []RelatedPage `json:"projectLinks1hop"`
	HasBackLinksOrIcons bool          `json:"hasBackLinksOrIcons"`
}

// RelatedPage は関連ページリストの要素である。
type RelatedPage struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	TitleLc      string    `json:"titleLc"`
	Image        string    `json:"image,omitempty"`
	Descriptions []string  `json:"descriptions"`
	LinksLc      []string  `json:"linksLc"`
	Linked       int       `json:"linked"`
	PageRank     float64   `json:"pageRank"`
	Created      Timestamp `json:"created"`
	Updated      Timestamp `json:"updated"`
	Accessed     Timestamp `json:"accessed"`
}

// SearchResult は全文検索のレスポンスである。
type SearchResult struct {
	ProjectName           string          `json:"projectName"`
	SearchQuery           string          `json:"searchQuery"`
	Limit                 int             `json:"limit"`
	Count                 int             `json:"count"`
	ExistsExactTitleMatch bool            `json:"existsExactTitleMatch"`
	Field                 string          `json:"field"`
	Query                 ParsedQuery     `json:"query"`
	Backend               string          `json:"backend"`
	Pages                 []SearchHitPage `json:"pages"`
}

// ParsedQuery はサーバーが解釈した検索クエリである。
type ParsedQuery struct {
	Words    []string `json:"words"`
	Excludes []string `json:"excludes"`
}

// SearchHitPage は検索でヒットしたページである。Lines はヒットした行の抜粋である。
type SearchHitPage struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Image   string    `json:"image,omitempty"`
	Words   []string  `json:"words"`
	Lines   []string  `json:"lines"`
	Views   int       `json:"views"`
	Linked  int       `json:"linked"`
	Created Timestamp `json:"created"`
	Updated Timestamp `json:"updated"`
}

// TitleEntry は GET /api/pages/:project/search/titles の要素である。
type TitleEntry struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Links   []string  `json:"links"`
	Image   string    `json:"image,omitempty"`
	Updated Timestamp `json:"updated"`
}

// Project は GET /api/projects/:project のレスポンスである。
type Project struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"displayName"`
	PublicVisible  bool      `json:"publicVisible"`
	Theme          string    `json:"theme"`
	Image          string    `json:"image,omitempty"`
	GyazoTeamsName string    `json:"gyazoTeamsName,omitempty"`
	Translation    bool      `json:"translation"`
	Infobox        bool      `json:"infobox"`
	IsMember       bool      `json:"isMember"`
	Plan           string    `json:"plan,omitempty"`
	Created        Timestamp `json:"created"`
	Updated        Timestamp `json:"updated"`
}

// ProjectList は GET /api/projects のレスポンスである。
type ProjectList struct {
	Projects []Project `json:"projects"`
}

// User は GET /api/users/me のレスポンスである。未ログイン時は IsGuest のみ true になる。
// csrfToken などの秘匿値は意図的にフィールドに含めない。
type User struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	Email       string    `json:"email,omitempty"`
	Photo       string    `json:"photo,omitempty"`
	IsGuest     bool      `json:"isGuest"`
	Created     Timestamp `json:"created,omitempty"`
	Updated     Timestamp `json:"updated,omitempty"`
}

// VectorSearchResult はベクトル検索のレスポンスである。
type VectorSearchResult struct {
	Pages []VectorHit `json:"pages"`
}

// VectorHit はベクトル検索でヒットしたページである。
// Exists が false の空ページ (リンクだけが存在するページ) は ID や更新日時を持たない。
type VectorHit struct {
	ID         string    `json:"id,omitempty"`
	Title      string    `json:"title"`
	Image      string    `json:"image,omitempty"`
	Score      float64   `json:"score"`
	Linked     int       `json:"linked"`
	Exists     bool      `json:"exists"`
	Views      int       `json:"views,omitempty"`
	PageRank   float64   `json:"pageRank,omitempty"`
	LinesCount int       `json:"linesCount,omitempty"`
	CharsCount int       `json:"charsCount,omitempty"`
	Created    Timestamp `json:"created,omitempty"`
	Updated    Timestamp `json:"updated,omitempty"`
}

// RelatedResult は関連ページリストの取得結果である。
type RelatedResult struct {
	Hops       Hops          `json:"hops"`
	Pages      []RelatedPage `json:"pages"`
	Pagination Pagination    `json:"pagination"`
}

// Pagination は関連ページリストのページング情報である。
type Pagination struct {
	PerPage int `json:"perPage"`
	// Total は検索語で絞り込む前の関連ページ数である。
	Total   int    `json:"total"`
	HasNext bool   `json:"hasNext"`
	NextID  string `json:"nextId,omitempty"`
}
