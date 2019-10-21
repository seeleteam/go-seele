package core

import (
	"fmt"
	"reflect"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

/*
commit.go mainly implement the functions each node call with commmit: send/handle commit
*/

/*
type core struct {
	config  *istanbul.Config
	address common.Address
	state   State
	log  *log.SeeleLog

	backend               istanbul.Backend
	events                *event.TypeMuxSubscription
	finalCommittedSub     *event.TypeMuxSubscription
	timeoutSub            *event.TypeMuxSubscription
	futurePreprepareTimer *time.Timer

	valSet                istanbul.ValidatorSet
	waitingForRoundChange bool
	validateFn            func([]byte, []byte) (common.Address, error)

	backlogs   map[common.Address]*prque.Prque
	backlogsMu *sync.Mutex

	current   *roundState
	handlerWg *sync.WaitGroup

	roundChangeSet   *roundChangeSet
	roundChangeTimer *time.Timer

	pendingRequests   *prque.Prque
	pendingRequestsMu *sync.Mutex

	consensusTimestamp time.Time
	// the meter to record the round change rate
	roundMeter metrics.Meter
	// the meter to record the sequence update rate
	sequenceMeter metrics.Meter
	// the timer to record consensus duration (from accepting a preprepare to final committed stage)
	consensusTimer metrics.Timer
}

*/

// sendCommit send commits
func (c *core) sendCommit() {
	// get the subject
	subject := c.current.Subject()
	// broadcast subject
	c.broadcastCommit(subject)
}

// sendOldCommit send commit for old block
func (c *core) sendOldCommit(view *bft.View, digest common.Hash) {
	subject := &bft.Subject{
		View:   view,
		Digest: digest,
	}
	c.broadcastCommit(subject)
}

func (c *core) handleCommit(msg *message, src bft.Verifier) error {
	// Decode->checkMessage->verifyCommit->acceptCommit->check state and commit
	var commit *bft.Subject
	err := msg.Decode(&commit)
	if err != nil {
		return errDecodeCommit
	}
	if err := c.checkMessage(msgCommit, commit.View); err != nil {
		return nil
	}
	if err := c.verifyCommit(commit, src); err != nil {
		return err
	}
	c.acceptCommit(msg, src)

	// if we already have enough commit and meanwhile not in committed state-> commit!
	if c.current.Commits.Size() > 2*c.verSet.F() && c.state.Cmp(StateCommitted) < 0 {
		// Still need to call LockHash here since state can skip Prepared state and jump directly to the Committed state.
		c.current.LockHash()
		c.commit()
	}

	return nil
}

// verifyCommit verifies if the received COMMIT message is equivalent to our subject
func (c *core) verifyCommit(commit *bft.Subject, src bft.Verifier) error {
	sub := c.current.Subject()
	if !reflect.DeepEqual(commit, sub) {
		c.log.Warn("Inconsistent subjects between commit and proposal. expected %v. got %v.", sub, commit)
		return errInconsistentSubjects
	}

	return nil
}

// broadcastCommit broadcast commit out
func (c *core) broadcastCommit(sub *bft.Subject) {
	encodedSubject, err := Encode(sub)
	if err != nil {
		c.log.Error("Failed to encode. subject %v。 state %d", sub, c.state)
		return
	}
	c.broadcast(&message{
		Code: msgCommit,
		Msg:  encodedSubject,
	})
	fmt.Println("broadcastCommit->broadcast->Post")
}

func (c *core) acceptCommit(msg *message, src bft.Verifier) error {
	if err := c.current.Commits.Add(msg); err != nil {
		c.log.Error("failed to accept commit message: %v with error: %s", msg, err)
		return err
	}
	return nil
}
