package core

import (
	"github.com/seeleteam/go-seele/consensus/bft"
)

/*
request.go (part of core package)

*/

func (c *core) handleRequest() {}
func (c *core) processPendingRequests() {
	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	// TODO since future request will push back and this is infinity loop
	// we may need a indicator to help to enter this loop in order to loop idly
	for !(c.pendingRequests.Empty()) {
		msg, priority := c.pendingRequests.Pop()
		req, ok := msg.(*bft.Request)
		if !ok {
			c.logger.Warn("Malformed request, skip. msg %v", m)
			continue
		}

		err := c.checkRequestMsg(r)
		if err != nil {
			// this is a future message, need to push back
			if err == errFutureMessage {
				c.log.Info("future request with height %d hash %s", req.Proposal.Height(), req.Proposal.Hash())
				c.pendingRequests.Psh(msg, priority)
				break
			}
			c.log.Info("check request with error %s, height %d hash %s", err, req.Proposal.Height(), req.Proposal.Hash())
			continue
		}
		c.log.Info("send pending request at height %d hash %s", req.Proposal.Height(), req.Proposal.Hash())
		go c.sendEvent(bft.RequestEvent{
			Proposal: req.Proposal,
		})
	}

}
func (c *core) checkRequestMsg() {}
func (c *core) storeRequestMsg() {}
