package core

import (
	"github.com/seeleteam/go-seele/consensus/bft"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

/*backlog: previous commit and */

var (
	// msgPriority is defined for calculating processing priority to speedup consensus
	// msgPreprepare > msgCommit > msgPrepare
	// If we already have a proposal, we may have chance to speed up the consensus process
	// by committing the proposal without PREPARE messages.
	msgPriority = map[uint64]int{
		msgPreprepare: 1,
		msgCommit:     2,
		msgPrepare:    3,
	}
)

type backlogEvent struct {
	src bft.Verifier
	msg *message
}

func (c *core) checkMessage(msgCode uint64, view *bft.View) error {
	if view == nil || view.Sequence == nil || view.Round == nil {
		return errInvalidMsg
	}
	if view.Cmp(c.currentView()) > 0 {
		return errMsgFromFuture
	}

	if view.Cmp(c.currentView()) < 0 {
		return errOldMsg
	}
	if c.waitingForRoundChange {
		return errMsgFromFuture
	}

	// StateAcceptRequest only accepts msgPreprepare
	// other messages are future messages
	if c.state == StateAcceptRequest {
		if msgCode > msgPreprepare {
			return errMsgFromFuture
		}
		return nil
	}
	return nil
}

func (c *core) storeBacklog(msg *message, src bft.Verifier) {
	if src.Address() == c.Address() {
		c.log.Warn("backlog from self from %s, state %d", src, c.state)
		return
	}
	c.log.Debug("store future message")
	c.backlogsMu.Lock()
	defer c.backlogsMu.Unlock()
	c.log.Debug("retrieveing backlog queue for %s, backlog_size %d", src.Address(), len(c.backlogs))
	backlog := c.backlogs[src.Address()]
	if backlog == nil {
		backlog = prque.New()
	}
	switch msg.Code {
	case msgPreprepare: // preprepare message
		var p *bft.Preprepare
		err := msg.Decode(&p)
		if err == nil {
			backlog.Push(msg, toPriority(msg.Code, p.View))
		}
	default: // msgRoundChange, msgPrepare and msgCommit cases
		var p *bft.Subject
		err := msg.Decode(&p)
		if err == nil {
			backlog.Push(msg, toPriority(msg.Code, p.View))
		}
	}
	c.backlogs[src.Address()] = backlog
}

func (c *core) processBacklog() {
	c.backlogsMu.Lock()
	defer c.backlogsMu.Unlock()

	for srcAddress, backlog := range c.backlogs {
		if backlog == nil {
			continue
		}
		_, src := c.verSet.GetByAddress(srcAddress)
		if src == nil {
			// validator is not available
			delete(c.backlogs, srcAddress)
			continue
		}
		isFuture := false

		// We stop processing if
		//   1. backlog is empty
		//   2. The first message in queue is a future message
		for !(backlog.Empty() || isFuture) {
			m, prio := backlog.Pop()
			msg := m.(*message)
			var view *bft.View
			switch msg.Code {
			case msgPreprepare:
				var m *bft.Preprepare
				err := msg.Decode(&m)
				if err == nil {
					view = m.View
				}
				// for msgRoundChange, msgPrepare and msgCommit cases
			default:
				var sub *bft.Subject
				err := msg.Decode(&sub)
				if err == nil {
					view = sub.View
				}
			}
			if view == nil {
				c.log.Debug("Nil view. msg %v", msg)
				continue
			}
			// Push back if it's a future message
			err := c.checkMessage(msg.Code, view)
			if err != nil {
				if err == errMsgFromFuture {
					c.log.Debug("Stop processing backlog. msg %v", msg)
					backlog.Push(msg, prio)
					isFuture = true
					break
				}

				c.log.Debug("Skip the backlog event. msg %v. err %s", msg, err)
				continue
			}
			c.log.Debug("Post backlog event. msg %v", msg)

			go c.sendEvent(backlogEvent{
				src: src,
				msg: msg,
			})
		}
	}
}

func toPriority(msgCode uint64, view *bft.View) float32 {
	if msgCode == msgRoundChange {
		// For msgRoundChange, set the message priority based on its sequence
		return -float32(view.Sequence.Uint64() * 1000)
	}
	// FIXME: round will be reset as 0 while new sequence
	// 10 * Round limits the range of message code is from 0 to 9
	// 1000 * Sequence limits the range of round is from 0 to 99
	return -float32(view.Sequence.Uint64()*1000 + view.Round.Uint64()*10 + uint64(msgPriority[msgCode]))
}
