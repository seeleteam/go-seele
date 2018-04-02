/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

// SeeleProtocol service implementation of seele
type SeeleProtocol struct {
	p2p.Protocol
	peerSet *peerSet

	networkID uint64
	txPool    *core.TransactionPool //same instance with seeleService tx pool
	chain     *core.Blockchain      //same instance with seeleService chain
	log       *log.SeeleLog
}

// NewSeeleProtocol create SeeleProtocol
func NewSeeleProtocol(seele *SeeleService, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:       SeeleProtoName,
			Version:    SeeleVersion,
			Length:     1,
			AddPeer:    s.handleAddPeer,
			DeletePeer: s.handleDelPeer,
		},
		networkID: seele.networkID,
		txPool:    seele.TxPool(),
		chain:     seele.BlockChain(),
		log:       log,
		peerSet:   newPeerSet(),
	}

	event.TransactionInsertedEventManager.AddAsyncListener(s.findNewTx)
	return s, nil
}

func (p *SeeleProtocol) findNewTx(e event.Event) {
	p.log.Debug("find new tx")
	tx := e.(*types.Transaction)

	p.peerSet.ForEach(func(peer *peer) bool {
		p.log.Debug("handle node %s", peer.Node.String())

		p.log.Debug("send tx")
		peer.SendTransactionHash(tx)

		return true
	})
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	newPeer := newPeer(SeeleVersion, p2pPeer, rw)
	if err := newPeer.HandShake(); err != nil {
		newPeer.Disconnect(DiscHandShakeErr)
		p.log.Error("handleAddPeer err. %s", err)
		return
	}

	p.peerSet.Add(newPeer)
	go p.handleMsg(newPeer)
}

func (p *SeeleProtocol) handleDelPeer(p2pPeer *p2p.Peer) {
	p.peerSet.Remove(p2pPeer.Node.ID)
}

func (p *SeeleProtocol) handleMsg(peer *peer) {
	for {
		msg, err := peer.rw.ReadMsg()
		if err != nil {
			p.log.Error("get error when read msg from %s, %s", peer.peerID.ToHex(), err)
			break
		}

		if msg.Code == transactionMsgCode {
			var tx types.Transaction
			common.Deserialize(msg.Payload, &tx)
			p.log.Debug("got transaction msg %s", tx.Hash.ToHex())

		} else if msg.Code == blockMsgCode {
			var block types.Block
			common.Deserialize(msg.Payload, &block)

		} else if msg.Code == transactionHashMsgCode {
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("msg deserialize err %s", err.Error())
				continue
			}

			if !peer.knownTxs.Has(txHash) {
				err := peer.SendTransactionRequest(txHash)
				if err != nil {
					p.log.Error("send transaction request error:%s", err.Error())
					break
				}
			}

		} else if msg.Code == blockHashMsgCode {

		} else {
			p.log.Warn("unknown code %s", msg.Code)
		}
	}

	p.peerSet.Remove(peer.peerID)
}

// Stop stops protocol, called when seeleService quits.
func (p SeeleProtocol) Stop() {
	//TODO add a quit channel
}
