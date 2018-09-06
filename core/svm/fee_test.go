/**
* @file
* @copyright defined in go-seele/LICENSE
 */
package svm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_contractCreationFee(t *testing.T) {
	code0byte := []byte(nil)
	// [0, 4KB]
	fee0 := contractCreationFee(code0byte)
	assert.Equal(t, contractFeeSimple, fee0)

	code8b := []byte{2, 0, 1, 8, 0, 9, 0, 5}
	fee8b := contractCreationFee(code8b)
	assert.Equal(t, contractFeeSimple, fee8b)

	code4kb := code8b
	for len(code4kb) != 4*1024*1024 {
		code4kb = append(code4kb, code4kb...)
	}
	fee4kb := contractCreationFee(code4kb)
	assert.Equal(t, contractFeeSimple, fee4kb)

	// (4KB, 8KB]
	code4kb8b := append(code4kb, code8b...)
	fee4kb8b := contractCreationFee(code4kb8b)
	assert.Equal(t, contractFeeStandardToken, fee4kb8b)

	code8kb := append(code4kb, code4kb...)
	fee8kb := contractCreationFee(code8kb)
	assert.Equal(t, contractFeeStandardToken, fee8kb)

	// (8KB, 16KB]
	code8kb8b := append(code8kb, code8b...)
	fee8kb8b := contractCreationFee(code8kb8b)
	assert.Equal(t, contractFeeCustomToken, fee8kb8b)

	code16kb := append(code8kb, code8kb...)
	fee16kb := contractCreationFee(code16kb)
	assert.Equal(t, contractFeeCustomToken, fee16kb)

	// (16KB, ∞)
	code16kb8b := append(code16kb, code8b...)
	fee16kb8b := contractCreationFee(code16kb8b)
	assert.Equal(t, contractFeeComplex, fee16kb8b)
}

func Test_usedGasFee(t *testing.T) {
	// 0 gas
	used0gas := uint64(0)
	fee0gas := usedGasFee(used0gas)
	assert.Equal(t, gasFeeZero, fee0gas)

	// (0, 50000gas]
	used8gas := uint64(8)
	fee8gas := usedGasFee(used8gas)
	assert.Equal(t, gasFeeLowPrice, fee8gas)

	used5wgas := lowPriceGas
	fee5wgas := usedGasFee(used5wgas)
	assert.Equal(t, gasFeeLowPrice, fee5wgas)

	// (50000gas, ∞)
	used5w8gas := uint64(58000)
	fee5w8gas := usedGasFee(used5w8gas)
	overUsed := (used5w8gas-lowPriceGas)/overUsedStep + 1
	assert.Equal(t, new(big.Int).SetUint64(gasFeeHighPriceUnit.Uint64()*overUsed*overUsed), fee5w8gas)
}
