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
	mutex   sync.Mutex
}

func NewDebtPool() *DebtPool {
	return &DebtPool{
		hashMap: make(map[common.Hash]*types.Debt, 0),
		mutex:   sync.Mutex{},
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
	dp.mutex.Unlock()

	delete(dp.hashMap, hash)
}
