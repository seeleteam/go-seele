package contract

import (
	"crypto/ecdsa"

	"github.com/seeleteam/go-seele/crypto"
)

// Operator wraps request operations of smart contract.
// All the operations should be signed.
type Operator struct {
	privKey *ecdsa.PrivateKey
}

// NewOperator return a operator for smart contract operations.
func NewOperator(privKey *ecdsa.PrivateKey) *Operator {
	return &Operator{privKey}
}

// Register a smart contract with specified code.
func (operator *Operator) Register(code []byte) {
	operator.sendTx("", txTypeContractReg, code)
}

func (operator *Operator) sendTx(toAddr string, txType int, payload []byte) {
	hash := crypto.Keccak256Hash(payload)

	sig, err := crypto.NewSignature(operator.privKey, hash)
	if err != nil {
		log.Error(err.Error())
	}

	tx := &Transaction{
		from:    Address(operator.privKey),
		to:      toAddr,
		txType:  txType,
		payload: payload,
		sig:     sig,
	}

	TxServ.SendTransaction(tx)
}

// Invoke smart contract with specified parameters.
func (operator *Operator) Invoke(contractAddress string, msg []byte) {
	operator.sendTx(contractAddress, txTypeContractCall, msg)
}

// Destroy the specified smart contract
func (operator *Operator) Destroy(contractAddress string) {
	operator.sendTx(contractAddress, txTypeContractDel, []byte(""))
}
