/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package errors

import (
	"fmt"
)

// seeleError represents a seele error with code and message.
type seeleError struct {
	code ErrorCode
	msg  string
}

// seeleParameterizedError represents a seele error with code and parameterized message.
// For type safe of common used business error, developer could define a concrete error to process.
type seeleParameterizedError struct {
	seeleError
	parameters []interface{}
}

func newSeeleError(code ErrorCode, msg string) error {
	return &seeleError{code, msg}
}

// Error implements the error interface.
func (err *seeleError) Error() string {
	return err.msg
}

// Get returns a seele error with specified error code.
func Get(code ErrorCode) error {
	err, found := constErrors[code]
	if !found {
		return fmt.Errorf("system internal error, cannot find the error code %v", code)
	}

	return err
}

// Create creates a seele error with specified error code and parameters.
func Create(code ErrorCode, args ...interface{}) error {
	errFormat, found := parameterizedErrors[code]
	if !found {
		return fmt.Errorf("system internal error, cannot find the error code %v", code)
	}

	return &seeleParameterizedError{
		seeleError: seeleError{code, fmt.Sprintf(errFormat, args...)},
		parameters: args,
	}
}
