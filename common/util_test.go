/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_Bytes(t *testing.T) {
	str := "123456789"
	arrayByte := Bytes(str)

	arrayByte1, err := json.Marshal(arrayByte)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.HasPrefix(string(arrayByte1), "\"0x"), true)

	tx := struct {
		ID      int
		PayLoad Bytes
	}{
		ID:      1,
		PayLoad: arrayByte,
	}

	arrayByte2, err := json.Marshal(tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.Contains(string(arrayByte2), "0x"), true)

	tx1 := struct {
		ID      int
		PayLoad Bytes
	}{}

	err = json.Unmarshal(arrayByte2, &tx1)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx.ID, tx1.ID)
	assert.Equal(t, tx.PayLoad, tx1.PayLoad)
	assert.Equal(t, string(tx1.PayLoad), str)

	tx.PayLoad = nil
	arrayByte3, err := json.Marshal(tx)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.Contains(string(arrayByte3), "0x"), false)

	tx2 := struct {
		ID      int
		PayLoad Bytes
	}{}

	err = json.Unmarshal(arrayByte3, &tx2)
	assert.Equal(t, err, nil)
	assert.Equal(t, tx2.ID, tx.ID)
	assert.Equal(t, tx2.PayLoad == nil, true)
}

func Test_CopyBytes(t *testing.T) {
	// src is valid with length > 0
	src := make([]byte, 1, 1)
	dest := CopyBytes(src)
	assert.Equal(t, bytes.Compare(src, dest), 0)

	src = make([]byte, 10, 10)
	src = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	dest = CopyBytes(src)
	assert.Equal(t, bytes.Compare(src, dest), 0)

	// src is nil
	src = nil
	dest = CopyBytes(src)
	assert.Equal(t, dest, []byte(nil))
}

func Test_MustNewCache(t *testing.T) {
	cache := MustNewCache(3)
	if cache == nil {
		t.Fatal()
	}

	assert.Panic(t, func() { MustNewCache(0) }, "Must provide a positive size")
	assert.Panic(t, func() { MustNewCache(-1) }, "Must provide a positive size")
}
