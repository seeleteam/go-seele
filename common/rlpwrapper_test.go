/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		fmt.Println(err)
	}

	data := buffer.Bytes()
	fmt.Println(len(data))

	nst := Student{}
	err = rlp.Decode(&buffer, &nst)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%v\n", nst)

	assert.Equal(t, nst.Name, s.Name)
	assert.Equal(t, nst.NO, s.NO)
}

// test gob effective
func Test_Encoding(t *testing.T) {
	data, err := Encoding(&s)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(len(data))

	nst := Student{}
	err = Decoding(data, &nst)

	fmt.Printf("%v\n", nst)
}

// test json effective
func Test_Json(t *testing.T) {
	buff, err := json.Marshal(&s)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(len(buff))

	fmt.Println(string(buff))
}
