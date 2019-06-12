/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	rand2 "math/rand"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

var (
	statusNotDownloading int32
	statusDownloading    int32 = 1

	// ErrIsSynchronising indicates the synchronising  is in processing
	ErrIsSynchronising = errors.New("Is synchronising")
)

// Downloader sync block chain with remote peer
type Downloader struct {
	cancelCh   chan struct{} // Cancel current synchronising session
	msgCh      chan *p2p.Message
	syncStatus int32
	chain      BlockChain
	wg         sync.WaitGroup
	log        *log.SeeleLog
	lock       sync.RWMutex
}

// NewDownloader create Downloader
func newDownloader(chain BlockChain) *Downloader {
	d := &Downloader{
		chain:      chain,
		syncStatus: statusNotDownloading,
	}

	d.log = log.GetLogger("lightsync")
	return d
}

// Synchronise try to sync with remote peer.
func (d *Downloader) synchronise(p *peer) error {
	// Make sure only one routine can pass at once
	d.lock.Lock()
	if d.syncStatus == statusDownloading {
		d.lock.Unlock()
		return ErrIsSynchronising
	}

	d.cancelCh = make(chan struct{})
	d.msgCh = make(chan *p2p.Message)
	d.syncStatus = statusDownloading
	d.wg.Add(1)

	d.lock.Unlock()
	go d.doSynchronise(p)
	return nil
}

func (d *Downloader) doSynchronise(p *peer) {
	defer func() {
		d.cancel()

		d.wg.Done()
		d.lock.Lock()
		close(d.msgCh)
		d.syncStatus = statusNotDownloading
		d.lock.Unlock()
	}()

	ancestor, err := p.findAncestor()
	if err != nil {
		d.log.Info("doSynchronise called, but ancestor not found")
		return
	}

	err = d.reverseLightBCstore(ancestor)
	if err != nil {
		d.log.Error("failed to reverse the light chain to height %d", ancestor)
		return
	}

	reqID := rand2.Uint32()
	if err := p.sendDownloadHeadersRequest(reqID, ancestor); err != nil {
		d.log.Error("doSynchronise sendDownloadHeadersRequest err=%s", err)
		return
	}

needQuit:
	for {
		select {
		case msg := <-d.msgCh:
			if msg.Code != downloadHeadersResponseCode {
				break
			}

			var headMsg DownloadHeader
			if err := common.Deserialize(msg.Payload, &headMsg); err != nil {
				d.log.Debug("Downloader.doSynchronise Deserialize error. %s", err)
				break needQuit
			}

			if headMsg.ReqID != reqID {
				d.log.Debug("Downloader.doSynchronise received but reqID not match")
				break
			}

			if len(headMsg.Hearders) <= 1 {
				break needQuit
			}

			ancestorHead := headMsg.Hearders[0]
			if localBlock, err := d.chain.GetStore().GetBlockByHeight(ancestorHead.Height); err == nil {
				if ancestorHead.Hash() != localBlock.HeaderHash {
					d.log.Debug("Downloader.doSynchronise get ancestor ok, but not match, peer: %v, height: %d, localHash: %v, peerHash: %v", p.peerID.Hex(), ancestorHead.Height, localBlock.HeaderHash, ancestorHead.Hash())
					break needQuit
				}
			} else {
				d.log.Debug("Downloader.doSynchronise get ancestor from local error. %s, peer: %v", err, p.peerID.Hex())
				break needQuit
			}

			curHeight := uint64(0)
			counter := 0
			for _, head := range headMsg.Hearders[1:] {
				if err = d.chain.WriteHeader(head); err != nil && !errors.IsOrContains(err, core.ErrBlockAlreadyExists) {
					d.log.Warn("Downloader.doSynchronise WriteHeader error. %s", err)
					if errors.IsOrContains(err, consensus.ErrBlockNonceInvalid) {
						p.Peer.Disconnect("light PeerDownload anormaly")
					}
					break needQuit
				}
				d.log.Debug("Downloader.doSynchronise WriteHeader to chain, Height=%d, hash=%s, newHeader=%v, peer ID, %s, error: %s", head.Height, head.Hash(), err == nil, p.peerID.Hex(), err)
				curHeight = head.Height
				if errors.IsOrContains(err, core.ErrBlockAlreadyExists) {
					counter++
				}
			}

			if headMsg.HasFinished {
				d.log.Debug("Downloader.doSynchronise, has finished!")
				break needQuit
			}

			if counter >= len(headMsg.Hearders) - 1  && counter >= 250 {
				d.log.Debug("Downloader.doSynchronise, update ancestor! counter: %d, curHeight: %d， ancestor: %d", counter, curHeight, p.updatedAncestor)
				d.lock.Lock()
				if curHeight > p.updatedAncestor {
					p.updatedAncestor = curHeight
				}
				d.lock.Unlock()
			}

			reqID = rand2.Uint32()
			if err := p.sendDownloadHeadersRequest(reqID, curHeight); err != nil {
				d.log.Error("doSynchronise sendDownloadHeadersRequest err=%s", err)
				break needQuit
			}

		case <-d.cancelCh:
			d.log.Debug("Downloader.doSynchronise received cancelCh")
			break needQuit
		case <-p.quitCh:
			d.log.Debug("Downloader.doSynchronise received peer's quitCh")
			break needQuit
		}
	}

	d.log.Debug("Downloader.doSynchronise runs out")
	return
}

// DeliverMsg called by lightprotocol to deliver received msg from network
func (d *Downloader) deliverMsg(p *peer, msg *p2p.Message) {
	defer func() {
		if r := recover(); r != nil {
			d.log.Error("Downloader paniced. %s", r)
		}
	}()
	d.msgCh <- msg
	return
}

// cancel cancels current session.
func (d *Downloader) cancel() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
		default:
			close(d.cancelCh)
		}
	}
}

// Terminate close Downloader, cannot called anymore.
func (d *Downloader) Terminate() {
	d.cancel()
	d.wg.Wait()
}


// reverse the light chain back to the common ancestor of local light chain and peer chain
func (d *Downloader) reverseLightBCstore(ancestor uint64) error {

	bcStore := d.chain.GetStore()
	localCurHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		return nil
	}

	localCurHeader, err := bcStore.GetBlockHeader(localCurHash)
	if err != nil {
		return err
	}

	localHeight := localCurHeader.Height
	curHeight := localHeight
	localHashes := make([]common.Hash, 0)	

	for curHeight > ancestor {
		hash, err := bcStore.GetBlockHash(curHeight)
		d.log.Debug("light reverse curHeight: %d, hash: %v", curHeight, hash)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block hash by height %v", curHeight)
		}

		err = d.updateLightChainHeadInfo(curHeight - 1)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to update head: %v while reversing lightchain", hash)
		}

		// save the local blocks
		localHashes = append([]common.Hash{hash}, localHashes...)

		if err = bcStore.DeleteBlockHeader(hash); err != nil {
			return errors.NewStackedErrorf(err, "failed to delete block header %v", hash)
		}

		// delete the block hash in canonical chain.
		_, err = bcStore.DeleteBlockHash(curHeight)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to delete block hash by height %v", curHeight)
		}

		curHeight--
	}

	// rollback the txs in txpool
	d.chain.GetHeadRollbackEventManager().Fire(localHashes)

	return nil

}

func (d *Downloader) updateLightChainHeadInfo(height uint64) error {

	bcStore := d.chain.GetStore()
	curHash, err := bcStore.GetBlockHash(height)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block hash by height: %v", height)
	}

	curHeader, err := bcStore.GetBlockHeader(curHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block header by hash: %v", curHash)
	}
	curTd, err := bcStore.GetBlockTotalDifficulty(curHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block td by hash: %v", curHash)
	}
  
	d.log.Debug("update current block header: %d, hash: %v, td: %d", height, curHash, curTd)
	err = bcStore.PutHeadBlockHash(curHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to put head block: %v", curHash)
	}
	d.chain.PutTd(curTd)
	d.chain.PutCurrentHeader(curHeader)

	return nil
}
