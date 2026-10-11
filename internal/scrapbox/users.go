package scrapbox

import "context"

// UsersService handles user APIs.
type UsersService service

// Me fetches the authenticated user. Without authentication, it returns a User with IsGuest set.
//
// GET /api/users/me
func (s *UsersService) Me(ctx context.Context) (*User, error) {
	var out User
	if _, err := s.client.getJSON(ctx, "users/me", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
