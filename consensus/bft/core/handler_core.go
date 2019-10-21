package core

import (
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

func (c *core) Start() error {
	c.startNewRound(common.Big0)
	c.subscribeEvents()
	go c.handleEvents()
	return nil
}

func (c *core) Stop() error {
	c.stopTimer()
	c.unsubscribeEvents()
	c.handlerWg.Wait()
	return nil
}

func (c *core) subscribeEvents() {
	c.events = c.server.EventMux().Subscribe(
		bft.RequestEvent{},
		bft.MessageEvent{},
		backlogEvent{},
	)
	c.timeoutSub = c.server.EventMux().Subscribe(
		timeoutEvent{},
	)
	c.finalCommittedSub = c.server.EventMux().Subscribe(
		bft.FinalCommittedEvent{}, //TODO
	)
}
func (c *core) unsubscribeEvents() {
	c.events.Unsubscribe()
	c.timeoutSub.Unsubscribe()
	c.finalCommittedSub.Unsubscribe()
}
func (c *core) handleEvents() {
	// reset
	defer func() {
		c.current = nil
		c.handlerWg.Done()
	}()

	c.handlerWg.Add(1)
	for {
		select { // wait for channel and then execute
		case event, ok := <-c.events.Chan():
			if !ok {
				return
			}
			switch e := event.Data.(type) {
			case bft.RequestEvent: // proposal handle
				req := &bft.Request{
					Proposal: e.Proposal,
				}
				err := c.handleRequest(req)
				if err == errMsgFromFuture {
					c.storeRequestMsg(req)
				}
			case bft.MessageEvent: // prepare, commit all other msgs
				if err := c.handleMsg(e.Payload); err == nil {
					c.server.Gossip(c.verSet, e.Payload)
				}
			case backlogEvent: // internal event
				if err := c.handleCheckedMsg(e.msg, e.src); err == nil {
					p, err := e.msg.Payload()
					if err != nil {
						c.log.Warn("failed to get message payload with err %v", err)
						continue
					}
					c.server.Gossip(c.verSet, p)
				}
			}
		case _, ok := <-c.timeoutSub.Chan():
			if !ok {
				return
			}
			c.handleTimeoutMsg()
		case e, ok := <-c.finalCommittedSub.Chan():
			if !ok {
				return
			}
			switch e.Data.(type) {
			case bft.FinalCommittedEvent:
				c.handleFinalCommitted()
			}
		}
	}
}
func (c *core) sendEvent(event interface{}) {
	c.server.EventMux().Post(event)
	fmt.Println("Post in sendEvent")
}
func (c *core) handleMsg(payload []byte) error {
	msg := new(message)
	if err := msg.ValidatePayload(payload, c.verifyFn); err != nil {
		c.log.Error("failed to validate msg payload with err %s", err)
		return err
	}
	_, src := c.verSet.GetByAddress(msg.Address)
	if src == nil {
		c.log.Error("invalid address in messageg %v", msg)
		return ErrAddressUnauthorized
	}
	return c.handleCheckedMsg(msg, src)
}

func (c *core) handleCheckedMsg(msg *message, src bft.Verifier) error {
	// record the message if it is a future message
	backlog := func(err error) error {
		if err == errMsgFromFuture {
			c.storeBacklog(msg, src)
		}
		return err
	}

	switch msg.Code {
	case msgPreprepare:
		return backlog(c.handlePreprepare(msg, src)) //TODO
	case msgPrepare:
		return backlog(c.handlePrepare(msg, src)) //TODO
	case msgCommit:
		return backlog(c.handleCommit(msg, src)) //TODO
	case msgRoundChange:
		return backlog(c.handleRoundChange(msg, src)) //TODO
	default:
		c.log.Error("invalid message: msg %v address %s from %v", msg, c.address, src)
	}
	return errInvalidMsg
}

func (c *core) handleTimeoutMsg() {
	if !c.waitingForRoundChange {
		maxRound := c.roundChangeSet.MaxRound(c.verSet.F() + 1)
		if maxRound != nil && maxRound.Cmp(c.current.Round()) > 0 {
			c.sendRoundChange(maxRound)
			return
		}
	}
	lastProposal, _ := c.server.LastProposal()
	if lastProposal != nil && lastProposal.Height() >= c.current.Sequence().Uint64() {
		c.log.Info("round change timeout, catch up lastest sequence at height %d", lastProposal.Height())
		c.startNewRound(common.Big0)
	} else {
		c.sendNextRoundChange()
	}
}

func (c *core) handleFinalCommitted() error {
	c.log.Debug("Received a final committed proposal")
	c.startNewRound(common.Big0)
	return nil
}
