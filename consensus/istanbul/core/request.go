/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package core

import "github.com/seeleteam/go-seele/consensus/istanbul"

func (c *core) handleRequest(request *istanbul.Request) error {
	if err := c.checkRequestMsg(request); err != nil {
		if err == errInvalidMessage {
			c.logger.Warn("invalid request")
			return err
		}
		c.logger.Warn("unexpected request. err %s. height %d. hash %s", err, request.Proposal.Height(), request.Proposal.Hash())
		return err
	}

	c.logger.Debug("handleRequest. height %d. hash %s", request.Proposal.Height(), request.Proposal.Hash())

	c.current.pendingRequest = request
	if c.state == StateAcceptRequest {
		c.sendPreprepare(request)
	}
	return nil
}

// check request state
// return errInvalidMessage if the message is invalid
// return errFutureMessage if the sequence of proposal is larger than current sequence
// return errOldMessage if the sequence of proposal is smaller than current sequence
func (c *core) checkRequestMsg(request *istanbul.Request) error {
	if request == nil || request.Proposal == nil {
		return errInvalidMessage
	}

	if c := c.current.sequence.Uint64() - request.Proposal.Height(); c > 0 {
		return errOldMessage
	} else if c < 0 {
		return errFutureMessage
	} else {
		return nil
	}
}

func (c *core) storeRequestMsg(request *istanbul.Request) {
	c.logger.Debug("Store future request. height %d. hash %s. state %d", request.Proposal.Height(), request.Proposal.Hash(), c.state)

	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	c.pendingRequests.Push(request, float32(-request.Proposal.Height()))
}

func (c *core) processPendingRequests() {
	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	for !(c.pendingRequests.Empty()) {
		m, prio := c.pendingRequests.Pop()
		r, ok := m.(*istanbul.Request)
		if !ok {
			c.logger.Warn("Malformed request, skip. msg %v", m)
			continue
		}
		// Push back if it's a future message
		err := c.checkRequestMsg(r)
		if err != nil {
			if err == errFutureMessage {
				c.logger.Debug("Stop processing request height %d. hash %s", r.Proposal.Height(), r.Proposal.Hash())
				c.pendingRequests.Push(m, prio)
				break
			}
			c.logger.Debug("Skip the pending request err %s. height %d. hash %s", err, r.Proposal.Height(), r.Proposal.Hash())
			continue
		}
		c.logger.Debug("Post pending request height %d. hash %s", r.Proposal.Height(), r.Proposal.Hash())

		go c.sendEvent(istanbul.RequestEvent{
			Proposal: r.Proposal,
		})
	}
}
