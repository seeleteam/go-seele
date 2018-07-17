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
	// revert reverts the state change in the specified statedb
	revert(*Statedb)

	// dirtyAccount returns the account address of dirty data in statedb.
	// Return nil if the changed data not saved in statedb.
	dirtyAccount() *common.Address
}

type journal struct {
	entries []journalEntry
	dirties map[common.Address]uint
}

func newJournal() *journal {
	return &journal{
		dirties: make(map[common.Address]uint),
	}
}

func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtyAccount(); addr != nil {
		j.dirties[*addr]++
	}
}

func (j *journal) snapshot() int {
	return len(j.entries)
}

func (j *journal) revert(statedb *Statedb, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		j.entries[i].revert(statedb)

		if addr := j.entries[i].dirtyAccount(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}

		j.entries[i] = nil
	}

	j.entries = j.entries[:snapshot]
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

func (ch refundChange) dirtyAccount() *common.Address {
	return nil
}

func (ch storageChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prev)
}

func (ch storageChange) dirtyAccount() *common.Address {
	return ch.account
}

func (ch balanceChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).account.Amount = ch.prev
}

func (ch balanceChange) dirtyAccount() *common.Address {
	return ch.account
}

func (ch codeChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).setCode(ch.prev)
}

func (ch codeChange) dirtyAccount() *common.Address {
	return ch.account
}

func (ch nonceChange) revert(s *Statedb) {
	s.getStateObject(*ch.account).account.Nonce = ch.prev
}

func (ch nonceChange) dirtyAccount() *common.Address {
	return ch.account
}

func (ch suicideChange) revert(s *Statedb) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prevSuicided
		obj.account.Amount = ch.prevBalance
	}
}

func (ch suicideChange) dirtyAccount() *common.Address {
	return ch.account
}

func (ch createObjectChange) revert(s *Statedb) {
	delete(s.stateObjects, *ch.account)
}

func (ch createObjectChange) dirtyAccount() *common.Address {
	return ch.account
}
