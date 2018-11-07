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
