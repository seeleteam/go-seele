package core

import (
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/log"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

type core struct {
	config  *bft.BFTConfig
	address common.Address
	state   State
	log     *log.SeeleLog

	server                bft.Server
	events                *event.TypeMuxSubscription
	finalCommittedSub     *event.TypeMuxSubscription
	timeoutSub            *event.TypeMuxSubscription
	futurePreprepareTimer *time.Timer

	verSet                bft.VerifierSet
	waitingForRoundChange bool
	verifyFn              func([]byte, []byte) (common.Address, error)

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

func (c *core) broadcast(msg *message) {
	payload, err := c.finalizeMessage(msg)
	if err != nil {
		c.logger.Error("Failed to finalize message. msg %v. err %s. state %d", msg, err, c.state)
		return
	}

	// Broadcast payload
	if err = c.backend.Broadcast(c.valSet, payload); err != nil {
		c.logger.Error("Failed to broadcast message. msg %v. err %s. state %d", msg, err, c.state)
		return
	}
}

func (c *core) currentView() *bft.View {
	return &bft.View{
		Sequence: new(big.Int).Set(c.current.Sequence()),
		Round:    new(big.Int).Set(c.current.Round()),
	}
}

func (c *core) setState(state State) {
	if c.state != state {
		c.state = state
	}
	if state == StateAcceptRequest {
		c.processPendingRequests()
	}
	c.processBacklog()
}

////////////////////////////////////////////////////////////////////////////////////////////////
func (c *core) startNewRound(round *big.Int) {
	rounChanged := false
	lastProposal, lastProposer := c.server.LastProposal()
	if c.current == nil {
		c.log.Info("initiate round")
	} else if lastProposal.Height() >= c.current.Sequence().Uint64() {
		heightdiff := new(big.Int).Sub(new(big.Int).SetUint64(lastProposal.Height()), c.current.Sequence())
		c.sequenceMeter.Mark(new(big.Int).Add(heightdiff, common.Big1).Int64())
		if !c.consensusTimestamp.IsZero() {
			c.consensusTimer.UpdateSince(c.consensusTimestamp)
			c.consensusTimestamp = time.Time{}
		}
		c.log.Info("catch up latest proposal with height %d hash %s", lastProposal.Height(), lastProposal.Hash())
	} else if lastProposal.Height() == c.current.Sequence().Uint64()-1 {
		if round.Cmp(common.Big0) == 0 {
			// same req and round -> don't need to start new round
			return
		} else if round.Cmp(c.current.Round()) < 0 {
			c.log.Warn("new round %d is smaller than current round %d, NOT allowed", round, c.current.Round())
			return
		}
		rounChanged = true
	} else {
		c.log.Warn("new sequence should be larger than current sequence.")
		return
	}

	var newView *bft.View
	if rounChanged {
		newView = &bft.View{
			Sequence: new(big.Int).Set(c.current.Sequence()),
			Round:    new(big.Int).Set(round),
		}
	} else {
		newView = &bft.View{
			Sequence: new(big.Int).Add(new(big.Int).SetUint64(lastProposal.Height()), common.Big1),
			Round:    new(big.Int),
		}
		c.verSet = c.server.Verifiers(lastProposal)
	}
	//clear up
	c.roundChangeSet = newRoundChangeSet(c.verSet) //
	// update roundState
	c.updateRoundState(newView, c.verSet, rounChanged) // TODO implement updateRoundState
	// calculate new proposer
	c.verSet.CalcProposer(lastProposer, newView.Round.Uint64())
	c.waitingForRoundChange = false
	c.setState(StateAcceptRequest)
	if rounChanged && c.isProposer() && c.current != nil {
		if c.current.IsHashLocked() {
			req := &bft.Request{
				Proposal: c.current.Proposal(),
			}
			c.sendPreprepare(req)
		} else if c.current.pendingRequest != nil {
			c.sendPreprepare(c.current.pendingRequest)
		}
	}
	c.newRoundChangeTimer()
	c.log.Info("New round", "new_round", newView.Round, "new_seq", newView.Sequence, "new_proposer", c.verSet.GetProposer(), "valSet", c.verSet.List(), "size", c.verSet.Size(), "isProposer", c.isProposer())
}

func (c *core) newRoundChangeTimer() {
	c.stopTimer()

	// set timeout based on the round number
	timeout := time.Duration(c.config.RequestTimeout) * time.Millisecond
	round := c.current.Round().Uint64()
	if round > 0 {
		timeout += time.Duration(math.Pow(2, float64(round))) * time.Second
	}

	c.roundChangeTimer = time.AfterFunc(timeout, func() {
		c.sendEvent(timeoutEvent{})
	})
}

func (c *core) catchUpRound(view *bft.View) {
	if view.Round.Cmp(c.current.Round()) > 0 {
		c.roundMeter.Mark(new(big.Int).Sub(view.Round, c.current.Round()).Int64())
	}
	c.waitingForRoundChange = true

	// Need to keep block locked for round catching up
	c.updateRoundState(view, c.verSet, true)
	c.roundChangeSet.Clear(view.Round) // TODO
	c.newRoundChangeTimer()

	c.log.Debug("Catch up round. new_round %d. new_seq %d. new_proposer %s", view.Round, view.Sequence, c.valSet)
}

func (c *core) stopFuturePreprepareTimer() {
	if c.futurePreprepareTimer != nil {
		c.futurePreprepareTimer.Stop()
	}
}
