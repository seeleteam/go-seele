/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/factory"
	"github.com/seeleteam/go-seele/consensus/pow"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/core/types"
)

var defaultMinerAddr = common.BytesToAddress([]byte{1})
var seele SeeleBackend = TestSeeleBackend{}

func Test_NewMiner(t *testing.T) {
	miner := createMiner()

	assert.Equal(t, miner != nil, true)
	checkMinerMembers(miner, defaultMinerAddr, seele, t)

	assert.Equal(t, miner.GetCoinbase(), defaultMinerAddr)
	assert.Equal(t, miner.IsMining(), false)
}

func Test_SetCoinbase(t *testing.T) {
	miner := createMiner()

	assert.Equal(t, miner.GetCoinbase(), defaultMinerAddr)

	newAddr := common.BytesToAddress([]byte{2})
	miner.SetCoinbase(newAddr)
	assert.Equal(t, miner.GetCoinbase(), newAddr)
}

func Test_Start(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	miner := createMiner()
	miner.seele = NewTestSeeleBackend(db)
	miner.mining = 1

	err := miner.Start()
	assert.Equal(t, err, ErrMinerIsRunning)

	miner.mining = 0
	miner.canStart = 0
	err = miner.Start()
	assert.Equal(t, err, ErrNodeIsSyncing)

	miner.canStart = 1
	err = miner.Start()
	assert.Equal(t, err, nil)

	assert.Equal(t, miner.stopped, int32(0))
	assert.Equal(t, miner.mining, int32(1))
	miner.Stop()
	assert.Equal(t, miner.stopped, int32(1))
	assert.Equal(t, miner.mining, int32(0))
}

func createMiner() *Miner {
	return NewMiner(defaultMinerAddr, seele, nil, factory.MustGetConsensusEngine(common.Sha256Algorithm))
}

func checkMinerMembers(miner *Miner, addr common.Address, seele SeeleBackend, t *testing.T) {
	assert.Equal(t, miner.coinbase, addr)

	assert.Equal(t, miner.mining, int32(0))
	assert.Equal(t, miner.canStart, int32(1))
	assert.Equal(t, miner.stopped, int32(0))
	assert.Equal(t, miner.seele, seele)
	assert.Equal(t, miner.isFirstDownloader, int32(1))
	assert.Equal(t, miner.isFirstBlockPrepared, int32(0))
	assert.Equal(t, miner.isFirstDownloader, int32(1))
}

// TestSeeleBackend implements the SeeleBackend interface.
type TestSeeleBackend struct {
	txPool     *core.TransactionPool
	debtPool   *core.DebtPool
	blockchain *core.Blockchain
}

func NewTestSeeleBackend(db database.Database) *TestSeeleBackend {
	seeleBeckend := &TestSeeleBackend{}

	seeleBeckend.txPool = newTestPool(core.DefaultTxPoolConfig(), db)
	seeleBeckend.blockchain = newTestBlockchain(db)
	seeleBeckend.debtPool = core.NewDebtPool(seeleBeckend.blockchain, nil)

	return seeleBeckend
}

func (t TestSeeleBackend) TxPool() *core.TransactionPool {
	return t.txPool
}

func (t TestSeeleBackend) DebtPool() *core.DebtPool {
	return t.debtPool
}

func (t TestSeeleBackend) BlockChain() *core.Blockchain {
	return t.blockchain
}

func newTestBlockchain(db database.Database) *core.Blockchain {
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(db))

	genesis := newTestGenesis()
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := core.NewBlockchain(bcStore, db, "", pow.NewEngine(1), nil)
	if err != nil {
		panic(err)
	}

	return bc
}

func newDbInstance(dbPath string) database.Database {
	db, err := leveldb.NewLevelDB(dbPath)
	if err != nil {
		panic(err)
	}

	return db
}

func prepareDbFolder(pathRoot string, subDir string) string {
	dir, err := ioutil.TempDir(pathRoot, subDir)
	if err != nil {
		panic(err)
	}

	return dir
}

func newTestGenesis() *core.Genesis {
	accounts := make(map[common.Address]*big.Int)
	for _, account := range testGenesisAccounts {
		accounts[account.addr] = account.amount
	}

	return core.GetGenesis(core.NewGenesisInfo(accounts, 1, 0, big.NewInt(0), types.PowConsensus, nil))
}

var testGenesisAccounts = []*testAccount{
	newTestAccount(big.NewInt(100000), 0),
	newTestAccount(big.NewInt(100000), 0),
	newTestAccount(big.NewInt(100000), 0),
}

func newTestAccount(amount *big.Int, nonce uint64) *testAccount {
	addr, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	return &testAccount{
		addr:    *addr,
		privKey: privKey,
		amount:  new(big.Int).Set(amount),
		nonce:   nonce,
	}
}

type testAccount struct {
	addr    common.Address
	privKey *ecdsa.PrivateKey
	amount  *big.Int
	nonce   uint64
}

func newTestPool(config *core.TransactionPoolConfig, db database.Database) *core.TransactionPool {
	chain := newTestBlockchain(db)
	txPool := core.NewTransactionPool(*config, chain)

	return txPool
}
