package types

import (
	"github.com/seeleteam/go-seele/common"
)

// IsVerifierTx return whether the tx Is operate tx or not?
func (tx *Transaction) IsVerifierTx() bool {
	/*
		define Tx to verfier check condition here
		// verifier tx
	*/
	return true
}

func (tx *Transaction) IsDepositTx() bool {
	/*
		define Tx to verfier check condition here
	*/
	return false
}
func (tx *Transaction) IsChallengedTx() bool {
	/*
		define Tx to verfier check condition here
	*/
	return false
}

func (tx *Transaction) IsExitTx() bool {
	/*
		define Tx to verfier check condition here
	*/
	return false
}

// VerifiersFromTxBytes convert verifiers from common.Address type to bytes, so we can store into the header.SecondWitness.
func (tx *Transaction) VerifiersFromTxBytes(addrs []common.Address) []byte {
	var versBytes []byte
	for _, addr := range addrs {
		versBytes = append(versBytes, addr.Bytes()...)
	}
	return versBytes
}
