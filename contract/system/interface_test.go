package system

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestContext(db database.Database, contractAddr common.Address, blockHeader *types.BlockHeader) *Context {
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

	return NewContext(tx, statedb, blockHeader)
}

func Test_NewContext(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	blockHeader := newTestBlockHeader()

	context := newTestContext(db, domainNameContractAddress, blockHeader)
	assert.Equal(t, context.tx.Data.To, domainNameContractAddress)
	assert.Equal(t, context.BlockHeader, blockHeader)
}

func Test_RequiredGas(t *testing.T) {
	c, ok := contracts[domainNameContractAddress]
	assert.Equal(t, ok, true)

	// input is nil
	gas := c.RequiredGas(nil)
	assert.Equal(t, gas, gasInvalidCommand)

	// cmdCreateDomainName is valid command
	gas = c.RequiredGas([]byte{cmdCreateDomainName})
	assert.Equal(t, gas, gasCreateDomainName)

	// byte(123) is invalid command
	gas = c.RequiredGas([]byte{byte(123)})
	assert.Equal(t, gas, gasInvalidCommand)
}

func Test_Run(t *testing.T) {
	c, ok := contracts[domainNameContractAddress]
	assert.Equal(t, ok, true)

	// input and context are nil
	arrayByte, err := c.Run(nil, nil)
	assert.Equal(t, err, errInvalidCommand)
	assert.Equal(t, arrayByte == nil, true)

	// context is nil
	arrayByte, err = c.Run([]byte{cmdCreateDomainName}, nil)
	assert.Equal(t, err, errInvalidContext)
	assert.Equal(t, arrayByte == nil, true)

	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	blockHeader := newTestBlockHeader()

	context := newTestContext(db, domainNameContractAddress, blockHeader)

	// input inclues cmdCreateDomainName command, but not domain name
	arrayByte, err = c.Run([]byte{cmdCreateDomainName}, context)
	assert.Equal(t, err != nil, true)
	assert.Equal(t, arrayByte == nil, true)

	arrayByte, err = c.Run([]byte{cmdCreateDomainName, byte(1), byte(2)}, context)
	assert.Equal(t, err, nil)
	assert.Equal(t, arrayByte == nil, true)

	// byte(123) is invalid command
	arrayByte, err = c.Run([]byte{byte(123)}, context)
	assert.Equal(t, err, errInvalidCommand)
	assert.Equal(t, arrayByte == nil, true)
}

func Test_GetContractByAddress(t *testing.T) {
	c := GetContractByAddress(domainNameContractAddress)
	assert.Equal(t, c, &contract{domainNameCommands})

	contractAddress := common.BytesToAddress([]byte{123, 1})
	c1 := GetContractByAddress(contractAddress)
	assert.Equal(t, c1, nil)
}
