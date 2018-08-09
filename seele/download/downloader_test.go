/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func newTestTx(t *testing.T, amount int64, nonce uint64) *types.Transaction {
	fromPrivKey, fromAddress := randomAccount(t)
	_, toAddress := randomAccount(t)

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(0), nonce)
	tx.Sign(fromPrivKey)

	return tx
}

func newTestBlock(t *testing.T, parentHash common.Hash, height uint64, db database.Database, nonce uint64, difficulty int64) *types.Block {
	txs := []*types.Transaction{
		newTestTx(t, 1, 1),
		newTestTx(t, 2, 2),
		newTestTx(t, 3, 3),
	}

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		t.Fatal()
	}

	for _, tx := range txs {
		statedb.CreateAccount(tx.Data.From)
		statedb.SetBalance(tx.Data.From, big.NewInt(10))
		statedb.SetNonce(tx.Data.From, nonce)
	}

	batch := db.NewBatch()
	stateHash, err := statedb.Commit(batch)
	if err != nil {
		t.Fatal()
	}

	if err = batch.Commit(); err != nil {
		t.Fatal()
	}

	header := &types.BlockHeader{
		PreviousBlockHash: parentHash,
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         stateHash,
		TxHash:            types.MerkleRootHash(txs),
		Height:            height,
		Difficulty:        big.NewInt(difficulty),
		CreateTimestamp:   big.NewInt(1),
		Nonce:             10,
	}

	return &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: txs,
	}
}

func newTestBlockchain(db database.Database) *core.Blockchain {
	bcStore := store.NewBlockchainDatabase(db)

	genesis := core.GetGenesis(core.GenesisInfo{})
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := core.NewBlockchain(bcStore, db, "")
	if err != nil {
		panic(err)
	}
	return bc
}

func newTestDownloader(db database.Database) *Downloader {
	bc := newTestBlockchain(db)
	return NewDownloader(bc)
}

type TestPeer struct {
	head common.Hash
	td   *big.Int // total difficulty
}

// Head retrieves a copy of the current head hash and total difficulty.
func (p TestPeer) Head() (hash common.Hash, td *big.Int) {
	return hash, new(big.Int).Set(p.td)
}

// RequestHeadersByHashOrNumber fetches a batch of blocks' headers
func (p TestPeer) RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error {
	return nil
}

// RequestBlocksByHashOrNumber fetches a batch of blocks
func (p TestPeer) RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error {
	return nil
}

func Test_findCommonAncestorHeight_localHeightIsZero(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)
	height := uint64(1000)
	var testPeer TestPeer
	p := newPeerConn(testPeer, "test", nil)
	ancestorHeight, err := dl.findCommonAncestorHeight(p, height)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint64(0), ancestorHeight)
}
