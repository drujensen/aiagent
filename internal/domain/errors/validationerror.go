package errors

import "fmt"

type ValidationError struct {
	message string
}

func (v *ValidationError) Error() string {
	return v.message
}

func ValidationErrorf(format string, args ...any) *ValidationError {
	return &ValidationError{
		message: fmt.Sprintf(format, args...),
	}
}

var _ error = &ValidationError{}
