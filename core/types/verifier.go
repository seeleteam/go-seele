package types

import (
	"github.com/seeleteam/go-seele/common"
)

// IsVerifierTx return whether the tx Is operate tx or not?
func (tx *Transaction) IsVerifierTx() bool {
	// fmt.Printf("_____transcation types_____\n\n")
	// fmt.Printf("From: %s \n", tx.Data.From)
	// fmt.Printf("Type: %d \n", tx.Data.Type)
	// fmt.Printf("Roots: %+v\n\n", common.RootAccounts)
	if tx.Data.Type == 1 && tx.Data.From == common.RootAccounts[0] {
		return true
	}

	return false
}

// IsResignTx return
func (tx *Transaction) IsResignTx() bool {
	// fmt.Printf("_____transcation types_____\n\n")
	// fmt.Printf("From: %s \n", tx.Data.From)
	// fmt.Printf("Type: %d \n", tx.Data.Type)
	// fmt.Printf("Roots: %+v\n\n", common.RootAccounts)
	if tx.Data.Type == 1 && tx.Data.From == common.RootAccounts[1] {
		return true
	}

	return false
}

// IsDepositTx return
func (tx *Transaction) IsDepositTx() bool {
	// fmt.Printf("_____transcation types_____\n\n")
	// fmt.Printf("From: %s \n", tx.Data.From)
	// fmt.Printf("Type: %d \n", tx.Data.Type)
	// fmt.Printf("Roots: %+v\n\n", common.RootAccounts)
	if tx.Data.Type == 0 && tx.Data.From == common.RootAccounts[0] {
		return true
	}

	return false
}

// IsExitTx return
func (tx *Transaction) IsExitTx() bool {
	// fmt.Printf("_____transcation types_____\n\n")
	// fmt.Printf("From: %s \n", tx.Data.From)
	// fmt.Printf("Type: %d \n", tx.Data.Type)
	// fmt.Printf("Roots: %+v\n\n", common.RootAccounts)
	if tx.Data.Type == 0 && tx.Data.From == common.RootAccounts[1] {
		return true
	}

	return false

}

// IsChallengedTx return
func (tx *Transaction) IsChallengedTx() bool {
	// fmt.Printf("_____transcation types_____\n\n")
	// fmt.Printf("From: %s \n", tx.Data.From)
	// fmt.Printf("Type: %d \n", tx.Data.Type)
	// fmt.Printf("Roots: %+v\n\n", common.RootAccounts)
	if tx.Data.Type == 0 && tx.Data.From == common.RootAccounts[2] {
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
