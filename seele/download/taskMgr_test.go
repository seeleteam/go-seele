/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"

	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_TaskMgr_NewPeerHeadInfo(t *testing.T) {
	p := newPeerHeadInfo()
	assert.Equal(t, p != nil, true)
	assert.Equal(t, len(p.headers), 0)
}

func Test_TaskMgr_NewTaskMgrAndRun(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	d := newTestDownloader(db)
	taskMgr := newTestTaskMgr(d, db)
	defer taskMgr.close()

	assert.Equal(t, taskMgr != nil, true)
	assert.Equal(t, taskMgr.log, d.log)
	assert.Equal(t, taskMgr.downloader, d)
	assert.Equal(t, taskMgr.fromNo, from)
	assert.Equal(t, taskMgr.toNo, to)
	assert.Equal(t, taskMgr.toNo, to)
	assert.Equal(t, taskMgr.curNo, from)
	assert.Equal(t, taskMgr.downloadedNum, uint64(0))
	assert.Equal(t, taskMgr.masterPeer, masterPeer)

	assert.Equal(t, len(taskMgr.peersHeaderMap), 0)
	assert.Equal(t, len(taskMgr.downloadInfoList), 0)

}

func Test_TaskMgr_GetWaitProcessingBlocks(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	d := newTestDownloader(db)
	taskMgr := newTestTaskMgr(d, db)
	defer taskMgr.onPeerQuit(masterPeer)
	defer taskMgr.close()

	// empty block
	di := taskMgr.getWaitProcessingBlocks()
	assert.Equal(t, len(di), 0)
	assert.Equal(t, taskMgr.curNo, uint64(0))
	assert.Equal(t, taskMgr.isDone(), false)

	// add one block that needs to be processed
	taskMgr.downloadInfoList = []*downloadInfo{newDownloadInfo(1, taskStatusWaitProcessing)}
	di = taskMgr.getWaitProcessingBlocks()
	assert.Equal(t, len(di), 1)
	assert.Equal(t, taskMgr.curNo, uint64(1))
	assert.Equal(t, taskMgr.isDone(), true)

	// add another one block that needs to be processed
	taskMgr.downloadInfoList = append(taskMgr.downloadInfoList, newDownloadInfo(2, taskStatusWaitProcessing))
	di = taskMgr.getWaitProcessingBlocks()
	assert.Equal(t, len(di), 1)
	assert.Equal(t, taskMgr.curNo, uint64(2))
	assert.Equal(t, taskMgr.isDone(), false)
}

func Test_TaskMgr_GetReqHeaderInfo(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	d := newTestDownloader(db)
	taskMgr := newTestTaskMgr(d, db)
	defer taskMgr.onPeerQuit(masterPeer)
	defer taskMgr.close()

	// case 1: init
	pc := testTaskMgrPeerConn("testPeerID")
	startNo, amount := taskMgr.getReqHeaderInfo(pc)
	assert.Equal(t, startNo, uint64(0))
	assert.Equal(t, amount, 1)

	// case 2: MaxHeaderFetch
	taskMgr.peersHeaderMap["testPeerID"] = newPeerHeadInfos(2)
	startNo, amount = taskMgr.getReqHeaderInfo(pc)
	assert.Equal(t, startNo, uint64(3))
	assert.Equal(t, amount, MaxHeaderFetch)

	// case 3: master peer
	pc = testTaskMgrPeerConn("masterPeer")
	startNo, amount = taskMgr.getReqHeaderInfo(pc)
	assert.Equal(t, startNo, uint64(0))
	assert.Equal(t, amount, 1)
}

func Test_TaskMgr_GetReqBlocks(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	d := newTestDownloader(db)
	taskMgr := newTestTaskMgr(d, db)
	defer taskMgr.onPeerQuit(masterPeer)
	defer taskMgr.close()

	pc := testTaskMgrPeerConn("testPeerID")
	startNo, amount := taskMgr.getReqBlocks(pc)
	assert.Equal(t, startNo, uint64(0))
	assert.Equal(t, amount, 0)

	taskMgr.peersHeaderMap["testPeerID"] = newPeerHeadInfos(3)
	taskMgr.downloadInfoList = []*downloadInfo{newDownloadInfo(1, taskStatusIdle), newDownloadInfo(2, taskStatusIdle), newDownloadInfo(3, taskStatusIdle)}
	taskMgr.curNo = 0
	startNo, amount = taskMgr.getReqBlocks(pc)
	assert.Equal(t, startNo, uint64(0))
	assert.Equal(t, amount, 0)
}

func Test_TaskMgr_DeliverHeaderMsg(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	d := newTestDownloader(db)
	taskMgr := newTestTaskMgr(d, db)
	defer taskMgr.onPeerQuit(masterPeer)
	defer taskMgr.close()

	// case 1: headers is nil
	err := taskMgr.deliverHeaderMsg(masterPeer, nil)
	assert.Equal(t, err, nil)

	// case 2: errMasterHeadersNotMatch
	err = taskMgr.deliverHeaderMsg(masterPeer, newTestBlockHeaders())
	assert.Equal(t, err, errMasterHeadersNotMatch)

	// case 3: errHeadInfoNotFound
	taskMgr.downloadInfoList = []*downloadInfo{newDownloadInfo(1, taskStatusIdle)}
	err = taskMgr.deliverHeaderMsg(masterPeer, newTestBlockHeaders())
	assert.Equal(t, err, errHeadInfoNotFound)

	// case 3: ok
	taskMgr.peersHeaderMap[masterPeer] = newPeerHeadInfos(1)
	taskMgr.downloadInfoList = []*downloadInfo{newDownloadInfo(1, taskStatusIdle)}
	err = taskMgr.deliverHeaderMsg(masterPeer, newTestBlockHeaders())
	assert.Equal(t, err, nil)
}

func Test_TaskMgr_DeliverBlockMsg(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	d := newTestDownloader(db)
	taskMgr := newTestTaskMgr(d, db)
	defer taskMgr.onPeerQuit(masterPeer)
	defer taskMgr.close()

	// case 1: not block
	taskMgr.deliverBlockMsg(masterPeer, nil)
	assert.Equal(t, taskMgr.downloadedNum, uint64(0))

	// case: headInfo.peerID != peerID
	taskMgr.downloadInfoList = []*downloadInfo{newDownloadInfo(0, taskStatusIdle), newDownloadInfo(1, taskStatusIdle)}
	taskMgr.deliverBlockMsg(masterPeer, []*types.Block{taskMgr.downloadInfoList[0].block})
	assert.Equal(t, taskMgr.downloadInfoList[0].status, taskStatusIdle)
	assert.Equal(t, taskMgr.downloadInfoList[1].status, taskStatusIdle)
	assert.Equal(t, taskMgr.downloadedNum, uint64(0))

	// case 3: ok
	taskMgr.downloadInfoList[0].block.Header.Height = 0
	taskMgr.downloadInfoList[1].block.Header.Height = 1
	taskMgr.deliverBlockMsg("peerID", []*types.Block{taskMgr.downloadInfoList[0].block, taskMgr.downloadInfoList[1].block})
	assert.Equal(t, taskMgr.downloadInfoList[0].status, taskStatusWaitProcessing)
	assert.Equal(t, taskMgr.downloadInfoList[1].status, taskStatusWaitProcessing)
	assert.Equal(t, taskMgr.downloadedNum, uint64(2))
}

var (
	masterPeer = "masterPeer"
	from       = uint64(0)
	to         = uint64(0)
)

func newTestTaskMgr(d *Downloader, db database.Database) *taskMgr {
	taskMgr := newTaskMgr(d, masterPeer, from, to)

	return taskMgr
}

func newDownloadInfo(height uint64, status int) *downloadInfo {
	return &downloadInfo{
		header: newTestBlockHeaderWithHeight(height),
		block:  newTestBlocks()[0],
		peerID: "peerID",
		status: status,
	}
}

func testTaskMgrPeerConn(peerID string) *peerConn {
	var peer TestDownloadPeer
	pc := newPeerConn(peer, peerID, nil)

	return pc
}

func newPeerHeadInfos(num int) *peerHeadInfo {
	p := newPeerHeadInfo()

	for i := 0; i < num; i++ {
		p.headers[uint64(i)] = newTestBlockHeaderWithHeight(uint64(i + 1))
	}
	p.maxNo = uint64(num)

	return p
}

func newTestBlockHeaderWithHeight(height uint64) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           common.EmptyAddress,
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            height,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Witness:           common.CopyBytes([]byte("witness")),
		ExtraData:         common.CopyBytes([]byte("ExtraData")),
	}
}
