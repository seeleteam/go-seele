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

// ExtractVerifiers get all verifiers address from extraData
func ExtractVerifiers(extraData []byte) []common.Address {
	addrs := make([]common.Address, (len(extraData) / common.AddressLen)) // need to be careful for the len of verifier
	for i := 0; i < len(addrs); i++ {
		copy(addrs[i][:], extraData[i*common.AddressLen:])
	}
	return addrs
}

// ValidateExtraData : only allows extraData = n * verifierLen
func ValidExtraData(extraData []byte) bool {
	return len(extraData)%common.AddressLen == 0
}
