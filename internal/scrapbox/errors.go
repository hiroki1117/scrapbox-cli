package scrapbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Sentinel errors for errors.Is.
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
)

// APIError is returned when the API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Method     string
	URL        string
	// Name and Message come from the Scrapbox error response {"name": "...", "message": "..."}.
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

// Is maps the status code to the sentinel errors.
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

// maxErrorBody is the maximum size of an error response body to read.
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
