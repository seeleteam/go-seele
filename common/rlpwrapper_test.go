package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/check.v1"
	"testing"
)

type Student struct {
	Name  string
	NO    uint
	score uint
	_age  uint
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&Student{
	Name:  "s1",
	NO:    123,
	score: 100,
	_age:  24,
})

// test rlp correction
func (s *Student) Test_RLP(c *check.C) {
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

	c.Assert(s.Name, check.Equals, nst.Name)
	c.Assert(s.NO, check.Equals, nst.NO)
}

// test gob effective
func (s *Student) Test_Encoding(c *check.C) {
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
func (s *Student) Test_Json(c *check.C) {
	buff, err := json.Marshal(&s)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(len(buff))

	fmt.Println(string(buff))
}
