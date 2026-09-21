package apperr

import "fmt"

const (
	CodeValidation     = "VALIDATION_ERROR"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeAccessDenied   = "ACCESS_DENIED"
	CodeNotFound       = "NOT_FOUND"
	CodeTimeout        = "TIMEOUT"
	CodeUnavailable    = "UNAVAILABLE"
	CodeProviderError  = "PROVIDER_ERROR"
	CodeLogicError     = "LOGIC_ERROR"
	CodeServerError    = "SERVER_ERROR"
	CodeBadRequest     = "BAD_REQUEST"
	CodeInternal       = "INTERNAL_ERROR"
)

type Error struct {
	HTTPStatus int
	Code       string
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Cause)
	}
	return e.Code
}

func (e *Error) Unwrap() error { return e.Cause }

func New(status int, code, message string) *Error {
	return &Error{HTTPStatus: status, Code: code, Message: message}
}

func Wrap(status int, code, message string, cause error) *Error {
	return &Error{HTTPStatus: status, Code: code, Message: message, Cause: cause}
}

func As(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	e, ok := err.(*Error)
	return e, ok
}
