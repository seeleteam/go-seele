package types

import (
	"github.com/seeleteam/go-seele/common"
)

// IsVerifierTx return whether the tx Is operate tx or not?
func (tx *Transaction) IsVerifierTx(rootAccounts []common.Address) bool {
	if tx.Data.Type == 1 && tx.Data.From == rootAccounts[0] {
		return true
	}

	return false
}

// IsResignTx return
func (tx *Transaction) IsResignTx(rootAccounts []common.Address) bool {
	if tx.Data.Type == 1 && tx.Data.From == rootAccounts[1] {
		return true
	}

	return false
}

// IsDepositTx return
func (tx *Transaction) IsDepositTx(rootAccounts []common.Address) bool {

	if tx.Data.Type == 0 && tx.Data.From == rootAccounts[0] {
		return true
	}

	return false
}

// IsExitTx return
func (tx *Transaction) IsExitTx(rootAccounts []common.Address) bool {

	if tx.Data.Type == 0 && tx.Data.From == rootAccounts[1] {
		return true
	}

	return false

}

// IsChallengedTx return
func (tx *Transaction) IsChallengedTx(rootAccounts []common.Address) bool {

	if tx.Data.Type == 0 && tx.Data.From == rootAccounts[2] {
		return true
	}

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
