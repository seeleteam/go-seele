package vm

import (
	"bytes"
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_ecrecover(t *testing.T) {
	addr, privKey, err := crypto.GenerateKeyPair()
	assert.Nil(t, err)

	input := newEcrecoverInput(privKey)

	contract := &ecrecover{}
	recoveredAddr, err := contract.Run(input)
	assert.Nil(t, err)
	assert.Equal(t, common.LeftPadBytes(addr.Bytes(), 32), recoveredAddr)
}

func newEcrecoverInput(privKey *ecdsa.PrivateKey) []byte {
	hash := crypto.MustHash("hello, world!!!").Bytes()
	sig := crypto.MustSign(privKey, hash)

	r := sig.Sig[:32]
	s := sig.Sig[32:64]
	v := sig.Sig[64] + 27
	v32 := common.LeftPadBytes([]byte{v}, 32)

	return bytes.Join([][]byte{hash, v32, r, s}, []byte{})
}
