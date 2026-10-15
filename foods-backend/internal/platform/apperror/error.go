// Package apperror defines transport-independent failures returned by use cases.
package apperror

type Kind uint8

const (
	Invalid Kind = iota
	Unauthenticated
	Forbidden
	NotFound
	Conflict
	Unavailable
	Internal
)

type Error struct {
	Kind          Kind
	Code, Message string
	// Cause keeps the technical failure for server logs; it is never serialized.
	Cause error
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func (e *Error) Unwrap() error { return e.Cause }
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap returns a safe failure that preserves its technical cause for diagnostics.
func Wrap(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}
