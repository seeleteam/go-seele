package contract

import (
	"github.com/seeleteam/go-seele/crypto"
)

// HandleTransaction handles smart contract transation.
func HandleTransaction(tx *Transaction) {
	hash := crypto.Keccak256Hash(tx.payload)
	if !tx.sig.Verify(hash) {
		log.Error("Transaction signature is invalid, and will not be archived into blockchain.")
		return
	}

	switch tx.txType {
	case txTypeContractReg:
		onRegisterContract(tx)
	case txTypeContractCall:
		onInvokeContract(tx)
	case txTypeContractDel:
		onDeleteContrct(tx)
	}
}

func onRegisterContract(tx *Transaction) {
	codeAddr := codeAddress(tx.payload)
	ChainServ.WriteCode(codeAddr, tx.payload)

	// TODO run smart contract for init?
	account := &Account{
		AccountAddress: tx.from,
		CodeAddress:    codeAddr,
		State:          make([]byte, 0),
	}
	ChainServ.WriteContractAccount(account)

	ChainServ.WriteTransaction(tx)
}

func onInvokeContract(tx *Transaction) {
	account := ChainServ.GetContractAccount(tx.to)
	code := ChainServ.GetCode(account.CodeAddress)
	vm := exeVM{}
	vm.Execute(code, tx.payload)
}

func onDeleteContrct(tx *Transaction) {
	// TODO logic vs. physical deletion.
}
