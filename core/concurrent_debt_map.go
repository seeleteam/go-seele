/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package core

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/types"
)

var errDebtFull = errors.New("too many debts in to confirmed debt")

type ConcurrentDebtMap struct {
	capacity int
	lock     sync.RWMutex
	value    map[common.Hash]*types.Debt
}

func NewConcurrentDebtMap(capacity int) *ConcurrentDebtMap {
	return &ConcurrentDebtMap{
		value:    make(map[common.Hash]*types.Debt),
		capacity: capacity,
		lock:     sync.RWMutex{},
	}
}

func (m *ConcurrentDebtMap) count() int {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return len(m.value)
}

func (m *ConcurrentDebtMap) remove(hash common.Hash) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.value, hash)
}

// removeByValue remove debts by Debt
func (m *ConcurrentDebtMap) removeByValue(debt *types.Debt) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.value, debt.Hash)
}

func (m *ConcurrentDebtMap) add(debt *types.Debt) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.value) >= m.capacity && m.value[debt.Hash] == nil {
		return errDebtFull
	}

	m.value[debt.Hash] = debt
	return nil
}

func (m *ConcurrentDebtMap) get(hash common.Hash) *types.Debt {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.value[hash]
}

func (m *ConcurrentDebtMap) has(hash common.Hash) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.value[hash] != nil
}

func (m *ConcurrentDebtMap) items() map[common.Hash]*types.Debt {
	tmp := make(map[common.Hash]*types.Debt)

	m.lock.RLock()
	defer m.lock.RUnlock()
	for h, d := range m.value {
		tmp[h] = d
	}

	return tmp
}

func (m *ConcurrentDebtMap) getList() []*types.Debt {
	m.lock.RLock()
	defer m.lock.RUnlock()

	tmp := make([]*types.Debt, len(m.value))
	i := 0
	for _, d := range m.value {
		tmp[i] = d
		i++
	}

	return tmp
}
