package scrapbox

import "context"

// ProjectsService handles project APIs.
type ProjectsService service

// Get fetches a project.
//
// GET /api/projects/:project
func (s *ProjectsService) Get(ctx context.Context, project string) (*Project, error) {
	var out Project
	if _, err := s.client.getJSON(ctx, "projects/"+escapeSegment(project), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List lists the projects the authenticated user belongs to.
//
// GET /api/projects
func (s *ProjectsService) List(ctx context.Context) (*ProjectList, error) {
	var out ProjectList
	if _, err := s.client.getJSON(ctx, "projects", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
