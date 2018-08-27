/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// DebtPool debt pool
type DebtPool struct {
	hashMap map[common.Hash]*types.Debt
	mutex   sync.RWMutex
}

func NewDebtPool() *DebtPool {
	return &DebtPool{
		hashMap: make(map[common.Hash]*types.Debt, 0),
		mutex:   sync.RWMutex{},
	}
}

func (dp *DebtPool) Add(debts []*types.Debt) {
	dp.mutex.Lock()
	defer dp.mutex.Unlock()

	for _, debt := range debts {
		dp.hashMap[debt.Hash] = debt
	}
}

func (dp *DebtPool) Remove(hash common.Hash) {
	dp.mutex.Lock()
	defer dp.mutex.Unlock()

	delete(dp.hashMap, hash)
}

func (dp *DebtPool) Get(size int) ([]*types.Debt, int) {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	remainSize := size
	results := make([]*types.Debt, 0)
	for _, d := range dp.hashMap {
		tmp := remainSize - d.Size()
		if tmp > 0 {
			remainSize = tmp
			results = append(results, d)
		}
	}

	return results, remainSize
}
