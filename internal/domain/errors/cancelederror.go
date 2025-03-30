package errors

import "fmt"

type CanceledError struct {
	message string
}

func (v *CanceledError) Error() string {
	return v.message
}

func CanceledErrorf(format string, args ...interface{}) *CanceledError {
	return &CanceledError{
		message: fmt.Sprintf(format, args...),
	}
}

var _ error = &CanceledError{}
