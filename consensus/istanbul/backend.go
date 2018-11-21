/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package istanbul

import (
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/seeleteam/go-seele/common"
)

// Backend provides application specific functions for Istanbul core
type Backend interface {
	// Address returns the owner's address
	Address() common.Address

	// Validators returns the validator set
	Validators(proposal Proposal) ValidatorSet

	// EventMux returns the event mux in backend
	EventMux() *event.TypeMux

	// Broadcast sends a message to all validators (include self)
	Broadcast(valSet ValidatorSet, payload []byte) error

	// Gossip sends a message to all validators (exclude self)
	Gossip(valSet ValidatorSet, payload []byte) error

	// Commit delivers an approved proposal to backend.
	// The delivered proposal will be put into blockchain.
	Commit(proposal Proposal, seals [][]byte) error

	// Verify verifies the proposal. If a consensus.ErrBlockCreateTimeOld error is returned,
	// the time difference of the proposal and current time is also returned.
	Verify(Proposal) (time.Duration, error)

	// Sign signs input data with the backend's private key
	Sign([]byte) ([]byte, error)

	// CheckSignature verifies the signature by checking if it's signed by
	// the given validator
	CheckSignature(data []byte, addr common.Address, sig []byte) error

	// LastProposal retrieves latest committed proposal and the address of proposer
	LastProposal() (Proposal, common.Address)

	// HasPropsal checks if the combination of the given hash and height matches any existing blocks
	HasPropsal(hash common.Hash) bool

	// GetProposer returns the proposer of the given block height
	GetProposer(height uint64) common.Address

	// ParentValidators returns the validator set of the given proposal's parent block
	ParentValidators(proposal Proposal) ValidatorSet

	// HasBadBlock returns whether the block with the hash is a bad block
	HasBadProposal(hash common.Hash) bool
}
