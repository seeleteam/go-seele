package core

import (
	"github.com/seeleteam/go-seele/consensus/bft"
)

/*
request.go (part of core package)

*/

func (c *core) handleRequest(request *bft.Request) error {
	if err := c.checkRequestMsg(request); err != nil {
		if err == errInvalidMsg {
			c.log.Warn("invalid request")
			return err
		}
		c.log.Warn("unexpected request, err %s, height %d, ")
		return err
	}
	c.log.Debug("handleRequest height %d, hash %s", request.Proposal.Height(), request.Proposal.Hash())
	c.current.pendingRequest = request
	if c.state == StateAcceptRequest { // state is ready
		c.sendPreprepare(request)
	}
	return nil
}

func (c *core) processPendingRequests() {
	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	// TODO since future request will push back and this is an infinity loop
	// we may need an indicator to help to enter this loop in order not to loop idly
	for !(c.pendingRequests.Empty()) {
		msg, priority := c.pendingRequests.Pop()
		req, ok := msg.(*bft.Request)
		if !ok {
			c.log.Warn("Malformed request, skip. msg %v", msg)
			continue
		}

		err := c.checkRequestMsg(req)
		if err != nil {
			// this is a future message, need to push back
			if err == errMsgFromFuture {
				c.log.Info("future request with height %d hash %s", req.Proposal.Height(), req.Proposal.Hash())
				c.pendingRequests.Push(msg, priority)
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

// checkRequestMsg check request: invalid / future / old
func (c *core) checkRequestMsg(request *bft.Request) error {
	if request == nil || request.Proposal == nil {
		return errInvalidMsg
	}
	if c.current.sequence.Uint64() > request.Proposal.Height() {
		return errOldMsg
	} else if c.current.sequence.Uint64() < request.Proposal.Height() {
		return errMsgFromFuture
	} else {
		return nil
	}
}

func (c *core) storeRequestMsg(request *bft.Request) {
	c.log.Debug("Store future request. height %d. hash %s. state %d", request.Proposal.Height(), request.Proposal.Hash(), c.state)

	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	c.pendingRequests.Push(request, float32(-request.Proposal.Height()))
}
