/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package errors

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StackedError_Error(t *testing.T) {
	e := errors.New("111")
	e = NewStackedError(e, "222")

	p := 4
	e = NewStackedErrorf(e, "333-%v", p)

	assert.Equal(t, strings.Join([]string{"333-4", "222", "111"}, errSeparator), e.Error())
}

func Test_StackedError_IsOrContains(t *testing.T) {
	err := errors.New("111")

	// same err instance
	assert.True(t, IsOrContains(err, err))

	// different err instances
	assert.False(t, IsOrContains(err, errors.New("111"))) // not equal even error message is the same
	assert.False(t, IsOrContains(err, errors.New("222")))

	// StackedError cases
	se := NewStackedError(err, "abc")
	assert.True(t, IsOrContains(se, err))

	se = NewStackedError(se, "edf")
	assert.True(t, IsOrContains(se, err))
}
