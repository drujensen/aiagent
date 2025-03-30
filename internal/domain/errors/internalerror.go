package errors

import "fmt"

type InternalError struct {
	message string
}

func (v *InternalError) Error() string {
	return v.message
}

func InternalErrorf(format string, args ...interface{}) *InternalError {
	return &InternalError{
		message: fmt.Sprintf(format, args...),
	}
}

var _ error = &InternalError{}
