package scrapbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// errors.Is で判定するための番兵エラーである。
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
)

// APIError は API が 2xx 以外を返したときのエラーである。
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	// Name と Message は Scrapbox のエラーレスポンス {"name": "...", "message": "..."} の値である。
	Name    string
	Message string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	if e.Name != "" {
		msg = e.Name + ": " + msg
	}
	return fmt.Sprintf("%s %s: %d %s", e.Method, e.URL, e.StatusCode, msg)
}

// Is はステータスコードを番兵エラーに対応付ける。
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case ErrForbidden:
		return e.StatusCode == http.StatusForbidden
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	}
	return false
}

// maxErrorBody はエラーレスポンスとして読み込むボディの上限である。
const maxErrorBody = 64 << 10

func newAPIError(req *http.Request, resp *http.Response) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Method:     req.Method,
		URL:        req.URL.Redacted(),
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if err != nil {
		return e
	}
	var payload struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil {
		e.Name, e.Message = payload.Name, payload.Message
	}
	return e
}
