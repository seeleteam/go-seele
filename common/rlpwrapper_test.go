/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
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

var sWithPartialValues = &Student{
	Name: "s1",
}

var sEmpty = &Student{}

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
	// Full object
	data := serializeStudent(s)
	nst := Student{}
	err := Deserialize(data, &nst)

	assert.Equal(t, err, nil)
	assert.Equal(t, nst.Name, s.Name)
	assert.Equal(t, nst.NO, s.NO)
	assert.Equal(t, nst.score, uint(0))
	assert.Equal(t, nst._age, uint(0))

	// Partial object
	data = serializeStudent(sWithPartialValues)
	nst = Student{}
	err = Deserialize(data, &nst)

	assert.Equal(t, err, nil)
	assert.Equal(t, nst.Name, s.Name)
	assert.Equal(t, nst.NO, uint(0))
	assert.Equal(t, nst.score, uint(0))
	assert.Equal(t, nst._age, uint(0))

	// Empty object
	data = serializeStudent(sEmpty)
	nst = Student{}
	err = Deserialize(data, &nst)

	assert.Equal(t, err, nil)
	assert.Equal(t, nst.Name, "")
	assert.Equal(t, nst.NO, uint(0))
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

func serializeStudent(s *Student) []byte {
	data, err := Serialize(s)
	if err != nil {
		panic(err)
	}

	return data
}

type A struct {
	A1 uint
}

type B struct {
	Num uint
	B1  *Address
}

func Test_StructNil(t *testing.T) {
	b := &B{
		Num: 1,
		B1:  nil,
	}

	buff, err := Serialize(b)
	assert.Nil(t, err)

	result := &B{}
	err = Deserialize(buff, result)
	assert.NotNil(t, err)
}

type small struct {
	Str string
}

type C struct {
	Num uint
	small
}

func Test_UnexportedStruct(t *testing.T) {
	c := &C{
		Num: 1,
	}

	c.Str = "a"

	buff, err := Serialize(c)
	assert.Nil(t, err)

	result := &C{}
	err = Deserialize(buff, result)
	assert.Nil(t, err)
	assert.NotEqual(t, result.Str, "a")
}
