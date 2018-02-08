/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package common

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/log"
)

func Test_Hex(t *testing.T) {
	str := "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"

	bytes, err := HexToBytes(str)
	if err != nil {
		log.Fatal(err.Error())
	}

	res := BytesToHex(bytes)

	assert.Equal(t, res, str)

	bytes, err = HexToBytes("")
	assert.Equal(t, err, ErrEmptyString)
}
