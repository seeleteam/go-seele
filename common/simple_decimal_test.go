/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
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

func Test_MaxMinIntToDecimal(t *testing.T) {
	var num int64 = math.MaxInt64 // 9223372036854775807
	var number = big.NewInt(num)
	assert.Equal(t, BigToDecimal(number), "92233720368.54775807")

	num = math.MinInt64 // -9223372036854775808
	number = big.NewInt(num)
	assert.Equal(t, BigToDecimal(number), "-92233720369.45224192")
}
