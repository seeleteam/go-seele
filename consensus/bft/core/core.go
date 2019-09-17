package core

import (
	"bytes"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/core/types"
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

type timeoutEvent struct{}

// NewCore initiate a new core
func NewCore(server bft.Server, config *bft.BFTConfig) Engine {
	c := &core{
		config:             config,
		address:            server.Address(),
		state:              StateAcceptRequest,
		handlerWg:          new(sync.WaitGroup),
		log:                log.GetLogger("bft_core"),
		server:             server,
		backlogs:           make(map[common.Address]*prque.Prque),
		backlogsMu:         new(sync.Mutex),
		pendingRequests:    prque.New(),
		pendingRequestsMu:  new(sync.Mutex),
		consensusTimestamp: time.Time{},
		roundMeter:         metrics.GetOrRegisterMeter("consensus/bft/core/round", nil),
		sequenceMeter:      metrics.GetOrRegisterMeter("consensus/bft/core/sequence", nil),
		consensusTimer:     metrics.GetOrRegisterTimer("consensus/bft/core/consensus", nil),
	}
	c.verifyFn = c.checkValidatorSignature
	return c
}

func (c *core) checkValidatorSignature(data []byte, sig []byte) (common.Address, error) {
	return bft.CheckValidatorSignature(c.verSet, data, sig)
}

func (c *core) broadcast(msg *message) {
	payload, err := c.finalizeMessage(msg)
	if err != nil {
		c.log.Error("Failed to finalize message. msg %v. err %s. state %d", msg, err, c.state)
		return
	}

	// Broadcast payload
	if err = c.server.Broadcast(c.verSet, payload); err != nil {
		c.log.Error("Failed to broadcast message. msg %v. err %s. state %d", msg, err, c.state)
		return
	}
}

func (c *core) finalizeMessage(msg *message) ([]byte, error) {
	var err error
	msg.Address = c.Address()
	msg.CommittedSeal = []byte{}
	if msg.Code == msgCommit && c.current.Proposal() != nil {
		seal := PrepareCommittedSeal(c.current.Proposal().Hash())
		msg.CommittedSeal, err = c.server.Sign(seal)
		if err != nil {
			return nil, err
		}
	}

	data, err := msg.PayloadNoSig()
	if err != nil {
		return nil, err
	}

	msg.Signature, err = c.server.Sign(data)
	if err != nil {
		return nil, err
	}

	// convert to payload
	payload, err := msg.Payload()
	if err != nil {
		return nil, err
	}

	return payload, nil

}

// PrepareCommittedSeal returns a committed seal for the given hash
func PrepareCommittedSeal(hash common.Hash) []byte {
	var buf bytes.Buffer
	buf.Write(hash.Bytes())
	buf.Write([]byte{byte(msgCommit)})
	return buf.Bytes()
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
	c.log.Info("New round", "new_round", newView.Round, "new_seq", newView.Sequence, "new_proposer", c.verSet.GetProposer(), "verSet", c.verSet.List(), "size", c.verSet.Size(), "isProposer", c.isProposer())
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

	c.log.Debug("Catch up round. new_round %d. new_seq %d. new_proposer %s", view.Round, view.Sequence, c.verSet)
}

func (c *core) stopFuturePreprepareTimer() {
	if c.futurePreprepareTimer != nil {
		c.futurePreprepareTimer.Stop()
	}
}

func (c *core) stopTimer() {
	c.stopFuturePreprepareTimer()
	if c.roundChangeTimer != nil {
		c.roundChangeTimer.Stop()
	}
}

func (c *core) commit() {
	c.setState(StateCommitted)

	proposal := c.current.Proposal()
	if proposal != nil {
		committedSeals := make([][]byte, c.current.Commits.Size())
		for i, v := range c.current.Commits.Values() {
			committedSeals[i] = make([]byte, types.IstanbulExtraSeal)
			copy(committedSeals[i][:], v.CommittedSeal[:])
		}

		if err := c.server.Commit(proposal, committedSeals); err != nil {
			c.current.UnlockHash() //Unlock block when insertion fails
			c.sendNextRoundChange()
			return
		}
	}
}

func (c *core) Address() common.Address {
	return c.address
}

func (c *core) isProposer() bool {
	v := c.verSet
	if v == nil {
		return false
	}
	return v.IsProposer(c.server.Address())
}
