/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func newTestContext(db database.Database, contractAddr common.Address) *Context {
	tx := &types.Transaction{
		Data: types.TransactionData{
			From: common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}),
			To:   contractAddr,
		},
	}

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	return NewContext(tx, statedb, newTestBlockHeader())
}

func Test_DomainNameToKey(t *testing.T) {
	// nil domain name
	key, err := domainNameToKey(nil)
	assert.Equal(t, key, common.EmptyHash)
	assert.Equal(t, err, errNameEmpty)

	// empty domain name
	key, err = domainNameToKey([]byte{})
	assert.Equal(t, key, common.EmptyHash)
	assert.Equal(t, err, errNameEmpty)

	// too long domain name
	key, err = domainNameToKey(make([]byte, maxDomainNameLength+1))
	assert.Equal(t, key, common.EmptyHash)
	assert.Equal(t, err, errNameTooLong)

	// valid domain name
	name := []byte("test.seele")
	key, err = domainNameToKey(name)
	assert.Equal(t, key, common.BytesToHash(name))
	assert.Equal(t, err, nil)
}

func Test_CreateDomainName(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newTestContext(db, domainNameContractAddress)

	// valid name
	input := []byte{'a', 'b', 'c'}
	result, err := createDomainName(input, context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, nil)

	// validate statedb
	key, _ := domainNameToKey(input)
	value := context.statedb.GetData(domainNameContractAddress, key)
	assert.Equal(t, value, context.tx.Data.From.Bytes())

	// get domain creator with valid name
	input = []byte{'a', 'b', 'c'}
	result, err = domainNameCreator(input, context)
	assert.Equal(t, result, context.tx.Data.From.Bytes())
	assert.Equal(t, err, nil)

	// get domain creator with invalid name
	input = []byte{'a'}
	result, err = domainNameCreator(input, context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, nil)
}
