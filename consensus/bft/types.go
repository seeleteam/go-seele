package bft

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

type Subject struct {
	View   *View
	Digest common.Hash
}

// View includes a round number and a sequence number.
// Sequence is the block number we'd like to commit.
// Each round has a number and is composed by 3 steps: preprepare, prepare and commit.
//
// If the given block is not accepted by validators, a round change will occur
// and the validators start a new round with round+1.
type View struct {
	Round    *big.Int
	Sequence *big.Int
}

type Request struct {
	Proposal Proposal
}

type Preprepare struct {
	View     *View
	Proposal Proposal
}

type Proposal interface {
	// Height retrieves the sequence number of this proposal.
	Height() uint64

	// Hash retrieves the hash of this proposal.
	Hash() common.Hash
}

type RequestEvent struct {
	Proposal Proposal
}

type MessageEvent struct {
	Payload []byte
}

type FinalCommittedEvent struct {
}
