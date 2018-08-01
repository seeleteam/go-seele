package util

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_Util_NewHashLock(t *testing.T) {
	var data interface{}
	data = 1
	v, err := NewHashLock(data)

	assert.Equal(t, err, nil)
	assert.Equal(t, v.Data, data)
	assert.Equal(t, len(v.Hash), common.HashLength)

	data = struct {
		one int
		two []byte
	}{one: 1, two: []byte{1, 2}}
	v, err = NewHashLock(data)

	assert.Equal(t, err, nil)
	assert.Equal(t, v.Data, data)
	assert.Equal(t, len(v.Hash), common.HashLength)

}

func Test_Util_Claim(t *testing.T) {
	var data interface{}
	data = struct {
		one int
		two []byte
	}{one: 1, two: []byte{1, 2}}
	v, err := NewHashLock(data)

	assert.Equal(t, err, nil)
	assert.Equal(t, v.Data, data)
	assert.Equal(t, len(v.Hash), common.HashLength)

	assert.Equal(t, true, v.Claim(data))
}
