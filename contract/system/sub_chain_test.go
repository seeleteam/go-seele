/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_RegisterSubChain(t *testing.T) {
	regInfo := SubChainInfo{
		Name:              "test",
		Version:           "3.8",
		TokenFullName:     "TestCoin",
		TokenShortName:    "TC",
		TokenAmount:       1000000,
		GenesisDifficulty: 8000,
		GenesisAccounts: map[common.Address]*big.Int{
			*crypto.MustGenerateShardAddress(1): big.NewInt(1000),
			*crypto.MustGenerateShardAddress(1): big.NewInt(1000),
		},
	}

	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newTestContext(db, SubChainContractAddress)

	regInfo.Owner = context.tx.Data.From
	regInfo.CreateTimestamp = context.BlockHeader.CreateTimestamp

	encoded, err := json.Marshal(regInfo)
	if err != nil {
		panic(err)
	}

	// register with valid reg info
	result, err := registerSubChain(encoded, context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, nil)

	// query by invalid name
	result, err = querySubChain([]byte("test2"), context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, nil)

	// query by valid name
	result, err = querySubChain([]byte("test"), context)
	assert.Equal(t, err, nil)
	var regInfo2 SubChainInfo
	assert.Equal(t, json.Unmarshal(result, &regInfo2), nil)
	assert.Equal(t, regInfo2, regInfo)

	// create duplicate reg info
	result, err = registerSubChain(encoded, context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, errExists)
}
