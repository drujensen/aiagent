package errors

import "fmt"

type DuplicateError struct {
	message string
}

func (v *DuplicateError) Error() string {
	return v.message
}

func DuplicateErrorf(format string, args ...any) *DuplicateError {
	return &DuplicateError{
		message: fmt.Sprintf(format, args...),
	}
}

var _ error = &DuplicateError{}
