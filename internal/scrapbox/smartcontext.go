package scrapbox

import (
	"context"
	"fmt"
	"net/url"
)

// SmartContextService は Smart Context (UI 名: Export for AI) API を扱う。
type SmartContextService service

// Hops は Smart Context に含める関連ページの範囲である。
type Hops int

const (
	// OneHop は起点ページと 1 hop link のページを含める。
	OneHop Hops = 1
	// TwoHop は 1 hop に加えて 2 hop link のページも含める。レスポンスが 1MB を超えることがある。
	TwoHop Hops = 2
)

// Validate は Hops が API で扱える値かを検証する。
func (h Hops) Validate() error {
	if h != OneHop && h != TwoHop {
		return fmt.Errorf("hops must be 1 or 2, got %d", h)
	}
	return nil
}

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
