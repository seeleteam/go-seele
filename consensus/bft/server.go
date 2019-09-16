package bft

import (
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/seeleteam/go-seele/common"
)

type Server interface {
	Address() common.Address

	// Verifiers returns the Verifier set
	Verifiers(proposal Proposal) VerifierSet

	// EventMux returns the event mux in backend
	EventMux() *event.TypeMux

	// Broadcast sends a message to all Verifiers (include self)
	Broadcast(valSet VerifierSet, payload []byte) error

	// Gossip sends a message to all Verifiers (exclude self)
	Gossip(valSet VerifierSet, payload []byte) error

	// Commit delivers an approved proposal to backend.
	// The delivered proposal will be put into blockchain.
	Commit(proposal Proposal, seals [][]byte) error

	// Verify verifies the proposal. If a consensus.ErrBlockCreateTimeOld error is returned,
	// the time difference of the proposal and current time is also returned.
	Verify(Proposal) (time.Duration, error)

	// Sign signs input data with the backend's private key
	Sign([]byte) ([]byte, error)

	// CheckSignature verifies the signature by checking if it's signed by
	// the given Verifier
	CheckSignature(data []byte, addr common.Address, sig []byte) error

	// LastProposal retrieves latest committed proposal and the address of proposer
	LastProposal() (Proposal, common.Address)

	// HasPropsal checks if the combination of the given hash and height matches any existing blocks
	HasPropsal(hash common.Hash) bool

	// GetProposer returns the proposer of the given block height
	GetProposer(height uint64) common.Address

	// ParentVerifiers returns the Verifier set of the given proposal's parent block
	ParentVerifiers(proposal Proposal) VerifierSet

	// HasBadBlock returns whether the block with the hash is a bad block
	HasBadProposal(hash common.Hash) bool
}
