/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/pow"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/p2p"
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

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(1), nonce)
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
	}

	return &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: txs,
	}
}

func newTestBlockchain(db database.Database) *core.Blockchain {
	bcStore := store.NewBlockchainDatabase(db)

	genesis := core.GetGenesis(&core.GenesisInfo{})
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := core.NewBlockchain(bcStore, db, "", pow.NewEngine(1), nil)
	if err != nil {
		panic(err)
	}
	return bc
}

func newTestDownloader(db database.Database) *Downloader {
	bc := newTestBlockchain(db)
	d := NewDownloader(bc)
	d.tm = newTaskMgr(d, d.masterPeer, 1, 2)

	return d
}

type TestPeer struct {
	magic uint32
	head  common.Hash
	td    *big.Int // total difficulty
}

// Head retrieves a copy of the current head hash and total difficulty.
func (p *TestPeer) Head() (hash common.Hash, td *big.Int) {
	return p.head, new(big.Int).Set(p.td)
}

// RequestHeadersByHashOrNumber fetches a batch of blocks' headers
func (p *TestPeer) RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error {
	p.magic = magic
	p.head = origin
	return nil
}

// RequestBlocksByHashOrNumber fetches a batch of blocks
func (p *TestPeer) RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error {
	return nil
}

func (p *TestPeer) GetPeerRequestInfo() (uint32, common.Hash, uint64, int) {
	return p.magic, common.EmptyHash, 0, 0
}

func newTestPeer() *TestPeer {
	return &TestPeer{
		head: common.EmptyHash,
		td:   big.NewInt(0),
	}
}

func Test_Downloader_CodeToStr(t *testing.T) {
	assert.Equal(t, CodeToStr(GetBlockHeadersMsg), "downloader.GetBlockHeadersMsg")
	assert.Equal(t, CodeToStr(BlockHeadersMsg), "downloader.BlockHeadersMsg")
	assert.Equal(t, CodeToStr(GetBlocksMsg), "downloader.GetBlocksMsg")
	assert.Equal(t, CodeToStr(BlocksPreMsg), "downloader.BlocksPreMsg")
	assert.Equal(t, CodeToStr(BlocksMsg), "downloader.BlocksMsg")
	assert.Equal(t, CodeToStr(GetBlockHeadersMsg-1), "unknown")
	assert.Equal(t, CodeToStr(BlocksMsg+1), "unknown")
}

func Test_Downloader_GetReadableStatus(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	dl.syncStatus = statusNone
	assert.Equal(t, dl.getReadableStatus(), "NotSyncing")

	dl.syncStatus = statusPreparing
	assert.Equal(t, dl.getReadableStatus(), "Preparing")

	dl.syncStatus = statusFetching
	assert.Equal(t, dl.getReadableStatus(), "Downloading")

	dl.syncStatus = statusCleaning
	assert.Equal(t, dl.getReadableStatus(), "Cleaning")

	dl.syncStatus = statusNone - 1
	assert.Equal(t, dl.getReadableStatus(), "")

	dl.syncStatus = statusCleaning + 1
	assert.Equal(t, dl.getReadableStatus(), "")
}

func Test_Downloader_GetSyncInfo(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	// case 1: all info will be filled for statusFetching
	var info SyncInfo
	dl.syncStatus = statusFetching
	dl.getSyncInfo(&info)
	assert.Equal(t, info.Status, "Downloading")
	assert.Equal(t, len(info.Duration) > 0, true)
	assert.Equal(t, info.StartNum, dl.tm.fromNo)
	assert.Equal(t, info.Amount, dl.tm.toNo-dl.tm.fromNo+1)
	assert.Equal(t, info.Downloaded, dl.tm.downloadedNum)

	// case 2: NotSyncing
	var info1 SyncInfo
	dl.syncStatus = statusNone
	dl.getSyncInfo(&info1)
	assert.Equal(t, info1.Status, "NotSyncing")
	assert.Equal(t, len(info1.Duration), 0)
	assert.Equal(t, info1.StartNum, uint64(0))
	assert.Equal(t, info1.Amount, uint64(0))
	assert.Equal(t, info1.Downloaded, uint64(0))

	// case 3: Preparing
	dl.syncStatus = statusPreparing
	dl.getSyncInfo(&info1)
	assert.Equal(t, info1.Status, "Preparing")

	// case 4: Cleaning
	dl.syncStatus = statusCleaning
	dl.getSyncInfo(&info1)
	assert.Equal(t, info1.Status, "Cleaning")
}

func Test_Downloader_Synchronise(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	peerID := "peerID"
	head := common.EmptyHash
	td := big.NewInt(0)
	localTD := big.NewInt(0)

	// case 1: ErrIsSynchronising
	dl.syncStatus = statusPreparing
	err := dl.Synchronise(peerID, head, td, localTD)
	assert.Equal(t, err, ErrIsSynchronising)

	dl.syncStatus = statusFetching
	err = dl.Synchronise(peerID, head, td, localTD)
	assert.Equal(t, err, ErrIsSynchronising)

	dl.syncStatus = statusCleaning
	err = dl.Synchronise(peerID, head, td, localTD)
	assert.Equal(t, err, ErrIsSynchronising)

	// case 2: peer not found
	dl.syncStatus = statusNone
	err = dl.Synchronise(peerID, head, td, localTD)
	assert.Equal(t, err, errPeerNotFound)
}

func Test_Downloader_FetchHeight(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	testPeer1 := newTestPeer()
	pc1 := newPeerConn(testPeer1, "masterPeer", nil)
	pc1.peer = testPeer1
	go func() {
		header, err := dl.fetchHeight(pc1)
		assert.Equal(t, err, errHashNotMatch)
		assert.Equal(t, header == nil, true)
	}()
	time.Sleep(1000 * time.Millisecond)

	magic, _, _, _ := pc1.peer.GetPeerRequestInfo()
	blockHeadersMsgHeader := newBlockHeadersMsgBody(magic)
	payload := common.SerializePanic(blockHeadersMsgHeader)
	msg := newMessage(BlockHeadersMsg, payload)
	testPeer1.head = crypto.MustHash(blockHeadersMsgHeader.Headers[0])
	pc1.deliverMsg(BlockHeadersMsg, msg)
}

func Test_Downloader_VerifyBlockHeadersMsg(t *testing.T) {
	var msg interface{}

	// case 1: errInvalidPacketReceived
	msg = []*types.BlockHeader{}
	header, err := verifyBlockHeadersMsg(msg, common.EmptyHash)
	assert.Equal(t, err, errInvalidPacketReceived)
	assert.Equal(t, header == nil, true)

	// case 2: errHashNotMatch
	msg = newTestBlockHeaders()
	header, err = verifyBlockHeadersMsg(msg, common.EmptyHash)
	assert.Equal(t, err, errHashNotMatch)
	assert.Equal(t, header == nil, true)

	// case 3: ok
	msg = newTestBlockHeaders()
	header, err = verifyBlockHeadersMsg(msg, crypto.MustHash(msg.([]*types.BlockHeader)[0]))
	assert.Equal(t, err, nil)
	assert.Equal(t, header != nil, true)
	assert.Equal(t, header.PreviousBlockHash, common.StringToHash("PreviousBlockHash"))
	assert.Equal(t, header.Creator, common.EmptyAddress)
	assert.Equal(t, header.Difficulty, big.NewInt(1))
	assert.Equal(t, header.Height, uint64(1))
}

func Test_Downloader_FindCommonAncestorHeight(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	//findCommonAncestorHeight(conn *peerConn, height uint64) (uint64, error)
	pc := testTaskMgrPeerConn("peerID")
	height := uint64(1)

	aHeight, err := dl.findCommonAncestorHeight(pc, height)
	assert.Equal(t, err, nil)
	assert.Equal(t, aHeight, uint64(0))

	// case 2: empty block
	dl.chain = newTestBlockchain(db)
	height = 0
	aHeight, err = dl.findCommonAncestorHeight(pc, height)
	assert.Equal(t, err, nil)
	assert.Equal(t, aHeight, uint64(0))

	// case 2: one block
	genesisHash, err := dl.chain.GetStore().GetBlockHash(0)
	assert.Equal(t, err, nil)
	_, err = dl.chain.GetStore().GetBlock(genesisHash)
	assert.Equal(t, err, nil)
}

func Test_Downloader_GetTop(t *testing.T) {
	var localHeight, height uint64

	localHeight = 0
	height = 1
	top := getTop(localHeight, height)
	assert.Equal(t, top, uint64(0))

	localHeight = 1
	height = 1
	top = getTop(localHeight, height)
	assert.Equal(t, top, uint64(1))

	localHeight = 2
	height = 1
	top = getTop(localHeight, height)
	assert.Equal(t, top, uint64(1))
}

func Test_Downloader_GetMaxFetchAncestry(t *testing.T) {
	var top uint64

	top = 0
	maxFetchAncestry := getMaxFetchAncestry(top)
	assert.Equal(t, maxFetchAncestry, top+1)

	top = 1
	maxFetchAncestry = getMaxFetchAncestry(top)
	assert.Equal(t, maxFetchAncestry, top+1)

	top = uint64(MaxForkAncestry) - 1
	maxFetchAncestry = getMaxFetchAncestry(top)
	assert.Equal(t, maxFetchAncestry, top+1)

	top = uint64(MaxForkAncestry)
	maxFetchAncestry = getMaxFetchAncestry(top)
	assert.Equal(t, maxFetchAncestry, uint64(MaxForkAncestry))

	top = uint64(MaxForkAncestry) + 1
	maxFetchAncestry = getMaxFetchAncestry(top)
	assert.Equal(t, maxFetchAncestry, uint64(MaxForkAncestry))
}

func Test_Downloader_GetFetchCount(t *testing.T) {
	var maxFetchAncestry, cmpCount uint64

	maxFetchAncestry = 0
	cmpCount = 0
	fetchCount := getFetchCount(maxFetchAncestry, cmpCount)
	assert.Equal(t, fetchCount, uint64(0))

	maxFetchAncestry = uint64(MaxHeaderFetch) - 1
	cmpCount = 0
	fetchCount = getFetchCount(maxFetchAncestry, cmpCount)
	assert.Equal(t, fetchCount, maxFetchAncestry)

	maxFetchAncestry = uint64(MaxHeaderFetch)
	cmpCount = 0
	fetchCount = getFetchCount(maxFetchAncestry, cmpCount)
	assert.Equal(t, fetchCount, uint64(MaxHeaderFetch))

	maxFetchAncestry = uint64(MaxHeaderFetch) + 1
	cmpCount = 0
	fetchCount = getFetchCount(maxFetchAncestry, cmpCount)
	assert.Equal(t, fetchCount, uint64(MaxHeaderFetch))
}

func Test_Downloader_GetPeerBlockHaders(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	// case 1: received valid header
	testPeer1 := newTestPeer()
	pc1 := newPeerConn(testPeer1, "masterPeer", nil)
	pc1.peer = testPeer1
	go func() {
		headers, err := dl.getPeerBlockHaders(pc1, 0, 1)
		assert.Equal(t, err, nil)
		assert.Equal(t, len(headers), 1)
		assert.Equal(t, headers[0].PreviousBlockHash, common.StringToHash("PreviousBlockHash"))
		assert.Equal(t, headers[0].Creator, common.EmptyAddress)
		assert.Equal(t, headers[0].Difficulty, big.NewInt(1))
		assert.Equal(t, headers[0].Height, uint64(1))
	}()
	time.Sleep(500 * time.Millisecond)

	magic, _, _, _ := pc1.peer.GetPeerRequestInfo()
	blockHeadersMsgHeader := newBlockHeadersMsgBody(magic)
	payload := common.SerializePanic(blockHeadersMsgHeader)
	msg := newMessage(BlockHeadersMsg, payload)
	pc1.deliverMsg(BlockHeadersMsg, msg)

	// case 2: received valid header with empty payload
	testPeer2 := newTestPeer()
	pc2 := newPeerConn(testPeer2, "masterPeer", nil)
	pc2.peer = testPeer2
	go func() {
		headers, err := dl.getPeerBlockHaders(pc2, 0, 1)
		assert.Equal(t, err, errInvalidAncestor)
		assert.Equal(t, len(headers), 0)
	}()
	time.Sleep(500 * time.Millisecond)

	magic, _, _, _ = pc2.peer.GetPeerRequestInfo()
	blockHeadersMsgHeader = newBlockHeadersMsgWithEmptyBody(magic)
	payload = common.SerializePanic(blockHeadersMsgHeader)
	msg = newMessage(BlockHeadersMsg, payload)
	pc2.deliverMsg(BlockHeadersMsg, msg)

	// case 3: close channel before receiving data
	testPeer3 := newTestPeer()
	pc3 := newPeerConn(testPeer3, "masterPeer", nil)
	pc3.peer = testPeer3
	go func() {
		headers, err := dl.getPeerBlockHaders(pc3, 0, 1)
		assert.Equal(t, err, errPeerQuit)
		assert.Equal(t, len(headers), 0)
	}()
	time.Sleep(500 * time.Millisecond)
	pc3.close()

	// case 4: cancel channel before receiving data
	testPeer4 := newTestPeer()
	pc4 := newPeerConn(testPeer4, "masterPeer", nil)
	pc4.peer = testPeer4
	go func() {
		headers, err := dl.getPeerBlockHaders(pc4, 0, 1)
		assert.Equal(t, err, errReceivedQuitMsg)
		assert.Equal(t, len(headers), 0)
	}()
	time.Sleep(500 * time.Millisecond)
	close(dl.cancelCh)
}

func Test_Downloader_IsAncenstorFound(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	headers := newTestBlockHeaders()
	found, cmpHeight, err := dl.isAncenstorFound(headers)
	assert.Equal(t, found, true)
	assert.Equal(t, cmpHeight, uint64(0))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)

	headers = nil
	found, cmpHeight, err = dl.isAncenstorFound(headers)
	assert.Equal(t, found, false)
	assert.Equal(t, cmpHeight, uint64(0))
	assert.Equal(t, err, nil)
}

func Test_Downloader_RegisterPeerAndUnRegisterPeer(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	testPeer := newTestPeer()

	// init the peers is empty
	assert.Equal(t, len(dl.peers), 0)

	// register 1 peer
	dl.RegisterPeer("peerID", testPeer)
	assert.Equal(t, len(dl.peers), 1)

	// register duplicated peer
	dl.RegisterPeer("peerID", testPeer)
	assert.Equal(t, len(dl.peers), 1)

	// register one more peer
	dl.RegisterPeer("peerID1", testPeer)
	assert.Equal(t, len(dl.peers), 2)

	// unregister peerID
	dl.UnRegisterPeer("peerID")
	assert.Equal(t, len(dl.peers), 1)

	// unregister non-exist peer
	dl.UnRegisterPeer("non-exist peer")
	assert.Equal(t, len(dl.peers), 1)

	// unregister peerID1
	dl.UnRegisterPeer("peerID1")
	assert.Equal(t, len(dl.peers), 0)

	// duplicatedly unregister peerID1
	dl.UnRegisterPeer("peerID1")
	assert.Equal(t, len(dl.peers), 0)
}

func Test_Downloader_DeliverMsg(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	blockHeadersMsgHeader := newBlockHeadersMsgBody(uint32(1))
	payload := common.SerializePanic(blockHeadersMsgHeader)
	msg := newMessage(BlockHeadersMsg, payload)

	pc := testTaskMgrPeerConn("peerID")
	dl.peers["peerID"] = pc
	dl.peers["peerID"].waitingMsgMap[BlockHeadersMsg] = make(chan *p2p.Message)
	cancelCh := make(chan struct{})
	go func() {
		ret, err := dl.peers["peerID"].waitMsg(uint32(1), BlockHeadersMsg, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
	}()
	time.Sleep(500 * time.Millisecond)

	dl.DeliverMsg("peerID", msg)
	assert.Equal(t, len(dl.peers), 1)

	dl.UnRegisterPeer("peerID")
	assert.Equal(t, len(dl.peers), 0)
}

func Test_Downloader_Terminate(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)

	dl.Terminate()
	dl.Terminate()
	assert.Equal(t, dl.cancelCh == nil, true)
}

func Test_Downloader_PeerDownload(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)
	taskMgr := newTestTaskMgr(dl, db)

	// case 1: non-master peer
	testPeer1 := newTestPeer()
	pc1 := newPeerConn(testPeer1, "test", nil)
	go func() {
		dl.sessionWG.Add(1)
		dl.peerDownload(pc1, taskMgr)
	}()

	time.Sleep(300 * time.Millisecond)
	close(dl.cancelCh)

	// case 2: master peer
	dl.masterPeer = "masterPeer"
	testPeer2 := newTestPeer()
	pc2 := newPeerConn(testPeer2, "masterPeer", nil)
	go func() {
		dl.sessionWG.Add(1)
		dl.cancelCh = make(chan struct{})
		dl.peerDownload(pc2, taskMgr)
	}()
	time.Sleep(300 * time.Millisecond)

	// case 3: BlockHeadersMsg
	testPeer3 := newTestPeer()
	pc3 := newPeerConn(testPeer3, "masterPeer", nil)
	pc3.peer = testPeer3
	go func() {
		dl.sessionWG.Add(1)
		dl.cancelCh = make(chan struct{})
		dl.peerDownload(pc3, taskMgr)
	}()
	time.Sleep(100 * time.Millisecond)

	magic, _, _, _ := pc3.peer.GetPeerRequestInfo()
	blockHeadersMsgHeader := newBlockHeadersMsgBody(magic)
	payload := common.SerializePanic(blockHeadersMsgHeader)
	msg := newMessage(BlockHeadersMsg, payload)
	pc3.deliverMsg(BlockHeadersMsg, msg)
}

func Test_Downloader_ProcessBlocks(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)
	dl.chain = newTestBlockchain(db)

	headInfos := []*downloadInfo{newDownloadInfo(1, taskStatusWaitProcessing)}
	dl.processBlocks(headInfos)
	assert.Equal(t, headInfos[0].status, taskStatusWaitProcessing)
}

func Test_findCommonAncestorHeight_localHeightIsZero(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()
	dl := newTestDownloader(db)
	height := uint64(1000)
	var testPeer *TestPeer
	p := newPeerConn(testPeer, "test", nil)
	ancestorHeight, err := dl.findCommonAncestorHeight(p, height)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint64(0), ancestorHeight)
}
