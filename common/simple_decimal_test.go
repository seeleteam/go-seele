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
	var number = big.NewInt(100000000)
	assert.Equal(t, BigToDecimal(number), "1")

	number = big.NewInt(0)
	assert.Equal(t, BigToDecimal(number), "0")

	number = big.NewInt(12300)
	assert.Equal(t, BigToDecimal(number), "0.000123")

	number = big.NewInt(123)
	assert.Equal(t, BigToDecimal(number), "0.00000123")

	number = big.NewInt(100012345678)
	assert.Equal(t, BigToDecimal(number), "1000.12345678")

	number = big.NewInt(10012345600)
	assert.Equal(t, BigToDecimal(number), "100.123456")

	number = big.NewInt(800000600)
	assert.Equal(t, BigToDecimal(number), "8.000006")
}
