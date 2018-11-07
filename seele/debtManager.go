/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

type propagateDebts interface {
	propagateDebtMap(debtsMap [][]*types.Debt, filter bool)
}

type DebtManager struct {
	debts map[common.Hash]*types.Debt
	lock  *sync.RWMutex

	checker     types.DebtVerifier
	propagation propagateDebts
	log         *log.SeeleLog
}

func NewDebtManager(debtChecker types.DebtVerifier, p propagateDebts) *DebtManager {
	return &DebtManager{
		debts:       make(map[common.Hash]*types.Debt),
		checker:     debtChecker,
		lock:        &sync.RWMutex{},
		propagation: p,
		log:         log.GetLogger("debt_manager"),
	}
}

func (m *DebtManager) AddDebts(debts []*types.Debt) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, d := range debts {
		m.debts[d.Hash] = d
	}
}

func (m *DebtManager) AddDebtMap(debtMap [][]*types.Debt) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, debts := range debtMap {
		for _, d := range debts {
			m.debts[d.Hash] = d
		}
	}
}

func (m *DebtManager) Remove(hash common.Hash) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.debts, hash)
}

func (m *DebtManager) GetAll() []*types.Debt {
	m.lock.RLock()
	defer m.lock.RUnlock()

	results := make([]*types.Debt, len(m.debts))
	index := 0
	for _, d := range m.debts {
		results[index] = d
		index++
	}

	return results
}

func (m *DebtManager) Has(hash common.Hash) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.debts[hash] != nil
}

func (m *DebtManager) checking() {
	toChecking := m.GetAll()

	wg := sync.WaitGroup{}
	for _, d := range toChecking {
		wg.Add(1)
		go func() {
			ok, err := m.checker.CheckIfDebtPacked(d)
			if err != nil || ok {
				if ok {
					m.log.Info("remove debt as packed %s", d.Hash.ToHex())
				} else {
					m.log.Warn("remove debt cause got err when checking. err:%s. hash:%s", err, d.Hash.ToHex())
				}

				m.Remove(d.Hash)
			}

			wg.Done()
		}()
	}

	wg.Wait()

	// resend
	toSend := make([][]*types.Debt, common.ShardCount+1)
	for _, d := range toChecking {
		if m.Has(d.Hash) {
			shard := d.Data.Account.Shard()
			toSend[shard] = append(toSend[shard], d)

			m.log.Warn("not found debt info, send again. hash:%s", d.Hash.ToHex())
		}
	}

	m.propagation.propagateDebtMap(toSend, false)
}

func (m *DebtManager) TimingChecking() {
	for {
		m.log.Debug("start checking")
		m.checking()

		time.Sleep(5 * common.BlockPackInterval)
	}
}
