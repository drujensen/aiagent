package errors

import "fmt"

type NotFoundError struct {
	message string
}

func (v *NotFoundError) Error() string {
	return v.message
}

func NotFoundErrorf(format string, args ...interface{}) *NotFoundError {
	return &NotFoundError{
		message: fmt.Sprintf(format, args...),
	}
}

var _ error = &NotFoundError{}
