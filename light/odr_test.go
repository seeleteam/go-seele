/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Odr_AllFuncs(t *testing.T) {
	id := uint32(1)
	err := ""

	odr := newOdrItem(id, err)
	assert.Equal(t, odr.getRequestID(), id)
	assert.Nil(t, odr.getError())

	id = uint32(2)
	odr.setRequestID(id)
	assert.Equal(t, odr.getRequestID(), id)
	assert.Nil(t, odr.getError())

	errString := "something unexpected"
	odr = newOdrItem(id, errString)
	assert.Equal(t, odr.getError(), errors.New(errString))
}

func newOdrItem(id uint32, err string) *OdrItem {
	return &OdrItem{
		ReqID: id,
		Error: err,
	}
}
