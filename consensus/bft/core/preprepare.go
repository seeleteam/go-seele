package core

import (
	"fmt"
	"time"

	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
)

/*
preprepare.go (part of core package) mainly implement functions on the preprepare step;
send /
*/

// sendPreprepare
func (c *core) sendPreprepare(request *bft.Request) {
	// sequence is the proposal height and this node is the proposer
	// initiate the preprepare message and encode it
	if c.current.Sequence().Uint64() == request.Proposal.Height() && c.isProposer() {
		curView := c.currentView()
		preprepare, err := Encode(&bft.Preprepare{
			View:     curView,
			Proposal: request.Proposal,
		})
		if err != nil {
			c.log.Error("fail to encode preprepare state %d view %v", c.state, curView)
			return
		}
		// broadcast the message
		c.broadcast(&message{
			Code: msgPreprepare,
			Msg:  preprepare,
		})
		fmt.Println("sendPreprepare->broadcast->Post")

	}
}

//
// Decode -> checkMessage(make usre it is new) -> ensure it is from proposer -> verify proposal received -> accept preprepare
func (c *core) handlePreprepare(msg *message, src bft.Verifier) error {
	// 1. Decode preprepare message first
	var preprepare *bft.Preprepare
	err := msg.Decode(&preprepare)
	if err != nil {
		return errDecodePreprepare
	}

	// we need to check the message: ensure we have the same view with the preprepare message
	// if not (namely, it is old message), see if we need to broadcast Commit.
	if err := c.checkMessage(msgPreprepare, preprepare.View); err != nil {
		if err == errOldMsg {
			// get all verifiers for this proposal
			verSet := c.server.ParentVerifiers(preprepare.Proposal).Copy()
			previousProposer := c.server.GetProposer(preprepare.Proposal.Height() - 1)
			verSet.CalcProposer(previousProposer, preprepare.View.Round.Uint64())
			// proposer matches (sequence + round) && given block exists
			// then broadcast commit
			if verSet.IsProposer(src.Address()) && c.server.HasPropsal(preprepare.Proposal.Hash()) {
				c.sendOldCommit(preprepare.View, preprepare.Proposal.Hash())
				return nil
			}
		}
		return err
	}

	// only proposer will broadcast preprepare message
	if !c.verSet.IsProposer(src.Address()) {
		c.log.Warn("igonore preprepare message since it is not the proposer")
		return errNotProposer
	}

	// verify the proposal we received
	if duration, err := c.server.Verify(preprepare.Proposal); err != nil {
		c.log.Warn("failed to verify proposal with err %s duration %d", err, duration)
		// it is a future block, and re-handle it after duration
		if err == consensus.ErrBlockCreateTimeOld {
			c.stopFuturePreprepareTimer() // stop timer
			c.futurePreprepareTimer = time.AfterFunc(duration, func() {
				c.sendEvent(backlogEvent{
					src: src,
					msg: msg,
				})
			})
		} else {
			c.sendNextRoundChange()
		}
		return err
	}

	// accept the preprepare message
	if c.state == StateAcceptRequest {
		if c.current.IsHashLocked() { // there is a locked proposal
			if preprepare.Proposal.Hash() == c.current.GetLockedHash() { // at the same proposal
				c.acceptPreprepare(preprepare)
				c.setState(StatePreprepared)
				c.sendCommit()
			} else { // at different proposals. change round
				c.sendNextRoundChange()
			}
		} else { // there is no locked proposal
			c.acceptPreprepare(preprepare)
			c.setState(StatePreprepared)
			c.sendCommit()
		}
	}

	return nil
}

func (c *core) acceptPreprepare(preprepare *bft.Preprepare) {
	c.consensusTimestamp = time.Now()
	c.current.SetPreprepare(preprepare)
}
