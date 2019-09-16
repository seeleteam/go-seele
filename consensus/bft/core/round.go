package core

import (
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

type roundChangeSet struct {
	validatorSet bft.VerifierSet
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
			c.current = newRoundState(view, verifierSet, c.current.GetLockedHash(), c.current.Preprepare, c.current.pendingRequest, c.backend.HasBadProposal)
		} else {
			c.current = newRoundState(view, verifierSet, common.Hash{}, nil, c.current.pendingRequest, c.backend.HasBadProposal)
		}
	} else {
		c.current = newRoundState(view, verifierSet, common.Hash{}, nil, nil, c.backend.HasBadProposal)
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
