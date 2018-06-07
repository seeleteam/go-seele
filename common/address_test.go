/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_BytesToAddress(t *testing.T) {
	// Create address with single byte.
	b1 := make([]byte, addressLen)
	b1[addressLen-1] = 1
	assert.Equal(t, BytesToAddress([]byte{1}).Bytes(), b1)

	// Create address with multiple bytes.
	b2 := make([]byte, addressLen)
	b2[addressLen-2] = 1
	b2[addressLen-1] = 2
	assert.Equal(t, BytesToAddress([]byte{1, 2}).Bytes(), b2)

	// Create address with too long bytes.
	b3 := make([]byte, addressLen+1)
	for i := 0; i < len(b3); i++ {
		b3[i] = byte(i + 1)
	}
	assert.Equal(t, BytesToAddress(b3).Bytes(), b3[1:])
}

func Test_JsonMarshal(t *testing.T) {
	a := "0xd0c549b022f5a17a8f50a4a448d20ba579d01781"
	addr := HexMustToAddres(a)

	buff, err := json.Marshal(addr)
	assert.Equal(t, err, nil)

	var result Address
	err = json.Unmarshal(buff, &result)
	assert.Equal(t, err, nil)
	assert.Equal(t, result.Bytes(), addr.Bytes())
}

func Test_Address_Type(t *testing.T) {
	hashFunc := func(input interface{}) Hash {
		encoded := SerializePanic(input)
		hash := sha256.Sum256(encoded)
		return BytesToHash(hash[:])
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Equal(t, err, nil)

	addr := PubKeyToAddress(&privKey.PublicKey, hashFunc)
	assert.Equal(t, addr.Type(), AddressTypeExternal)

	contractAddr := addr.CreateContractAddress(38, hashFunc)
	assert.Equal(t, contractAddr.Type(), AddressTypeContract)
}
