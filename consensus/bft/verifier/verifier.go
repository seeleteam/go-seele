package verifier

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

type verifier struct {
	address common.Address
}

func NewVerifier(addr common.Address) bft.Verifier {
	return &basicVerifier{
		address: addr,
	}
}

func NewVerifierSet(addrs []common.Address, policy bft.ProposerPolicy) bft.VerifierSet {
	return newBasicSet(addrs, policy)
}

func ExtractVerifiers(extraData []byte) []common.Address {
	// get all verifiers address
	addrs := make([]common.Address, (len(extraData) / common.AddressLen))
	for i := 0; i < len(addrs); i++ {
		copy(addrs[i][:], extraData[i*common.AddressLen:])
	}
	return addrs
}

func ValidExtraData(extraData []byte) bool {
	return len(extraData)%common.AddressLen == 0
}
