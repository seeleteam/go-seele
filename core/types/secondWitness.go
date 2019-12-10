package types

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
)

type SecondWitnessExtra struct {
	DepositVers []common.Address
	ExitVers    []common.Address
}

func (swExtra *SecondWitnessExtra) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		swExtra.DepositVers,
		swExtra.ExitVers,
	})
}

func (swExtra *SecondWitnessExtra) DecodeRLP(s *rlp.Stream) error {
	var secondWitnessExtra struct {
		DepositVers []common.Address
		ExitVers    []common.Address
	}
	if err := s.Decode(&secondWitnessExtra); err != nil {
		return err
	}
	swExtra.DepositVers, swExtra.ExitVers = secondWitnessExtra.DepositVers, secondWitnessExtra.ExitVers
	return nil
}

// ExtractSWExtra extract verifiers from SecondWitness
func ExtractSWExtra(h *BlockHeader) (*SecondWitnessExtra, error) {
	// if len(h.ExtraData) < BftExtraVanity {
	// 	fmt.Printf("header extra data len %d is smaller than BftExtraVanity %d\n", len(h.ExtraData), BftExtraVanity)
	// 	return nil, ErrInvalidBftHeaderExtra
	// }

	var swExtra *SecondWitnessExtra
	err := rlp.DecodeBytes(h.SecondWitness, &swExtra)
	if err != nil {
		return nil, err
	}
	return swExtra, nil
}
