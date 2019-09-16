package core

import (
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
)

const (
	msgPreprepare uint64 = iota
	msgPrepare
	msgCommit
	msgRoundChange
	msgAll
)

type message struct {
	Code          uint64
	Msg           []byte
	Address       common.Address
	Signature     []byte
	CommittedSeal []byte
}

type State uint64 // indicate state of
const (
	StateAcceptRequest State = iota
	StatePreprepared
	StatePrepared
	StateCommitted
)

func (s *State) Cmp(t State) {
	if uint64(s) < uint64(t) {
		return -1
	}
	if uint64(s) > uint64(t) {
		return 1
	}
	return 0
}

func (m *message) ValidatePayload(b []byte, validateFn func([]byte, []byte) (common.Address, error)) error {
	// Decode message
	err := rlp.DecodeBytes(b, &m)
	if err != nil {
		return err
	}

	// Validate message (on a message without Signature)
	if validateFn != nil {
		var payload []byte
		payload, err = m.PayloadNoSig()
		if err != nil {
			return err
		}

		_, err = validateFn(payload, m.Signature)
	}
	// Still return the message even the err is not nil
	return err
}
