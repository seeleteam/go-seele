/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func Test_OdrBlock_Code(t *testing.T) {
	odrBlock := newTestOdrBlock(common.EmptyHash)
	assert.Equal(t, odrBlock.code(), blockRequestCode)
}

func Test_OdrBlock_Handle(t *testing.T) {
	// case 1: empty hash
	ob1 := newTestOdrBlock(common.EmptyHash)
	lp := newTestLightProtocol()

	code, resp := ob1.handle(lp)
	assert.Equal(t, code, blockResponseCode)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.getError())
	assert.Equal(t, resp.getRequestID(), uint32(1))

	// case 2: invalid block hash
	ob2 := newTestOdrBlock(common.StringToHash("1"))
	code, resp = ob2.handle(lp)
	assert.Equal(t, code, blockResponseCode)
	assert.NotNil(t, resp)
	assert.Equal(t, strings.Contains(resp.getError().Error(), "leveldb: not found"), true)
	assert.Equal(t, resp.getRequestID(), uint32(1))

	// case 3: valid block hash
	header := newTestBlockHeader()
	headerHash := header.Hash()
	ob3 := newTestOdrBlock(headerHash)
	code, resp = ob3.handle(lp)
	assert.Equal(t, code, blockResponseCode)
	assert.NotNil(t, resp)
	assert.Nil(t, resp.getError())
	assert.Equal(t, resp.getRequestID(), uint32(1))
}

func Test_OdrBlock_Validate(t *testing.T) {
	ob1 := newTestOdrBlock(common.EmptyHash)

	// case 1: block is nil
	testBlockChain := &TestBlockChain{}
	err := ob1.validate(ob1, testBlockChain.GetStore())
	assert.Nil(t, err)

	// case 2: ErrBlockHashMismatch
	ob2 := newTestOdrBlockWithBlock(common.EmptyHash)
	ob2.Hash = common.StringToHash("1")
	err = ob2.validate(ob2, testBlockChain.GetStore())
	assert.Equal(t, err, types.ErrBlockHashMismatch)

	// case 2: ok
	ob2.Hash = ob2.Block.HeaderHash
	err = ob2.validate(ob2, testBlockChain.GetStore())
	assert.Nil(t, err)
}

func newTestOdrBlock(hash common.Hash) *odrBlock {
	return &odrBlock{
		OdrItem: newTestOdrItem(),
		Hash:    hash,
	}
}

func newTestOdrBlockWithBlock(hash common.Hash) *odrBlock {
	return &odrBlock{
		OdrItem: newTestOdrItem(),
		Hash:    hash,
		Block:   newTestBlock(),
	}
}

func newTestOdrItem() OdrItem {
	return OdrItem{
		ReqID: 1,
	}
}

func newTestLightProtocol() *LightProtocol {
	testBlockChain := &TestBlockChain{}

	return &LightProtocol{
		chain: testBlockChain,
		log:   log.GetLogger("LightChain"),
	}
}

type TestBlockChain struct{}

func (chain *TestBlockChain) GetCurrentState() (*state.Statedb, error) { return nil, nil }

func (chain *TestBlockChain) GetState(root common.Hash) (*state.Statedb, error) { return nil, nil }

func (chain *TestBlockChain) GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error) {
	return nil, nil
}

func (chain *TestBlockChain) GetStore() store.BlockchainStore {
	db, _ := leveldb.NewTestDatabase()
	bcStore := newTestBlockchainDatabase(db)

	// put genesis block
	header := newTestBlockHeader()
	headerHash := header.Hash()
	bcStore.PutBlockHeader(headerHash, header, header.Difficulty, true)
	return bcStore
}

func (chain *TestBlockChain) CurrentHeader() *types.BlockHeader { return nil }

func (chain *TestBlockChain) WriteHeader(*types.BlockHeader) error { return nil }

func newTestBlock() *types.Block {
	header := newTestBlockHeader()
	txs := []*types.Transaction{
		newTestTx(10, 1, 1, true),
		newTestTx(20, 1, 2, true),
		newTestTx(30, 1, 3, true),
	}
	receipts := []*types.Receipt{
		newTestReceipt(),
		newTestReceipt(),
		newTestReceipt(),
	}

	return types.NewBlock(header, txs, receipts, nil)
}

func newTestTx(amount, price, nonce uint64, sign bool) *types.Transaction {
	fromPrivKey, fromAddress := randomAccount()
	toAddress := randomAddress()

	tx, err := types.NewTransaction(fromAddress, toAddress, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(price), nonce)
	if err != nil {
		panic(err)
	}

	if sign {
		tx.Sign(fromPrivKey)
	}

	return tx
}

func randomAccount() (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		panic(keyErr)
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func randomAddress() common.Address {
	_, address := randomAccount()
	return address
}
