package bft

import (
	"strings"

	"github.com/seeleteam/go-seele/common"
)

type Verifier interface {
	Address() common.Address
	String() string // representation of verifier
}

type Verifiers []Verifier

func (verifiers Verifiers) Len() int {
	return len(verifiers)
}

// Less return whether one verifier is smaller than another (by verifier.String())
func (verifiers Verifiers) Less(i, j int) bool {
	return strings.Compare(verifiers[i].String(), verifiers[j].String()) < 0
}

func (verifiers Verifiers) Swap(i, j int) {
	verifiers[i], verifiers[j] = verifiers[j], verifiers[i]
}

type VerifierSet interface {
	// Calculate the proposer
	CalcProposer(lastProposer common.Address, round uint64)
	// Return the Verifier size
	Size() int
	// Return the Verifier array
	List() []Verifier
	// Get Verifier by index
	GetByIndex(i uint64) Verifier
	// Get Verifier by given address
	GetByAddress(addr common.Address) (int, Verifier)
	// Get current proposer
	GetProposer() Verifier
	// Check whether the Verifier with given address is a proposer
	IsProposer(address common.Address) bool
	// Add Verifier
	AddVerifier(address common.Address) bool
	// Remove Verifier
	RemoveVerifier(address common.Address) bool
	// Copy Verifier set
	Copy() VerifierSet
	// Get the maximum number of faulty nodes
	F() int
	// Get proposer policy
	Policy() ProposerPolicy
}
