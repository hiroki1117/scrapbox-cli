package scrapbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// End is the insert anchor for the end of a page.
const End = "_end"

// Change is a single edit of a page. Create one with InsertBefore, Replace, DeleteLine or DeletePage.
type Change struct {
	Insert  string      `json:"_insert,omitempty"`
	Update  string      `json:"_update,omitempty"`
	Delete  string      `json:"_delete,omitempty"`
	Lines   *ChangeLine `json:"lines,omitempty"`
	Deleted bool        `json:"deleted,omitempty"`
}

// ChangeLine is the line content of a Change.
type ChangeLine struct {
	ID   string `json:"id,omitempty"`
	Text string `json:"text"`
}

// InsertBefore inserts a new line before the line with ID anchor, or at the end if anchor is End.
// Inserts with the same anchor keep their order.
func InsertBefore(anchor, text string) Change {
	return Change{Insert: anchor, Lines: &ChangeLine{ID: newLineID(), Text: text}}
}

// Replace replaces the text of a line. Replacing the first line renames the page.
func Replace(lineID, text string) Change {
	return Change{Update: lineID, Lines: &ChangeLine{Text: text}}
}

// DeleteLine deletes a line.
func DeleteLine(lineID string) Change {
	return Change{Delete: lineID}
}

// DeletePage deletes the whole page.
func DeletePage() Change {
	return Change{Deleted: true}
}

func newLineID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// EditPreview is the result of a dry-run edit. Pass PreviewID to SubmitEdit within 5 minutes to apply it.
type EditPreview struct {
	PreviewID    string       `json:"previewId"`
	ExpireAt     string       `json:"expireAt"`
	Page         *PagePreview `json:"pagePreview"`
	PageDelete   bool         `json:"pageDelete,omitempty"`
	ChangedLinks []string     `json:"changedLinks,omitempty"`
}

// PagePreview is a page as it will be after the edit.
type PagePreview struct {
	Title      string        `json:"title"`
	Persistent bool          `json:"persistent"`
	CommitID   string        `json:"commitId,omitempty"`
	Lines      []PreviewLine `json:"lines"`
}

// PreviewLine is a line of a PagePreview.
type PreviewLine struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// EditResult is the result of SubmitEdit. Page is set for created or updated pages, PageDeleted for deleted ones.
type EditResult struct {
	CommitID     string        `json:"commitId"`
	Page         *EditedPage   `json:"page,omitempty"`
	PageDeleted  *EditedPage   `json:"pageDeleted,omitempty"`
	TitleChanged *TitleChanged `json:"titleChanged,omitempty"`
}

// EditedPage is the page written by SubmitEdit.
type EditedPage struct {
	Title string `json:"title"`
}

// TitleChanged is set when an edit renamed the page.
type TitleChanged struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PreviewEdit validates changes without applying them. An empty pageID creates a new page,
// whose first inserted line is the title. Writes need a Personal Access Token or Service Account.
//
// POST /api/pages/v2/:project/page-edit-for-ai/preview
func (s *PagesService) PreviewEdit(ctx context.Context, project, pageID string, changes []Change) (*EditPreview, error) {
	body := struct {
		PageID  string   `json:"pageId,omitempty"`
		Changes []Change `json:"changes"`
	}{pageID, changes}
	var out EditPreview
	if err := s.client.postJSON(ctx, "pages/v2/"+escapeSegment(project)+"/page-edit-for-ai/preview", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SubmitEdit applies a preview. It fails with ErrConflict if the page changed after the preview.
// If a page with the same title exists, the server adds a suffix to the new title.
//
// POST /api/pages/v2/:project/page-edit-for-ai/submit
func (s *PagesService) SubmitEdit(ctx context.Context, project, previewID string) (*EditResult, error) {
	body := struct {
		PreviewID string `json:"previewId"`
	}{previewID}
	var out EditResult
	if err := s.client.postJSON(ctx, "pages/v2/"+escapeSegment(project)+"/page-edit-for-ai/submit", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplaceLinks rewrites links, hashtags and icons pointing to from so they point to to,
// across the project. It returns the server message, such as "3 pages have been successfully updated!".
//
// POST /api/pages/:project/replace/links
func (s *PagesService) ReplaceLinks(ctx context.Context, project, from, to string) (string, error) {
	body := struct {
		From string `json:"from"`
		To   string `json:"to"`
	}{from, to}
	var out struct {
		Message string `json:"message"`
	}
	if err := s.client.postJSON(ctx, "pages/"+escapeSegment(project)+"/replace/links", body, &out); err != nil {
		return "", err
	}
	return out.Message, nil
}
