/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

var (
	transactionHashMsgCode    uint16 = 0
	transactionRequestMsgCode uint16 = 1
	transactionsMsgCode       uint16 = 2
	blockHashMsgCode          uint16 = 3
	blockRequestMsgCode       uint16 = 4
	blockMsgCode              uint16 = 5

	statusDataMsgCode      uint16 = 6
	statusChainHeadMsgCode uint16 = 7

	protocolMsgCodeLength uint16 = 13
)

type BlockChain interface {
}

type TransactionPool interface {
	//AddRemotes(txs []*types.Transaction) []error
	//Status(hashes []common.Hash) []core.TxStatus
}

// SeeleProtocol service implementation of seele
type LightProtocol struct {
	p2p.Protocol

	bServerMode bool
	networkID   uint64
	txPool      TransactionPool
	chain       BlockChain

	wg                       sync.WaitGroup
	quitCh                   chan struct{}
	syncCh                   chan struct{}
	chainHeaderChangeChannel chan common.Hash
	log                      *log.SeeleLog
}

// NewLightProtocol create LightProtocol
func NewLightProtocol(networkID uint64, txPool TransactionPool, chain BlockChain, serverMode bool, log *log.SeeleLog) (s *LightProtocol, err error) {
	s = &LightProtocol{
		Protocol: p2p.Protocol{
			Name:    LightProtoName,
			Version: LightSeeleVersion,
			Length:  protocolMsgCodeLength,
		},
		bServerMode: serverMode,
		networkID:   networkID,
		txPool:      txPool,
		chain:       chain,
		log:         log,
		quitCh:      make(chan struct{}),
		syncCh:      make(chan struct{}),
	}

	s.Protocol.AddPeer = s.handleAddPeer
	s.Protocol.DeletePeer = s.handleDelPeer
	s.Protocol.GetPeer = s.handleGetPeer

	return s, nil
}

func (sp *LightProtocol) Start() {
	sp.log.Debug("SeeleProtocol.Start called!")

}

// Stop stops protocol, called when seeleService quits.
func (sp *LightProtocol) Stop() {
	close(sp.quitCh)
	close(sp.syncCh)
	sp.wg.Wait()
}

func (p *LightProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {

}

func (s *LightProtocol) handleGetPeer(address common.Address) interface{} {
	return nil
}

func (s *LightProtocol) handleDelPeer(peer *p2p.Peer) {
	s.log.Debug("delete peer from peer set. %s", peer.Node)
}

func (p *LightProtocol) handleMsg(peer *peer) {

	p.handleDelPeer(peer.Peer)
	p.log.Debug("seele.peer.run out!peer=%s!", peer.peerStrID)
}
