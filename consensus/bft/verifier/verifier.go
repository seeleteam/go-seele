package verifier

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

type verifier struct {
	address common.Address
}

func NewVerifier(addr common.Address) bft.Verifier {
	return &verifier{
		address: addr,
	}
}

func NewVerifierSet(addrs []common.Address, policy bft.ProposerPolicy) bft.VerifierSet {
	return newBasicSet(addrs, policy)
}
