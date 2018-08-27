/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"testing"

	"github.com/magiconair/properties/assert"
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

	return NewContext(tx, statedb)
}

func Test_DomainNameContract_RequiredGas(t *testing.T) {
	contract := domainNameContract{}

	// empty input
	assert.Equal(t, contract.RequiredGas(nil), gasInvalidCommand)
	assert.Equal(t, contract.RequiredGas([]byte{}), gasInvalidCommand)

	// invalid command
	assert.Equal(t, contract.RequiredGas([]byte{123}), gasInvalidCommand)

	// create domain name
	assert.Equal(t, contract.RequiredGas([]byte{cmdCreateDomainName}), gasCreateDomainName)

	// get domain creator
	assert.Equal(t, contract.RequiredGas([]byte{cmdDomainNameCreator}), gasDomainNameCreator)
}

func Test_DomainNameContract_DomainNameToKey(t *testing.T) {
	// nil domain name
	key, err := domainNameToKey(nil)
	assert.Equal(t, key, common.EmptyHash)
	assert.Equal(t, err, errDomainNameEmpty)

	// empty domain name
	key, err = domainNameToKey([]byte{})
	assert.Equal(t, key, common.EmptyHash)
	assert.Equal(t, err, errDomainNameEmpty)

	// too long domain name
	key, err = domainNameToKey(make([]byte, maxDomainNameLength+1))
	assert.Equal(t, key, common.EmptyHash)
	assert.Equal(t, err, errDomainNameTooLong)

	// valid domain name
	name := []byte("test.seele")
	key, err = domainNameToKey(name)
	assert.Equal(t, key, common.BytesToHash(name))
	assert.Equal(t, err, nil)
}

func Test_DomainNameContract_CreateDomainName(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newTestContext(db, domainNameContractAddress)
	contract := domainNameContract{}

	// valid name
	input := []byte{cmdCreateDomainName, 'a', 'b', 'c'}
	result, err := contract.Run(input, context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, nil)

	// validate statedb
	key, _ := domainNameToKey(input[1:])
	value := context.statedb.GetData(domainNameContractAddress, key)
	assert.Equal(t, value, context.tx.Data.From.Bytes())

	// get domain creator with valid name
	input = []byte{cmdDomainNameCreator, 'a', 'b', 'c'}
	result, err = contract.Run(input, context)
	assert.Equal(t, result, context.tx.Data.From.Bytes())
	assert.Equal(t, err, nil)

	// get domain creator with invalid name
	input = []byte{cmdDomainNameCreator, 'a'}
	result, err = contract.Run(input, context)
	assert.Equal(t, result, []byte(nil))
	assert.Equal(t, err, nil)
}
