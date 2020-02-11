/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package bft

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

const WitnessSize = 8

// GetSignatureAddress gets the signer address from the signature
func GetSignatureAddress(data []byte, sig []byte) (common.Address, error) {
	// 1. Keccak data
	hashData := crypto.Keccak256([]byte(data))
	// 2. Recover public key
	pubkey, err := crypto.SigToPub(hashData, sig)
	if err != nil {
		return common.Address{}, err
	}
	return *crypto.GetAddress(pubkey), nil
}

// CheckVerifierSignature check the validator in or not in the verset by signature
func CheckVerifierSignature(verSet VerifierSet, data []byte, sig []byte) (common.Address, error) {
	// 1. Get signature address
	signer, err := GetSignatureAddress(data, sig)
	if err != nil {
		return common.Address{}, err
	}

	// 2. Check validator
	if _, val := verSet.GetVerByAddress(signer); val != nil {
		return val.Address(), nil
	}

	return common.Address{}, ErrAddressUnauthorized
}
