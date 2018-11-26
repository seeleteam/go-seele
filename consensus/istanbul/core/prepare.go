/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package core

import (
	"reflect"

	"github.com/seeleteam/go-seele/consensus/istanbul"
)

func (c *core) sendPrepare() {
	sub := c.current.Subject()
	encodedSubject, err := Encode(sub)
	if err != nil {
		c.logger.Error("Failed to encode. subject %v. state %v", sub, c.state)
		return
	}
	c.broadcast(&message{
		Code: msgPrepare,
		Msg:  encodedSubject,
	})
}

func (c *core) handlePrepare(msg *message, src istanbul.Validator) error {
	// Decode PREPARE message
	var prepare *istanbul.Subject
	err := msg.Decode(&prepare)
	if err != nil {
		return errFailedDecodePrepare
	}

	if err := c.checkMessage(msgPrepare, prepare.View); err != nil {
		return err
	}

	// If it is locked, it can only process on the locked block.
	// Passing verifyPrepare and checkMessage implies it is processing on the locked block since it was verified in the Preprepared state.
	if err := c.verifyPrepare(prepare, src); err != nil {
		return err
	}

	c.acceptPrepare(msg, src)

	// Change to Prepared state if we've received enough PREPARE messages or it is locked
	// and we are in earlier state before Prepared state.
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
