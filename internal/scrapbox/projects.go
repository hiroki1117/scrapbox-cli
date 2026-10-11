package scrapbox

import "context"

// ProjectsService はプロジェクトに関する API を扱う。
type ProjectsService service

// Get はプロジェクトの情報を取得する。
//
// GET /api/projects/:project
func (s *ProjectsService) Get(ctx context.Context, project string) (*Project, error) {
	var out Project
	if _, err := s.client.getJSON(ctx, "projects/"+escapeSegment(project), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List は認証ユーザーが参加しているプロジェクトの一覧を取得する。
//
// GET /api/projects
func (s *ProjectsService) List(ctx context.Context) (*ProjectList, error) {
	var out ProjectList
	if _, err := s.client.getJSON(ctx, "projects", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
