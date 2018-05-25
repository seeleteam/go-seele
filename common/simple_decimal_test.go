/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"github.com/magiconair/properties/assert"
	"math/big"
	"testing"
)

func Test_BigToDecimal(t *testing.T) {
	var re string
	var equal = false
	var number = big.NewInt(100000000)
	re = BigToDecimal(number)
	equal = ("1" == re)
	assert.Equal(t, equal, true)

	number = big.NewInt(0)
	re = BigToDecimal(number)
	equal = ("0" == re)
	assert.Equal(t, equal, true)

	number = big.NewInt(12300)
	re = BigToDecimal(number)
	equal = ("0.000123" == re)
	assert.Equal(t, equal, true)

	number = big.NewInt(123)
	re = BigToDecimal(number)
	equal = ("0.00000123" == re)
	assert.Equal(t, equal, true)

	number = big.NewInt(100012345678)
	re = BigToDecimal(number)

	equal = ("1000.12345678" == re)
	assert.Equal(t, equal, true)

	number = big.NewInt(10012345600)
	re = BigToDecimal(number)
	equal = ("100.123456" == re)
	assert.Equal(t, equal, true)

	number = big.NewInt(800000600)
	re = BigToDecimal(number)
	equal = ("8.000006" == re)
	assert.Equal(t, equal, true)
}
