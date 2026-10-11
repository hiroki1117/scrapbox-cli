package scrapbox

import "fmt"

// Hops is the range of related pages, used by related pages and Smart Context.
type Hops int

const (
	// OneHop is the pages linked to or from the page.
	OneHop Hops = 1
	// TwoHop is the pages sharing a link with the 1-hop pages.
	TwoHop Hops = 2
)

// Validate reports an error if the API does not accept h.
func (h Hops) Validate() error {
	if h != OneHop && h != TwoHop {
		return fmt.Errorf("hops must be 1 or 2, got %d", h)
	}
	return nil
}

// UserRef is a user reference embedded in pages.
type UserRef struct {
	ID string `json:"id"`
}

// PageSummary is a page in a page list.
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

// Pinned reports whether the page is pinned.
func (p PageSummary) Pinned() bool { return p.Pin != 0 }

// PageList is the response of GET /api/pages/:project.
type PageList struct {
	ProjectName string        `json:"projectName"`
	Skip        int           `json:"skip"`
	Limit       int           `json:"limit"`
	Count       int           `json:"count"`
	Pages       []PageSummary `json:"pages"`
}

// Line is a line of a page. The first line is the title.
type Line struct {
	ID      string    `json:"id"`
	Text    string    `json:"text"`
	UserID  string    `json:"userId"`
	Created Timestamp `json:"created"`
	Updated Timestamp `json:"updated"`
}

// Page is the response of GET /api/pages/:project/:title.
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

// RelatedPages is the related page list of a page.
type RelatedPages struct {
	Links1Hop           []RelatedPage `json:"links1hop"`
	Links2Hop           []RelatedPage `json:"links2hop"`
	ProjectLinks1Hop    []RelatedPage `json:"projectLinks1hop"`
	HasBackLinksOrIcons bool          `json:"hasBackLinksOrIcons"`
}

// RelatedPage is an item of a related page list.
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

// SearchResult is the response of a full-text search.
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

// ParsedQuery is the search query as parsed by the server.
type ParsedQuery struct {
	Words    []string `json:"words"`
	Excludes []string `json:"excludes"`
}

// SearchHitPage is a page found by search. Lines are the matched lines.
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

// TitleEntry is an item of GET /api/pages/:project/search/titles.
type TitleEntry struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Links   []string  `json:"links"`
	Image   string    `json:"image,omitempty"`
	Updated Timestamp `json:"updated"`
}

// Project is the response of GET /api/projects/:project.
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

// ProjectList is the response of GET /api/projects.
type ProjectList struct {
	Projects []Project `json:"projects"`
}

// User is the response of GET /api/users/me. For guests, only IsGuest is set.
// Secrets such as csrfToken are deliberately left out.
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

// VectorSearchResult is the response of a vector search.
type VectorSearchResult struct {
	Pages []VectorHit `json:"pages"`
}

// VectorHit is a page found by vector search.
// Empty pages (Exists is false; they exist only as links) have no ID or timestamps.
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

// RelatedResult is the result of fetching related pages.
type RelatedResult struct {
	Hops       Hops          `json:"hops"`
	Pages      []RelatedPage `json:"pages"`
	Pagination Pagination    `json:"pagination"`
}

// Pagination is the paging info of related pages.
type Pagination struct {
	PerPage int `json:"perPage"`
	// Total is the number of related pages before filtering by search words.
	Total   int    `json:"total"`
	HasNext bool   `json:"hasNext"`
	NextID  string `json:"nextId,omitempty"`
}
