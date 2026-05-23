package config

import "fmt"

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(format string, args ...any) error {
	return &Error{Message: fmt.Sprintf(format, args...)}
}
