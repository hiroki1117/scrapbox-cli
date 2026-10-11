package scrapbox

import "context"

// UsersService はユーザーに関する API を扱う。
type UsersService service

// Me は認証ユーザーの情報を取得する。未認証の場合は IsGuest が true の User を返す。
//
// GET /api/users/me
func (s *UsersService) Me(ctx context.Context) (*User, error) {
	var out User
	if _, err := s.client.getJSON(ctx, "users/me", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
