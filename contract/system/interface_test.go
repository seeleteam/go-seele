/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package system

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestContext(db database.Database, contractAddr common.Address) *Context {
	tx := &types.Transaction{
		Data: types.TransactionData{
			From:         *crypto.MustGenerateShardAddress(1),
			To:           contractAddr,
			Amount:       big.NewInt(1),
			GasPrice:     big.NewInt(1),
			AccountNonce: 1,
		},
	}

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	statedb.CreateAccount(contractAddr)
	return NewContext(tx, statedb, newTestBlockHeader())
}

func Test_NewContext(t *testing.T) {
	tx := &types.Transaction{
		Data: types.TransactionData{
			From: common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}),
			To:   DomainNameContractAddress,
		},
	}

	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	blockHeader := newTestBlockHeader()

	context := NewContext(tx, statedb, blockHeader)
	assert.Equal(t, context.tx.Data.To, DomainNameContractAddress)
	assert.Equal(t, context.statedb, statedb)
	assert.Equal(t, context.BlockHeader, blockHeader)
}

func Test_RequiredGas(t *testing.T) {
	c, ok := contracts[DomainNameContractAddress]
	assert.Equal(t, ok, true)

	// input is nil
	gas := c.RequiredGas(nil)
	assert.Equal(t, gas, gasInvalidCommand)

	// CmdCreateDomainName is valid command
	gas = c.RequiredGas([]byte{CmdCreateDomainName})
	assert.Equal(t, gas, gasCreateDomainName)

	// byte(123) is invalid command
	gas = c.RequiredGas([]byte{byte(123)})
	assert.Equal(t, gas, gasInvalidCommand)
}

func Test_Run(t *testing.T) {
	c, ok := contracts[DomainNameContractAddress]
	assert.Equal(t, ok, true)

	// input and context are nil
	arrayByte, err := c.Run(nil, nil)
	assert.Equal(t, err, errInvalidCommand)
	assert.Equal(t, arrayByte == nil, true)

	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	context := newTestContext(db, DomainNameContractAddress)

	// input inclues CmdCreateDomainName command, but not domain name
	arrayByte, err = c.Run([]byte{CmdCreateDomainName}, context)
	assert.Equal(t, err != nil, true)
	assert.Equal(t, arrayByte == nil, true)

	domainName := []byte("seele-test")
	arrayByte, err = c.Run(append([]byte{CmdCreateDomainName}, domainName...), context)
	assert.Equal(t, err, nil)
	assert.Equal(t, arrayByte, context.tx.Data.From.Bytes())

	// byte(123) is invalid command
	arrayByte, err = c.Run([]byte{byte(123)}, context)
	assert.Equal(t, err, errInvalidCommand)
	assert.Equal(t, arrayByte == nil, true)
}

func Test_GetContractByAddress(t *testing.T) {
	c := GetContractByAddress(DomainNameContractAddress)
	assert.Equal(t, c, &contract{domainNameCommands})

	contractAddress := common.BytesToAddress([]byte{123, 1})
	c1 := GetContractByAddress(contractAddress)
	assert.Equal(t, c1, nil)
}
