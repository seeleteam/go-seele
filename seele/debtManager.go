/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"runtime"
	"sync"
	"time"

	"github.com/Jeffail/tunny"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

type propagateDebts interface {
	// propagateDebtMap send debts to other connected peers.
	// filter whether filter debt when it is marked as known debt for peer.
	propagateDebtMap(debtsMap [][]*types.Debt, filter bool)
}

const (
	checkInterval = 12 * common.BlockPackInterval
)

type DebtInfo struct {
	debt               *types.Debt
	lastCheckTimestamp time.Time

	// debt is packed, but not confirmed. confirmed block will be removed from debt manager.
	isPacked bool
}

type DebtManager struct {
	debts map[common.Hash]*DebtInfo
	lock  *sync.RWMutex

	checker     types.DebtVerifier
	propagation propagateDebts
	log         *log.SeeleLog
}

func NewDebtManager(debtChecker types.DebtVerifier, p propagateDebts) *DebtManager {
	return &DebtManager{
		debts:       make(map[common.Hash]*DebtInfo),
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
		m.debts[d.Hash] = &DebtInfo{
			debt:               d,
			lastCheckTimestamp: time.Now(),
		}
	}
}

func (m *DebtManager) AddDebtMap(debtMap [][]*types.Debt) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, debts := range debtMap {
		for _, d := range debts {
			m.debts[d.Hash] = &DebtInfo{
				debt:               d,
				lastCheckTimestamp: time.Now(),
			}
		}
	}
}

func (m *DebtManager) Remove(hash common.Hash) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.debts, hash)
}

func (m *DebtManager) GetAll() []*DebtInfo {
	m.lock.RLock()
	defer m.lock.RUnlock()

	results := make([]*DebtInfo, len(m.debts))
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
	pool := tunny.NewFunc(runtime.NumCPU(), func(i interface{}) interface{} {
		defer wg.Done()
		info := i.(*DebtInfo)
		debt := info.debt
		if time.Now().Sub(info.lastCheckTimestamp) > checkInterval {
			packed, confirmed, err := m.checker.IfDebtPacked(debt)

			// remove invalid block or confirmed block.
			if err != nil || confirmed {
				if confirmed {
					m.log.Info("remove debt as confirmed. hash:%s", debt.Hash.Hex())
				} else {
					m.log.Warn("remove debt cause got err when checking. err:%s. hash:%s", err, debt.Hash.Hex())
				}

				m.Remove(debt.Hash)
			}

			info.isPacked = packed
			info.lastCheckTimestamp = time.Now()
		}

		return nil
	})

	for _, d := range toChecking {
		wg.Add(1)
		pool.Process(d)
	}

	wg.Wait()
	pool.Close()

	// resend
	toSend := make([][]*types.Debt, common.ShardCount+1)
	for _, info := range toChecking {
		// if the debt is not packed or confirmed, we will send it again.
		if !info.isPacked && m.Has(info.debt.Hash) {
			shard := info.debt.Data.Account.Shard()
			toSend[shard] = append(toSend[shard], info.debt)

			m.log.Warn("debt is not packed or confirmed, send again. hash:%s", info.debt.Hash.Hex())
		}
	}

	m.propagation.propagateDebtMap(toSend, false)
}

func (m *DebtManager) TimingChecking() {
	for {
		m.log.Debug("start checking")
		m.checking()

		time.Sleep(2 * checkInterval)
	}
}
