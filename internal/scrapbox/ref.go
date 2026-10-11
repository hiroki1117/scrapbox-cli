package scrapbox

import (
	"fmt"
	"net/url"
	"strings"
)

// escapeTitle はページタイトルを URL のパスセグメントにエスケープする。
// Scrapbox の慣習に従い、半角スペースは "_" に置換する。
func escapeTitle(title string) string {
	return escapeSegment(strings.ReplaceAll(title, " ", "_"))
}

// escapeSegment は文字列を 1 つのパスセグメントとしてエスケープする。
// "." と ".." はパス正規化で消えないようにエスケープする。
func escapeSegment(s string) string {
	switch s {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	}
	return url.PathEscape(s)
}

// PageRef はプロジェクト名とページタイトルの組である。
type PageRef struct {
	Project string
	Title   string
}

// ParsePageURL は https://scrapbox.io/<project>/<title> 形式の URL を PageRef に変換する。
// タイトル中の "/" (エスケープ済み・未エスケープの両方) はタイトルの一部として扱う。
func ParsePageURL(raw string) (PageRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return PageRef{}, fmt.Errorf("invalid page URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return PageRef{}, fmt.Errorf("invalid page URL %q: scheme must be http or https", raw)
	}
	project, title, ok := strings.Cut(strings.Trim(u.Path, "/"), "/")
	if !ok || project == "" || title == "" {
		return PageRef{}, fmt.Errorf("invalid page URL %q: must be https://<host>/<project>/<title>", raw)
	}
	return PageRef{Project: project, Title: title}, nil
}

// IsURL は文字列が http(s) の URL として解釈できるかを返す。
func IsURL(s string) bool {
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}
