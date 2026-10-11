package scrapbox

import (
	"context"
	"fmt"
	"net/url"
)

// SmartContextService handles the Smart Context ("Export for AI") API.
type SmartContextService service

// Export returns a page and its related pages as text formatted for LLMs.
//
// GET /api/smart-context/export-:hopshop-links/:project.txt?title=
func (s *SmartContextService) Export(ctx context.Context, project, title string, hops Hops) (string, error) {
	if err := hops.Validate(); err != nil {
		return "", err
	}
	path := fmt.Sprintf("smart-context/export-%dhop-links/%s", hops, escapeSegment(project+".txt"))
	return s.client.getText(ctx, path, url.Values{"title": {title}})
}
