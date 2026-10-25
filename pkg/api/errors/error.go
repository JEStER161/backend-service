package errors

import "fmt"

type Error struct {
	Status   int
	Code     string
	Message  string
	Internal error
}

// New создаёт прикладную ошибку с публичным ответом и внутренней причиной
func New(status int, code, message string, internal error) *Error {
	return &Error{Status: status, Code: code, Message: message, Internal: internal}
}

// Error возвращает безопасное текстовое представление ошибки
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap возвращает внутреннюю причину для errors.Is и errors.As
func (e *Error) Unwrap() error {
	return e.Internal
}
