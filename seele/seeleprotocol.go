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
handler:
	for {
		msg, err := peer.rw.ReadMsg()
		if err != nil {
			p.log.Error("get error when read msg from %s, %s", peer.peerID.ToHex(), err)
			break
		}

		switch msg.Code {
		case transactionHashMsgCode:
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("deserialize transaction hash msg failed %s", err.Error())
				continue
			}

			if !peer.knownTxs.Has(txHash) {
				peer.knownTxs.Add(txHash) //update peer known transaction
				err := peer.SendTransactionRequest(txHash)
				if err != nil {
					p.log.Warn("send transaction request msg failed %s", err.Error())
					break handler
				}
			}

		case transactionRequestMsgCode:
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("deserialize transaction request msg failed %s", err.Error())
				continue
			}

			tx := p.txPool.GetTransaction(txHash)
			err = peer.SendTransaction(tx)
			if err != nil {
				p.log.Warn("send transaction msg failed %s", err.Error())
				break handler
			}

		case transactionMsgCode:
			var tx types.Transaction
			err := common.Deserialize(msg.Payload, &tx)
			if err != nil {
				p.log.Warn("deserialize transaction msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got transaction msg %s", tx.Hash.ToHex())
			p.txPool.AddTransaction(&tx)

		case blockHashMsgCode:
			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("deserialize block hash msg failed %s", err.Error())
				continue
			}

			if !peer.knownBlocks.Has(blockHash) {
				peer.knownBlocks.Add(blockHash)
				err := peer.SendBlockRequest(blockHash)
				if err != nil {
					p.log.Warn("send block request msg failed %s", err.Error())
					break handler
				}
			}

		case blockRequestMsgCode:
			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("deserialize block request msg failed %s", err.Error())
				continue
			}

			block, err := p.chain.GetBlockChainStore().GetBlock(blockHash)
			if err != nil {
				p.log.Warn("not found request block %s", err.Error())
				continue
			}

			err = peer.SendBlock(block)
			if err != nil {
				p.log.Warn("send block msg failed %s", err.Error())
			}

		case blockMsgCode:
			var block types.Block
			err := common.Deserialize(msg.Payload, &block)
			if err != nil {
				p.log.Warn("deserialize block msg failed %s", err.Error())
				continue
			}

			// @todo need to make sure WriteBlock handle block fork
			p.chain.WriteBlock(&block)

		default:
			p.log.Warn("unknown code %s", msg.Code)
		}
	}

	p.peerSet.Remove(peer.peerID)
}

// Stop stops protocol, called when seeleService quits.
func (p SeeleProtocol) Stop() {
	//TODO add a quit channel
}
