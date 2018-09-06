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
	"github.com/seeleteam/go-seele/p2p"
	"github.com/stretchr/testify/assert"
)

// TestDownloadPeer implements the inferace of Peer
type TestDownloadPeer struct{}

func (s TestDownloadPeer) Head() (common.Hash, *big.Int) {
	return common.EmptyHash, nil
}

func (s TestDownloadPeer) RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error {
	return nil
}

func (s TestDownloadPeer) RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error {
	return nil
}

func (s TestDownloadPeer) GetPeerRequestInfo() (uint32, common.Hash, uint64, int) {
	return 0, common.EmptyHash, 0, 0
}

func Test_Download_NewPeerConnAndClose(t *testing.T) {
	var peer TestDownloadPeer
	peerID := "testPeerID"

	pc := newPeerConn(peer, peerID, nil)
	defer pc.close()

	assert.Equal(t, pc != nil, true)
	assert.Equal(t, pc.peerID, peerID)
	assert.Equal(t, pc.peer, peer)
}

func Test_Download_WaitMsg(t *testing.T) {
	magic := uint32(1)
	msgCode := uint16(0)
	cancelCh := make(chan struct{})

	// quit message
	pc := testPeerConn()
	go func() {
		_, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, errPeerQuit)
	}()

	time.Sleep(100 * time.Millisecond)
	pc.close()

	// cancel message
	pc = testPeerConn()
	go func() {
		_, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, errReceivedQuitMsg)
	}()

	time.Sleep(100 * time.Millisecond)
	close(cancelCh)

	// BlockHeadersMsg
	msgCode = BlockHeadersMsg
	cancelCh = make(chan struct{})
	blockHeadersMsgHeader := newBlockHeadersMsgBody(magic)
	payload := common.SerializePanic(blockHeadersMsgHeader)
	msg := newMessage(BlockHeadersMsg, payload)
	pc = testPeerConn()

	go func() {
		ret, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
		assert.Equal(t, ret, blockHeadersMsgHeader.Headers)
	}()

	time.Sleep(100 * time.Millisecond)
	pc.waitingMsgMap[BlockHeadersMsg] <- msg

	// BlocksMsg
	msgCode = BlocksMsg
	cancelCh = make(chan struct{})
	blocksMsgHeader := newBlocksMsgBody(magic)
	payload = common.SerializePanic(blocksMsgHeader)
	msg = newMessage(BlockHeadersMsg, payload)
	pc = testPeerConn()

	go func() {
		ret, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
		assert.Equal(t, ret, blocksMsgHeader.Blocks)
	}()

	time.Sleep(100 * time.Millisecond)
	pc.waitingMsgMap[BlocksMsg] <- msg

	// BlocksMsg sent by deliverMsg
	msgCode = BlocksMsg
	cancelCh = make(chan struct{})
	pc = testPeerConn()

	go func() {
		ret, err := pc.waitMsg(magic, msgCode, cancelCh)
		assert.Equal(t, err, nil)
		assert.Equal(t, ret != nil, true)
		assert.Equal(t, ret, blocksMsgHeader.Blocks)
	}()

	time.Sleep(100 * time.Millisecond)
	pc.deliverMsg(msgCode, msg)
}

func testPeerConn() *peerConn {
	var peer TestDownloadPeer
	peerID := "testPeerID"
	pc := newPeerConn(peer, peerID, nil)

	return pc
}

func newMessage(code uint16, payload []byte) *p2p.Message {
	return &p2p.Message{
		Code:       code,
		Payload:    payload,
		ReceivedAt: time.Now(),
	}
}

func newBlockHeadersMsgBody(magic uint32) *BlockHeadersMsgBody {
	return &BlockHeadersMsgBody{
		Magic:   magic,
		Headers: newTestBlockHeaders(),
	}
}

func newTestBlockHeaders() []*types.BlockHeader {
	return []*types.BlockHeader{
		newTestBlockHeader(),
	}
}

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           common.EmptyAddress,
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Nonce:             1,
		ExtraData:         common.CopyBytes([]byte("")),
	}
}

func newBlocksMsgBody(magic uint32) *BlocksMsgBody {
	return &BlocksMsgBody{
		Magic:  magic,
		Blocks: newTestBlocks(),
	}
}

func newTestBlocks() []*types.Block {
	headers := newTestBlockHeaders()
	txs := []*types.Transaction{
		newTestBlockTx(10, 1, 1),
		newTestBlockTx(20, 1, 2),
		newTestBlockTx(30, 1, 3),
	}
	receipts := []*types.Receipt{
		newTestReceipt(),
		newTestReceipt(),
		newTestReceipt(),
	}
	debts := []*types.Debt{
		newDebt(),
		newDebt(),
		newDebt(),
	}

	block := types.NewBlock(headers[0], txs, receipts, debts)

	return []*types.Block{block}
}

func newTestBlockTx(amount, fee, nonce uint64) *types.Transaction {
	fromAddr := common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01781")
	toAddr := common.HexMustToAddres("0xd0c549b022f5a17a8f50a4a448d20ba579d01780")

	tx, err := types.NewTransaction(fromAddr, toAddr, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(fee), nonce)
	if err != nil {
		panic(err)
	}

	return tx
}

func newTestReceipt() *types.Receipt {
	return &types.Receipt{
		Result:    []byte("result"),
		PostState: common.StringToHash("post state"),
		Logs:      nil,
		TxHash:    common.StringToHash("tx hash"),
	}
}

func newDebt() *types.Debt {
	return &types.Debt{
		Hash: common.EmptyHash,
		Data: *newDebtData(),
	}
}

func newDebtData() *types.DebtData {
	return &types.DebtData{
		TxHash:  common.EmptyHash,
		Shard:   2,
		Account: common.EmptyAddress,
		Amount:  new(big.Int).SetUint64(10),
	}
}
