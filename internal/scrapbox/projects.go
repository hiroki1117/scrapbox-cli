package scrapbox

import (
	"context"
	"errors"
)

// ProjectsService handles project APIs.
type ProjectsService service

// Get fetches a project.
// The endpoint rejects Personal Access Tokens for private projects, so on 401 it looks
// the project up in List, which accepts them.
//
// GET /api/projects/:project
func (s *ProjectsService) Get(ctx context.Context, project string) (*Project, error) {
	var out Project
	_, err := s.client.getJSON(ctx, "projects/"+escapeSegment(project), nil, &out)
	if errors.Is(err, ErrUnauthorized) {
		if list, lerr := s.List(ctx); lerr == nil {
			for _, p := range list.Projects {
				if p.Name == project {
					return &p, nil
				}
			}
		}
	}
	if err != nil {
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
