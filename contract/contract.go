/**
* @file
* @copyright defined in go-seele/LICENSE
 */
package contract

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"

	"github.com/seeleteam/go-seele/crypto"
)

var (
	// TxServ provide the transaction related service.
	// TODO initialize with the default implementation.
	TxServ TransactionService

	// ChainServ provide the blockchain related service.
	// TODO initialize with the default implementation.
	ChainServ BlockchainService
)

// codeAddress returns the code address based on hash.
func codeAddress(code []byte) string {
	// TODO could use any other encoding method to shorten the address, e.g. base58
	return hex.EncodeToString(crypto.HashBytes(code).Bytes())
}

// Address returns account public address for the specified private key.
func Address(privKey *ecdsa.PrivateKey) string {
	data := elliptic.Marshal(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)
	// TODO could use any other encoding method to shorten the address, e.g. base58
	return hex.EncodeToString(data)
}
