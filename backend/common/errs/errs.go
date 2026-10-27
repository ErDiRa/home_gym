// Package errs defines the error the domain services in its subpackages return for
// problems the caller caused. It is shared so the HTTP layer can map the errors of every
// service to a status code in one place.
package errs

// Kind classifies an Error; the HTTP layer maps it to a status code.
type Kind int

const (
	KindInvalid      Kind = iota + 1 // the input is invalid or references something unknown
	KindNotFound                     // the thing acted on doesn't exist
	KindConflict                     // the input clashes with existing data, e.g. a taken email
	KindUnauthorized                 // the credentials are wrong
)

// Error is a failure the caller can do something about. Msg is safe to show to the client;
// Err is the underlying cause, if any, and is only meant for the log.
type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

// Invalid reports invalid input with a message for the client.
func Invalid(msg string) error { return &Error{Kind: KindInvalid, Msg: msg} }

// InvalidCause is Invalid with the cause kept for the log.
func InvalidCause(msg string, err error) error { return &Error{Kind: KindInvalid, Msg: msg, Err: err} }

// NotFound reports a missing thing; err is the cause, e.g. pgx.ErrNoRows, or nil.
func NotFound(msg string, err error) error { return &Error{Kind: KindNotFound, Msg: msg, Err: err} }

// Conflict reports input that clashes with existing data; err is the cause or nil.
func Conflict(msg string, err error) error { return &Error{Kind: KindConflict, Msg: msg, Err: err} }

// Unauthorized reports wrong credentials.
func Unauthorized(msg string) error { return &Error{Kind: KindUnauthorized, Msg: msg} }
