package main

import (
	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/istanbul"
	"github.com/seeleteam/go-seele/consensus/pbft/network"
	"github.com/seeleteam/go-seele/consensus/utils"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/rpc"
)

/*
	1. startNode
	2. send request
	3. primary node pre-prepare and send to normal operators
	4. normal peer verify pre-prepare message and then prepare message
	5.
*/
type PBFTEngine struct {
	threads  int
	log      *log.SeeleLog
	hashrate metrics.Meter
}

func newPBFTEngin(threads int) *PBFTEngine {
	return &PBFTEngine{
		threads:  threads,
		log:      log.GetLogger("pbft_engine"),
		hashrate: metrics.NewMeter(),
	}
}

func (engine *PBFTEngine) StartEngine(nodeID common.Address) {
	server := network.NewServer(nodeID.String())
	server.Start()
}

func (engine *PBFTEngine) SetThreads(threads int) {
	if threads < 0 {
		engine.threads = int(1)
	} else {
		engine.threads = threads
	}
}

func (engine *PBFTEngine) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{
		{
			Namespace: "miner",
			Version:   "1.0",
			Service:   &API{engine},
			Public:    true,
		},
	}
}

func (engine *PBFTEngine) VerifyHeader(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}
	if err := utils.VerifyHeaderCommon(header, parent); err != nil {
		return err
	}
	return nil
}

func (engine *PBFTEngine) Prepare(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}
	// header.Difficulty = utils.GetDifficult(header.CreateTimestamp.Uint64(), parent)

	// TODO send request from here!
	return nil

}

// func (engine *PBFTEngine) Seal (reader consensus.ChainReader, block *) {}

func (engine *PBFTEngine) VerifyTarget(header *types.BlockHeader) error {
	newHeader := header.Clone()
	/*
		verify target : verify all commit messages
		if all messages are valid, then broadcast this block as pow algorithm
	*/
	return nil
}

// CommitMessage commit message after VerifyTarget
func (engine *PBFTEngine) CommitMessage() error {
	return nil
}

func (engine *PBFTEngine) Seal() error {
	return nil
}

func (c *core) handleRequest(request *istanbul.Request) error {
	if err := c.checkRequestMsg(request); err != nil {
		if err == errInvalidMessage {
			c.logger.Warn("invalid request")
			return err
		}
		c.logger.Warn("unexpected request. err %s. height %d. hash %s", err, request.Proposal.Height(), request.Proposal.Hash())
		return err
	}

	c.logger.Debug("handleRequest. height %d. hash %s", request.Proposal.Height(), request.Proposal.Hash())

	c.current.pendingRequest = request
	if c.state == StateAcceptRequest {
		c.sendPreprepare(request)
	}
	return nil
}

// check request state
// return errInvalidMessage if the message is invalid
// return errFutureMessage if the sequence of proposal is larger than current sequence
// return errOldMessage if the sequence of proposal is smaller than current sequence
func (c *core) checkRequestMsg(request *istanbul.Request) error {
	if request == nil || request.Proposal == nil {
		return errInvalidMessage
	}

	if c.current.sequence.Uint64() > request.Proposal.Height() {
		return errOldMessage
	} else if c.current.sequence.Uint64() < request.Proposal.Height() {
		return errFutureMessage
	} else {
		return nil
	}
}

func (c *core) storeRequestMsg(request *istanbul.Request) {
	c.logger.Debug("Store future request. height %d. hash %s. state %d", request.Proposal.Height(), request.Proposal.Hash(), c.state)

	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	c.pendingRequests.Push(request, float32(-request.Proposal.Height()))
}

func (c *core) processPendingRequests() {
	c.pendingRequestsMu.Lock()
	defer c.pendingRequestsMu.Unlock()

	for !(c.pendingRequests.Empty()) {
		m, prio := c.pendingRequests.Pop()
		r, ok := m.(*istanbul.Request)
		if !ok {
			c.logger.Warn("Malformed request, skip. msg %v", m)
			continue
		}
		// Push back if it's a future message
		err := c.checkRequestMsg(r)
		if err != nil {
			if err == errFutureMessage {
				c.logger.Debug("Stop processing request height %d. hash %s", r.Proposal.Height(), r.Proposal.Hash())
				c.pendingRequests.Push(m, prio)
				break
			}
			c.logger.Debug("Skip the pending request err %s. height %d. hash %s", err, r.Proposal.Height(), r.Proposal.Hash())
			continue
		}
		c.logger.Debug("Post pending request height %d. hash %s", r.Proposal.Height(), r.Proposal.Hash())

		go c.sendEvent(istanbul.RequestEvent{
			Proposal: r.Proposal,
		})
	}
}
