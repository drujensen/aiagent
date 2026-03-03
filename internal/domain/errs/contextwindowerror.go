package errors

import "fmt"

type ContextWindowError struct {
	message string
}

func (v *ContextWindowError) Error() string {
	return v.message
}

func ContextWindowErrorf(format string, args ...any) *ContextWindowError {
	return &ContextWindowError{
		message: fmt.Sprintf(format, args...),
	}
}

var _ error = &ContextWindowError{}
