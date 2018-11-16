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
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BytesToAddress(t *testing.T) {
	// Create address with single byte.
	b1 := make([]byte, AddressLen)
	b1[AddressLen-1] = 1
	assert.Equal(t, BytesToAddress([]byte{1}).Bytes(), b1)

	// Create address with multiple bytes.
	b2 := make([]byte, AddressLen)
	b2[AddressLen-2] = 1
	b2[AddressLen-1] = 2
	assert.Equal(t, BytesToAddress([]byte{1, 2}).Bytes(), b2)

	// Create address with too long bytes.
	b3 := make([]byte, AddressLen+1)
	for i := 0; i < len(b3); i++ {
		b3[i] = byte(i + 1)
	}
	assert.Equal(t, BytesToAddress(b3).Bytes(), b3[1:])
}

func Test_ToHexAndEqualAndIsEmpty(t *testing.T) {
	// Hex
	b1 := make([]byte, AddressLen)
	b1[AddressLen-1] = 1
	addr1 := BytesToAddress([]byte{1})
	assert.Equal(t, addr1.Hex(), "0x0000000000000000000000000000000000000001")

	// Equal
	b2 := make([]byte, AddressLen)
	b2[AddressLen-1] = 1
	addr2 := BytesToAddress([]byte{1})
	assert.Equal(t, addr1.Equal(addr2), true)

	// IsEmpty
	emptyAddr := EmptyAddress
	assert.Equal(t, emptyAddr.IsEmpty(), true)
}

func Test_Big(t *testing.T) {
	b1 := make([]byte, AddressLen)
	b1[AddressLen-1] = 1
	addr1 := BytesToAddress([]byte{1})

	assert.Equal(t, addr1.Big(), big.NewInt(1))
}

func Test_Shard(t *testing.T) {
	b1 := make([]byte, AddressLen)
	b1[AddressLen-1] = 1
	addr1 := BytesToAddress([]byte{1})

	assert.Equal(t, addr1.Shard(), uint(1))
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

	contractAddr = BytesToAddress([]byte{1, 255})
	assert.Equal(t, contractAddr.Type(), AddressTypeReserved)
}

func Test_IsSystemAddress(t *testing.T) {
	contractAddr := BytesToAddress([]byte{5, 0})
	assert.Equal(t, contractAddr.IsReserved(), false)

	contractAddr = BytesToAddress([]byte{0})
	assert.Equal(t, contractAddr.IsReserved(), false)

	contractAddr = BytesToAddress([]byte{1, 255})
	assert.Equal(t, contractAddr.IsReserved(), true)

	contractAddr = BytesToAddress([]byte{1})
	assert.Equal(t, contractAddr.IsReserved(), true)
}

func Test_Address_InvalidType(t *testing.T) {
	// miss the last 4 bits - address type, length = 41
	hexAddrBase := "0x4c10f2cd2159bb432094e3be7e17904c2b4aeb2"

	types := []string{"0", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "A", "B", "C", "D", "E", "F"}
	for _, addrType := range types {
		addr := hexAddrBase + addrType
		_, err := HexToAddress(addr)
		assert.True(t, strings.Contains(err.Error(), "invalid address type"))
	}

	// empty address is always valid
	assert.NoError(t, EmptyAddress.Validate())
}
