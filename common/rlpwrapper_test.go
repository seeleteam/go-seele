/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/magiconair/properties/assert"
)

type Student struct {
	Name  string
	NO    uint
	score uint
	_age  uint
}

var s = &Student{
	Name:  "s1",
	NO:    123,
	score: 100,
	_age:  24,
}

// test rlp correction
func Test_RLP(t *testing.T) {
	buffer := bytes.Buffer{}
	err := rlp.Encode(&buffer, &s)
	if err != nil {
		panic(err)
	}

	nst := Student{}
	err = rlp.Decode(&buffer, &nst)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, nst.Name, s.Name)
	assert.Equal(t, nst.NO, s.NO)
	assert.Equal(t, nst.score, uint(0))
	assert.Equal(t, nst._age, uint(0))
}

// test rlp wrapper
func Test_RLPWrapper(t *testing.T) {
	data, err := Serialize(&s)
	if err != nil {
		panic(err)
	}

	nst := Student{}
	err = Deserialize(data, &nst)

	assert.Equal(t, nst.Name, s.Name)
	assert.Equal(t, nst.NO, s.NO)
	assert.Equal(t, nst.score, uint(0))
	assert.Equal(t, nst._age, uint(0))
}

// Panics when serialize unsupported data.
func Test_SerializePanic(t *testing.T) {
	type student struct {
		Score int64
	}

	_, expectedErr := Serialize(&student{1})

	defer func() {
		assert.Equal(t, recover(), expectedErr)
	}()

	SerializePanic(&student{1})
}
