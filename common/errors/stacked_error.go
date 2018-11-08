/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package errors

import (
	"bytes"
	"fmt"
)

const errSeparator = " ===> "

// StackedError records errors that occurred in call stack.
type StackedError struct {
	msg   string
	inner error
}

// NewStackedError returns a StackedError with specified inner error and error msg.
func NewStackedError(inner error, msg string) error {
	return &StackedError{
		msg:   msg,
		inner: inner,
	}
}

// NewStackedErrorf returns a StackedError with specified inner error and an error format specifier.
func NewStackedErrorf(inner error, format string, a ...interface{}) error {
	return &StackedError{
		msg:   fmt.Sprintf(format, a...),
		inner: inner,
	}
}

// Error implements the error interface.
func (err *StackedError) Error() string {
	var buf bytes.Buffer

	buf.WriteString(err.msg)

	for innerErr := err.inner; innerErr != nil; {
		buf.WriteString(errSeparator)

		if se, ok := innerErr.(*StackedError); ok {
			buf.WriteString(se.msg)
			innerErr = se.inner
		} else {
			buf.WriteString(innerErr.Error())
			innerErr = nil
		}
	}

	return buf.String()
}

// IsOrContains indicates whether the err is the specified inner error,
// or the err is StackedError and contains the specified inner error.
func IsOrContains(err error, inner error) bool {
	for err != nil {
		if err == inner {
			return true
		}

		if se, ok := err.(*StackedError); ok {
			err = se.inner
		} else {
			break
		}
	}

	return false
}
