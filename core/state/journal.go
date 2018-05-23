/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

type journalEntry interface {
	revert(*Statedb)
}

type journal struct {
	entries []journalEntry
}

func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
}

func (j *journal) revert(statedb *Statedb) {
	for i := len(j.entries) - 1; i >= 0; i-- {
		j.entries[i].revert(statedb)
	}

	j.entries = j.entries[:0]
}

type (
	refundChange struct {
		prev uint64
	}
	storageChange struct {
		account *common.Address
		key     common.Hash
		prev    common.Hash
	}
	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	codeChange struct {
		account *common.Address
		prev    []byte
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	suicideChange struct {
		account      *common.Address
		prevSuicided bool
		prevBalance  *big.Int
	}
	createObjectChange struct {
		account *common.Address
	}
)

func (ch refundChange) revert(s *Statedb) {
	s.refund = ch.prev
}

func (ch storageChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prev)
}

func (ch balanceChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).account.Amount = ch.prev
}

func (ch codeChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).setCode(ch.prev)
}

func (ch nonceChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).account.Nonce = ch.prev
}

func (ch suicideChange) revert(s *Statedb) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prevSuicided
		obj.account.Amount = ch.prevBalance
	}
}

func (ch createObjectChange) revert(s *Statedb) {
	s.stateObjects.Remove(*ch.account)
}
