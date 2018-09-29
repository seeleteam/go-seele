/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	"testing"

	"github.com/seeleteam/go-seele/common"
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

type testOdrObj struct {
	OdrItem
	Name string
}

func Test_OdrItem_Serialize(t *testing.T) {
	obj := testOdrObj{
		OdrItem: OdrItem{
			ReqID: 38,
			Error: "hello, error",
		},
		Name: "test name",
	}

	assertSerializable(t, &obj, &testOdrObj{})
}

func assertSerializable(t *testing.T, ptrToEncode interface{}, ptrForDecode interface{}) {
	encoded, err := common.Serialize(ptrToEncode)
	assert.Nil(t, err)

	assert.Nil(t, common.Deserialize(encoded, ptrForDecode))
	assert.Equal(t, ptrToEncode, ptrForDecode)
}
