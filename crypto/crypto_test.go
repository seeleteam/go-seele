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
	assert.Equal(t, common.GetShardNumber(contractAddr), uint(9))
}

func Test_MustGenerateShardAddress(t *testing.T) {
	addr := MustGenerateShardAddress(5)
	assert.Equal(t, common.GetShardNumber(*addr), uint(5))

	addr = MustGenerateShardAddress(10)
	assert.Equal(t, common.GetShardNumber(*addr), uint(10))
}
