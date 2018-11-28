/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package core

import (
	"reflect"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/istanbul"
)

func (c *core) sendCommit() {
	sub := c.current.Subject()
	c.broadcastCommit(sub)
}

func (c *core) sendCommitForOldBlock(view *istanbul.View, digest common.Hash) {
	sub := &istanbul.Subject{
		View:   view,
		Digest: digest,
	}
	c.broadcastCommit(sub)
}

func (c *core) broadcastCommit(sub *istanbul.Subject) {
	encodedSubject, err := Encode(sub)
	if err != nil {
		c.logger.Error("Failed to encode. subject %v。 state %d", sub, c.state)
		return
	}
	c.broadcast(&message{
		Code: msgCommit,
		Msg:  encodedSubject,
	})
}

func (c *core) handleCommit(msg *message, src istanbul.Validator) error {
	// Decode COMMIT message
	var commit *istanbul.Subject
	err := msg.Decode(&commit)
	if err != nil {
		return errFailedDecodeCommit
	}

	if err := c.checkMessage(msgCommit, commit.View); err != nil {
		return err
	}

	if err := c.verifyCommit(commit, src); err != nil {
		return err
	}

	c.acceptCommit(msg, src)

	// Commit the proposal once we have enough COMMIT messages and we are not in the Committed state.
	//
	// If we already have a proposal, we may have chance to speed up the consensus process
	// by committing the proposal without PREPARE messages.
	if c.current.Commits.Size() > 2*c.valSet.F() && c.state.Cmp(StateCommitted) < 0 {
		// Still need to call LockHash here since state can skip Prepared state and jump directly to the Committed state.
		c.current.LockHash()
		c.commit()
	}

	return nil
}

// verifyCommit verifies if the received COMMIT message is equivalent to our subject
func (c *core) verifyCommit(commit *istanbul.Subject, src istanbul.Validator) error {
	sub := c.current.Subject()
	if !reflect.DeepEqual(commit, sub) {
		c.logger.Warn("Inconsistent subjects between commit and proposal. expected %v. got %v.", sub, commit)
		return errInconsistentSubject
	}

	return nil
}

func (c *core) acceptCommit(msg *message, src istanbul.Validator) error {
	// Add the COMMIT message to current round state
	if err := c.current.Commits.Add(msg); err != nil {
		c.logger.Error("Failed to record commit message. msg %v. err %s", msg, err)
		return err
	}

	return nil
}
