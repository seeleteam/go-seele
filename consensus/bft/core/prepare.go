package core

import (
	"reflect"

	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/consensus/istanbul"
)

// sendPrepare : encode -> broadcast
func (c *core) sendPrepare() {
	subject := c.current.Subject()
	encodedSubject, err := Encode(subject)
	if err != nil {
		c.log.Error("failed to encode subject %v, state %v", subject, c.state)
		return
	}
	c.broadcast(&message{
		Code, msgPrepare,
		Msg, encodedSubject,
	})
}

// handlePrepare: Decode->checkMessage->verify->accept->change state & send commit
func (c *core) handlePrepare(msg *message, src bft.Verifier) error {
	// Decode PREPARE message
	var prepare *bft.Subject
	if err := msg.Decode(&prepare); err != nil {
		return errFailedDecodePrepare
	}
	if err := c.checkMessage(msgPrepare, prepare.View); err != nil {
		return nil
	}
	if err := c.verifyPrepare(prepare, src); err != nil {
		return nil
	}
	c.acceptPrepare(msg, src)

	if ((c.current.IsHashLocked() && prepare.Digest == c.current.GetLockedHash()) || c.current.GetPrepareOrCommitSize() > 2*c.valSet.F()) &&
		c.state.Cmp(StatePrepared) < 0 {
		c.current.LockHash()
		c.setState(StatePrepared)
		c.sendCommit()
	}
	return nil
}

// verifyPrepare verifies if the received PREPARE message is equivalent to our subject
func (c *core) verifyPrepare(prepare *istanbul.Subject, src istanbul.Validator) error {
	sub := c.current.Subject()
	if !reflect.DeepEqual(prepare, sub) {
		c.logger.Warn("Inconsistent subjects between PREPARE and proposal. from %s. state %d. expected %v. got %v",
			src, c.state, sub, prepare)
		return errInconsistentSubject
	}

	return nil
}
func (c *core) acceptPrepare(msg *message, src istanbul.Validator) error {
	// Add the PREPARE message to current round state
	if err := c.current.Prepares.Add(msg); err != nil {
		c.logger.Error("Failed to add PREPARE message to round state. from %s. state %d. msg %v. err %s",
			src, c.state, msg, err)
		return err
	}

	return nil
}
