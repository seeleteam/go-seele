/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func Test_RegisterSubChain(t *testing.T) {
	regInfo := SubChainInfo{
		Name:              "test",
		Version:           "3.8",
		StaticNodes:       []string{"ip1", "ip2"},
		TokenFullName:     "TestCoin",
		TokenShortName:    "TC",
		TokenAmount:       1000000,
		GenesisDifficulty: 8000,
		GenesisAccounts: map[common.Address]uint64{
			common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}): 1000,
			common.BytesToAddress([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}): 1000,
		},
	}

	encoded, err := json.Marshal(regInfo)
	if err != nil {
		panic(err)
	}

	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newTestContext(db, subChainContractAddress)

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
