/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

var (
	statusNotDownloading int32 = 0
	statusDownloading    int32 = 1

	ErrIsSynchronising = errors.New("Is synchronising")
)

// Downloader sync block chain with remote peer
type Downloader struct {
	cancelCh chan struct{} // Cancel current synchronising session

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
	d.wg.Add(1)
	d.lock.Unlock()
	go d.doSynchronise(p)
	return nil
}

func (d *Downloader) doSynchronise(p *peer) {
	defer d.wg.Done()
	defer close(d.cancelCh)
	//todo find common ancestor , and send block headers request message
	return
}

// DeliverMsg called by seeleprotocol to deliver received msg from network
func (d *Downloader) deliverMsg(p *peer, msg *p2p.Message) {
	//todo
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
