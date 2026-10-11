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
}

func (e *Error) Error() string                   { return e.Code + ": " + e.Message }
func New(kind Kind, code, message string) *Error { return &Error{kind, code, message} }
