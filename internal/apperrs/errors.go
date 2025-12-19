package apperrs

import "errors"

type Kind string

const (
	KindClient Kind = "client"
	KindServer Kind = "server"
)

const (
	CodeNotFound      = "NotFound"
	CodeInvalidInput  = "InvalidInput"
	CodeUnauthorized  = "Unauthorized"
	CodeInternalError = "InternalError"
	CodeConflict      = "Conflict"
	CodeForbidden     = "Forbidden"

	CodeInvalidSubdomain = "InvalidSubdomain"
)

type Error struct {
	Kind Kind
	Code string
	Msg  string
	Meta map[string]any
	Err  error // wrapped error
}

func (e *Error) SetMeta(key string, value any) *Error {
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}
	e.Meta[key] = value
	return e
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

func (e *Error) Unwrap() error {
	return e.Err
}

func Client(code, msg string) *Error {
	return &Error{
		Kind: KindClient,
		Code: code,
		Msg:  msg,
		Meta: make(map[string]any),
	}
}

func Server(msg string, err error) *Error {
	return &Error{
		Kind: KindServer,
		Code: "InternalError",
		Msg:  msg,
		Err:  err,
	}
}

func CodeIs(err error, code string) bool {
	var appErr *Error
	if ok := errors.As(err, &appErr); ok {
		return appErr.Code == code
	}
	return false
}
