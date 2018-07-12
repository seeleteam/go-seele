/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package crypto

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_GenerateKey(t *testing.T) {
	ecdsaKey, err := GenerateKey()

	assert.Equal(t, err, nil)
	assert.Equal(t, ecdsaKey != nil, true)
}

func Test_ToECDSAPub(t *testing.T) {
	// len == 0
	var pub = []byte{}
	ecdsaKey := ToECDSAPub(pub)
	assert.Equal(t, ecdsaKey == nil, true)

	// len == 65
	pub = []byte("01234567890012345678900123456789001234567890012345678900123456789")
	ecdsaKey = ToECDSAPub(pub)
	assert.Equal(t, ecdsaKey != nil, true)

	// len == 64
	pub = []byte("0123456789001234567890012345678900123456789001234567890012345678")
	ecdsaKey = ToECDSAPub(pub)
	assert.Equal(t, ecdsaKey != nil, true)

	// len == 1
	pub = []byte("0")
	ecdsaKey = ToECDSAPub(pub)
	assert.Equal(t, ecdsaKey != nil, true)
}

func Test_PubkeyToString(t *testing.T) {
	_, ecdsaPrivKey, err := GenerateKeyPair()

	assert.Equal(t, err, nil)
	assert.Equal(t, len(PubkeyToString(&ecdsaPrivKey.PublicKey)), 42)
}

func Test_FromECDSAPub(t *testing.T) {
	_, ecdsaPrivKey, err := GenerateKeyPair()

	assert.Equal(t, err, nil)
	assert.Equal(t, len(FromECDSAPub(&ecdsaPrivKey.PublicKey)), 65)
}

func Test_LoadECDSAFromString(t *testing.T) {
	ecdsaPrivKey, err := LoadECDSAFromString("0x0123456789001234567890012345678900123456789001234567890012345678")

	assert.Equal(t, err, nil)
	assert.Equal(t, ecdsaPrivKey != nil, true)
	assert.Equal(t, ecdsaPrivKey.PublicKey.X != nil, true)

	// should not start with 0x or 0X
	_, err = LoadECDSAFromString("0123456789001234567890012345678900123456789001234567890012345678")
	assert.Equal(t, err != nil, true)

	// odd length hex string
	_, err = LoadECDSAFromString("0x012345678900123456789001234567890012345678900123456789001234567")
	assert.Equal(t, err != nil, true)

	// invalid length, need 256 bits
	_, err = LoadECDSAFromString("0x01234567890012345678")
	assert.Equal(t, err != nil, true)
}

func Test_ToECDSA(t *testing.T) {
	// Normal case: 256 bits
	ecdsaPrivKey, err := ToECDSA([]byte("01234567890123456789012345678901"))
	assert.Equal(t, err, nil)
	assert.Equal(t, ecdsaPrivKey != nil, true)
	assert.Equal(t, ecdsaPrivKey.PublicKey.X != nil, true)

	// Bad case
	_, err = ToECDSA([]byte("0123456789012"))
	assert.Equal(t, err != nil, true)
}

func Test_FromECDSA(t *testing.T) {
	ecdsaPrivKey, err := ToECDSA([]byte("01234567890123456789012345678901"))
	assert.Equal(t, err, nil)

	bytes := FromECDSA(ecdsaPrivKey)
	assert.Equal(t, len(bytes), 32)
}

func Test_GetAddress(t *testing.T) {
	ecdsaPrivKey, err := ToECDSA([]byte("01234567890123456789012345678901"))
	assert.Equal(t, err, nil)

	addr := GetAddress(&ecdsaPrivKey.PublicKey)
	assert.Equal(t, len(addr), 20)
}

func Test_GenerateRandomAddress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		addr, err := GenerateRandomAddress()
		assert.Equal(t, err, nil)
		assert.Equal(t, len(addr), 20)
	}
}

func Test_MustGenerateRandomAddress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		addr := MustGenerateRandomAddress()
		assert.Equal(t, len(addr), 20)
	}
}

func Test_CreateAddress(t *testing.T) {
	// Same account, different nonce.
	addr1 := CreateAddress(common.BytesToAddress([]byte{1}), 4)
	addr2 := CreateAddress(common.BytesToAddress([]byte{1}), 5)
	assert.Equal(t, false, addr1.Equal(addr2))

	// Different account, same nonce.
	addr1 = CreateAddress(common.BytesToAddress([]byte{2}), 6)
	addr2 = CreateAddress(common.BytesToAddress([]byte{3}), 6)
	assert.Equal(t, false, addr1.Equal(addr2))

	// Different account and nonce.
	addr1 = CreateAddress(common.BytesToAddress([]byte{4}), 7)
	addr2 = CreateAddress(common.BytesToAddress([]byte{5}), 8)
	assert.Equal(t, false, addr1.Equal(addr2))

	// Same account and nonce.
	addr1 = CreateAddress(common.BytesToAddress([]byte{6}), 9)
	addr2 = CreateAddress(common.BytesToAddress([]byte{6}), 9)
	assert.Equal(t, true, addr1.Equal(addr2))
}

func Test_CreateAddress_Shard(t *testing.T) {
	fromAddr := MustGenerateShardAddress(9)
	contractAddr := CreateAddress(*fromAddr, 38)
	assert.Equal(t, contractAddr.Shard(), uint(9))
}

func Test_MustGenerateShardAddress(t *testing.T) {
	addr := MustGenerateShardAddress(5)
	assert.Equal(t, addr.Shard(), uint(5))

	addr = MustGenerateShardAddress(10)
	assert.Equal(t, addr.Shard(), uint(10))
}
