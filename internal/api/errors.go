package api

import "net/http"

var (
	ErrUnauthorized = newError("Unauthorized")
	ErrBadRequest   = newError("Body invalid")
)

// HTTPError is custom HTTP error for API
type HTTPError struct {
	Code int64  `json:"Code"`
	Msg  string `json:"Msg"`
}

func (e *HTTPError) Error() string {
	return e.Msg
}

func newError(msg string) *HTTPError {
	return &HTTPError{Code: http.StatusBadGateway, Msg: msg}
}
