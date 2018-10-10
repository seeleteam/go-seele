/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package svm

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

///////////////////////////////////////////////////////////////////////////////////////
// Gas fee model for test net
///////////////////////////////////////////////////////////////////////////////////////
var (
	contractFeeComplex       = new(big.Int).Div(common.SeeleToFan, big.NewInt(100))
	contractFeeCustomToken   = new(big.Int).Div(common.SeeleToFan, big.NewInt(200))
	contractFeeStandardToken = new(big.Int).Div(common.SeeleToFan, big.NewInt(500))
	contractFeeSimple        = new(big.Int).Div(common.SeeleToFan, big.NewInt(1000))

	lowPriceGas  = uint64(50000) // 2 storage op allowed
	overUsedStep = uint64(20000) // about 1 storage op

	gasFeeZero          = new(big.Int)
	gasFeeLowPrice      = new(big.Int).Div(contractFeeSimple, big.NewInt(1000))
	gasFeeHighPriceUnit = new(big.Int).Div(contractFeeSimple, big.NewInt(100))
)

// contractCreationFee returns the contract creation fee according to code size.
func contractCreationFee(code []byte) *big.Int {
	codeLen := len(code)

	// complex contract > 16KB
	if codeLen > 16*1024*1024 {
		return contractFeeComplex
	}

	// custom simple ERC20 token between (8KB, 16KB]
	if codeLen > 8*1024*1024 {
		return contractFeeCustomToken
	}

	// standard ERC20 token between (4KB, 8KB]
	if codeLen > 4*1024*1024 {
		return contractFeeStandardToken
	}

	// other simple contract
	return contractFeeSimple
}

// usedGasFee returns the contract execution fee according to used gas.
//   - if usedGas == 0, returns 0.
//   - if usedGas <= 50000 (2 store op allowed), returns 1/1000 * contractFeeSimple
//   - else returns 1/100 * contractFeeSimple * overUsed^2
func usedGasFee(usedGas uint64) *big.Int {
	if usedGas == 0 {
		return gasFeeZero
	}

	if usedGas <= lowPriceGas {
		return gasFeeLowPrice
	}

	overUsed := (usedGas-lowPriceGas)/overUsedStep + 1

	return new(big.Int).Mul(gasFeeHighPriceUnit, new(big.Int).SetUint64(overUsed*overUsed))
}
