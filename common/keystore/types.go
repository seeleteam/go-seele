package keystore

import (
	"crypto/ecdsa"

	"github.com/seeleteam/go-seele/common"
)

const (
	// Version keystore version
	Version = 1
)

// Key private key info for wallet
type Key struct {
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

type encryptedKey struct {
	Version int        `json:"version"`
	Address string     `json:"address"`
	Crypto  cryptoInfo `json:"crypto"`
}

type cryptoInfo struct {
	CipherText string `json:"ciphertext"`
	CipherIV   string `json:"iv"`
	Salt       string `json:"salt"`
	MAC        string `json:"mac"`
}
