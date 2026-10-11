package scrapbox

import (
	"fmt"
	"net/url"
	"strings"
)

// escapeTitle escapes a page title as a URL path segment.
// Spaces become "_", as Scrapbox does.
func escapeTitle(title string) string {
	return escapeSegment(strings.ReplaceAll(title, " ", "_"))
}

// escapeSegment escapes s as a single path segment.
// "." and ".." are escaped so path cleaning does not remove them.
func escapeSegment(s string) string {
	switch s {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	}
	return url.PathEscape(s)
}

// PageRef is a project name and a page title.
type PageRef struct {
	Project string
	Title   string
}

// ParsePageURL parses a URL like https://scrapbox.io/<project>/<title>.
// A "/" in the title, escaped or not, is part of the title.
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

// IsURL reports whether s looks like an http(s) URL.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}

// PageURL returns the browser URL of a page, derived from the API base URL.
func (c *Client) PageURL(project, title string) string {
	return strings.TrimSuffix(c.baseURL, "/api") + "/" + escapeSegment(project) + "/" + escapeTitle(title)
}
