package native

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_Domain_Name(t *testing.T) {
	statedb, bcStore, address, dispose := preprocessContract(1000*common.SeeleToFan.Uint64(), 38)
	defer dispose()

	header := newTestBlockHeader(address)
	contractAddr := common.BytesToAddress([]byte{1, 1}) // 0x0000000000000000000000000000000000000101
	// CreateDomainName
	byteD := []byte{0}
	input := append(byteD, []byte("seele.fan")...) // 0x007365656c652e66616e
	amount, fee, nonce := big.NewInt(0), big.NewInt(1), uint64(1)
	tx, err := types.NewMessageTransaction(address, contractAddr, amount, fee, nonce, input)
	assert.Equal(t, err, nil)

	nvm := NewNativeVM(tx, statedb, header, bcStore)
	receipt, err := nvm.Process(tx, 0)
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.Failed, false)
	assert.Equal(t, receipt.ContractAddress, contractAddr.Bytes())
	assert.Equal(t, receipt.TxHash, tx.Hash)

	postState, err := statedb.Hash()
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.PostState, postState)

	gasCreateDomainName := uint64(50000) // gas used to create a domain name
	assert.Equal(t, receipt.UsedGas, gasCreateDomainName)
	assert.Equal(t, receipt.TotalFee, new(big.Int).Add(usedGasFee(gasCreateDomainName), fee).Uint64())

	// DomainNameCreator
	byteD = []byte{1}
	input = append(byteD, []byte("seele.fan")...) // 0x017365656c652e66616e
	tx, err = types.NewMessageTransaction(address, contractAddr, amount, fee, nonce, input)
	assert.Equal(t, err, nil)

	nvm = NewNativeVM(tx, statedb, header, bcStore)
	receipt, err = nvm.Process(tx, 1)
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.Failed, false)
	assert.Equal(t, receipt.ContractAddress, contractAddr.Bytes())
	assert.Equal(t, receipt.TxHash, tx.Hash)

	postState, err = statedb.Hash()
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.PostState, postState)

	gasDomainNameCreator := uint64(100000) // gas used to query the creator of given domain name
	assert.Equal(t, receipt.UsedGas, gasDomainNameCreator)
	assert.Equal(t, receipt.TotalFee, new(big.Int).Add(usedGasFee(gasDomainNameCreator), fee).Uint64())

	// Invalid contract address
	contractAddr = common.BytesToAddress([]byte{1, 64})
	tx, err = types.NewMessageTransaction(address, contractAddr, amount, fee, nonce, input)
	assert.Equal(t, err, nil)

	nvm = NewNativeVM(tx, statedb, header, bcStore)
	_, err = nvm.Process(tx, 2)
	assert.Equal(t, err.Error(), "system contract[0x0000000000000000000000000000000000000140] that does not exist")
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
// and a default account with specified balance and nonce.
func preprocessContract(balance, nonce uint64) (*state.Statedb, store.BlockchainStore, common.Address, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	// Create a default account to test contract.
	addr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(addr)
	statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
	statedb.SetNonce(addr, nonce)

	return statedb, store.NewBlockchainDatabase(db), addr, func() {
		dispose()
	}
}

func newTestBlockHeader(coinbase common.Address) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("block previous hash"),
		Creator:           coinbase,
		StateHash:         crypto.MustHash("state root hash"),
		TxHash:            crypto.MustHash("tx root hash"),
		ReceiptHash:       crypto.MustHash("receipt root hash"),
		Difficulty:        big.NewInt(38),
		Height:            666,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Nonce:             10,
		ExtraData:         make([]byte, 0),
	}
}
