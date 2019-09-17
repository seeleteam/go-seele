package core

import (
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

/*
Round change flow
•	There are three conditions that would trigger ROUND CHANGE:
		o	Round change timer expires.
		o	Invalid PREPREPARE message.
		o	Block insertion fails.
•	When a verifier notices that one of the above conditions applies, it broadcasts a ROUND CHANGE message along with the proposed round number and waits for ROUND CHANGE messages from other verifiers. The proposed round number is selected based on following condition:
		o	If the verifier has received ROUND CHANGE messages from its peers, it picks the largest round number which has F + 1 of ROUND CHANGE messages.
		o	Otherwise, it picks 1 + current round number as the proposed round number.
•	Whenever a verifier receives F + 1 of ROUND CHANGE messages on the same proposed round number, it compares the received one with its own. If the received is larger, the verifier broadcasts ROUND CHANGE message again with the received number.
•	Upon receiving 2F + 1 of ROUND CHANGE messages on the same proposed round number, the verifier exits the round change loop, calculates the new proposer, and then enters NEW ROUND state.
•	Another condition that a verifier jumps out of round change loop is when it receives verified block(s) through peer synchronization.
*/
type roundChangeSet struct {
	verifierSet  bft.VerifierSet
	roundChanges map[uint64]*messageSet
	mu           *sync.Mutex
}

func newRoundChangeSet(verSet bft.VerifierSet) *roundChangeSet {
	return &roundChangeSet{
		verifierSet:  verSet,
		roundChanges: make(map[uint64]*messageSet),
		mu:           new(sync.Mutex),
	}
}

// updateRoundState updates round state by checking if locking block is necessary
func (c *core) updateRoundState(view *bft.View, verifierSet bft.VerifierSet, roundChanged bool) {
	// Lock only if both roundChange is true and it is locked
	if roundChanged && c.current != nil {
		if c.current.IsHashLocked() {
			c.current = newRoundState(view, verifierSet, c.current.GetLockedHash(), c.current.Preprepare, c.current.pendingRequest, c.server.HasBadProposal)
		} else {
			c.current = newRoundState(view, verifierSet, common.Hash{}, nil, c.current.pendingRequest, c.server.HasBadProposal)
		}
	} else {
		c.current = newRoundState(view, verifierSet, common.Hash{}, nil, nil, c.server.HasBadProposal)
	}
}

// MaxRound returns the max round which the number of messages is equal or larger than num
func (rcs *roundChangeSet) MaxRound(num int) *big.Int {
	rcs.mu.Lock()
	defer rcs.mu.Unlock()

	var maxRound *big.Int
	for k, rms := range rcs.roundChanges {
		if rms.Size() < num {
			continue
		}
		r := big.NewInt(int64(k))
		if maxRound == nil || maxRound.Cmp(r) < 0 {
			maxRound = r
		}
	}
	return maxRound
}

// sendRoundChange sends the ROUND CHANGE message with the given round
func (c *core) sendRoundChange(round *big.Int) {
	cv := c.currentView()
	if cv.Round.Cmp(round) >= 0 {
		c.log.Error("Cannot send out the round change. current round %s. target round %s", cv.Round, round)
		return
	}

	c.catchUpRound(&bft.View{
		// The round number we'd like to transfer to.
		Round:    new(big.Int).Set(round),
		Sequence: new(big.Int).Set(cv.Sequence),
	})

	// Now we have the new round number and sequence number
	cv = c.currentView()
	rc := &bft.Subject{
		View:   cv,
		Digest: common.Hash{},
	}

	payload, err := Encode(rc)
	if err != nil {
		c.log.Error("Failed to encode ROUND CHANGE. rc %v. err %s", rc, err)
		return
	}

	c.broadcast(&message{
		Code: msgRoundChange,
		Msg:  payload,
	})
}

// sendNextRoundChange sends the ROUND CHANGE message with current round + 1
func (c *core) sendNextRoundChange() {
	cv := c.currentView()
	c.sendRoundChange(new(big.Int).Add(cv.Round, common.Big1))
}

func (c *core) handleRoundChange(msg *message, src bft.Verifier) error {
	// Docode->
	var rc *bft.Subject
	if err := msg.Decode(&rc); err != nil {
		return err
	}
	cv := c.currentView()
	roundView := rc.View

	num, err := c.roundChangeSet.Add(roundView.Round, msg)
	if err != nil {
		c.log.Warn("failed to add round change msg %v from %v with err %s", msg, src, err)
		return err
	}

	// Once we received f+1 ROUND CHANGE messages, those messages form a weak certificate.
	// If our round number is smaller than the certificate's round number, we would
	// try to catch up the round number.
	if c.waitingForRoundChange && num == int(c.verSet.F()+1) {
		if cv.Round.Cmp(roundView.Round) < 0 {
			c.sendRoundChange(roundView.Round)
		}
		return nil
	} else if num == int(2*c.verSet.F()+1) && (c.waitingForRoundChange || cv.Round.Cmp(roundView.Round) < 0) {
		// We've received 2f+1 ROUND CHANGE messages, start a new round immediately.
		c.startNewRound(roundView.Round)
		return nil
	} else if cv.Round.Cmp(roundView.Round) < 0 {
		// Only gossip the message with current round to other verifiers.
		return errIgnored
	}
	return nil
}

// Add adds the round and message into round change set
func (rcs *roundChangeSet) Add(r *big.Int, msg *message) (int, error) {
	rcs.mu.Lock()
	defer rcs.mu.Unlock()

	round := r.Uint64()
	if rcs.roundChanges[round] == nil {
		rcs.roundChanges[round] = newMessageSet(rcs.verifierSet)
	}
	err := rcs.roundChanges[round].Add(msg)
	if err != nil {
		return 0, err
	}
	return rcs.roundChanges[round].Size(), nil
}

// Clear deletes the messages with smaller round
func (rcs *roundChangeSet) Clear(round *big.Int) {
	rcs.mu.Lock()
	defer rcs.mu.Unlock()

	for k, rms := range rcs.roundChanges {
		if len(rms.Values()) == 0 || k < round.Uint64() {
			delete(rcs.roundChanges, k)
		}
	}
}
