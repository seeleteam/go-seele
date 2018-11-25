/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/factory"
	"github.com/seeleteam/go-seele/core"
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

	miner := createMiner()
	miner.seele = NewTestSeeleBackend()
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

func NewTestSeeleBackend() *TestSeeleBackend {
	return NewTestSeeleBackendWithVerifier(nil)
}

func NewTestSeeleBackendWithVerifier(verifier types.DebtVerifier) *TestSeeleBackend {
	seeleBeckend := &TestSeeleBackend{}

	seeleBeckend.txPool = newTestPool(core.DefaultTxPoolConfig())
	seeleBeckend.blockchain = core.NewTestBlockchainWithVerifier(verifier)
	seeleBeckend.debtPool = core.NewDebtPool(seeleBeckend.blockchain, verifier)

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

func prepareDbFolder(pathRoot string, subDir string) string {
	dir, err := ioutil.TempDir(pathRoot, subDir)
	if err != nil {
		panic(err)
	}

	return dir
}

func newTestPool(config *core.TransactionPoolConfig) *core.TransactionPool {
	chain := core.NewTestBlockchain()
	txPool := core.NewTransactionPool(*config, chain)

	return txPool
}
