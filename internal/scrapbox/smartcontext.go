package scrapbox

import (
	"context"
	"fmt"
	"net/url"
)

// SmartContextService は Smart Context (UI 名: Export for AI) API を扱う。
type SmartContextService service

// Export は起点ページと関連ページの本文を LLM 向けに整形したテキストで返す。
//
// GET /api/smart-context/export-:hopshop-links/:project.txt?title=
func (s *SmartContextService) Export(ctx context.Context, project, title string, hops Hops) (string, error) {
	if err := hops.Validate(); err != nil {
		return "", err
	}
	path := fmt.Sprintf("smart-context/export-%dhop-links/%s", hops, escapeSegment(project+".txt"))
	return s.client.getText(ctx, path, url.Values{"title": {title}})
}
