package types

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
)

var (
	// BftWitness represents a hash of "seeles practical byzantine fault tolerance"
	// to identify whether the block is from Bft consensus engine
	BftWitness = common.MustHexToHash("0x7365656c65732062797a616e74696e65206661756c7420746f6c6572616e6365").Bytes()

	BftExtraVanity = 32 // Fixed number of extra-data bytes reserved for verifier vanity
	BftExtraSeal   = 65 // Fixed number of extra-data bytes reserved for verifier seal

	// ErrInvalidBftHeaderExtra is returned if the length of extra-data is less than 32 bytes
	ErrInvalidBftHeaderExtra = errors.New("invalid BFT header extra-data")
)

// BftExtra will be used as ExtraData in block for bft consensus algorithm
type BftExtra struct {
	Verifiers     []common.Address
	Seal          []byte
	CommittedSeal [][]byte
}

// EncodeRLP serializes bftExtra into the Ethereum RLP format.
func (bftExtra *BftExtra) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		bftExtra.Verifiers,
		bftExtra.Seal,
		bftExtra.CommittedSeal,
	})
}

// DecodeRLP implements rlp.Decoder, and load the bft fields from a RLP stream.
func (bftExtra *BftExtra) DecodeRLP(s *rlp.Stream) error {
	var bftBlockExtra struct {
		Verifiers     []common.Address
		Seal          []byte
		CommittedSeal [][]byte
	}
	if err := s.Decode(&bftBlockExtra); err != nil {
		return err
	}
	bftExtra.Verifiers, bftExtra.Seal, bftExtra.CommittedSeal = bftBlockExtra.Verifiers, bftBlockExtra.Seal, bftBlockExtra.CommittedSeal
	return nil
}

// ExtractBftExtra extracts all values of the BftExtra from the header. !!!
// It returns an error if the length of the given extra-data is less than 32 bytes or the extra-data can not be decoded.
func ExtractBftExtra(h *BlockHeader) (*BftExtra, error) {
	if len(h.ExtraData) < BftExtraVanity {
		fmt.Printf("header extra data len %d is smaller than BftExtraVanity %d\n", len(h.ExtraData), BftExtraVanity)
		return nil, ErrInvalidBftHeaderExtra
	}

	var bftExtra *BftExtra
	err := rlp.DecodeBytes(h.ExtraData[BftExtraVanity:], &bftExtra)
	if err != nil {
		return nil, err
	}
	return bftExtra, nil
}

// BftFilteredHeader returns a filtered header which some information (like seal, committed seals)
// are clean to fulfill the Bft hash rules. It returns nil if the extra-data cannot be
// decoded/encoded by rlp.
func BftFilteredHeader(h *BlockHeader, keepSeal bool) *BlockHeader {
	newHeader := h.Clone()
	bftExtra, err := ExtractBftExtra(newHeader)
	if err != nil {
		return nil
	}

	if !keepSeal {
		bftExtra.Seal = []byte{}
	}
	bftExtra.CommittedSeal = [][]byte{}

	payload, err := rlp.EncodeToBytes(&bftExtra)
	if err != nil {
		return nil
	}

	newHeader.ExtraData = append(newHeader.ExtraData[:BftExtraVanity], payload...)

	return newHeader
}
