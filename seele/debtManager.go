/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"runtime"
	"sync"
	"time"
	"encoding/binary"

	"github.com/Jeffail/tunny"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/database"
)

type propagateDebts interface {
	// propagateDebtMap send debts to other connected peers.
	// filter whether filter debt when it is marked as known debt for peer.
	propagateDebtMap(debtsMap [][]*types.Debt, filter bool)
}

const (
	checkInterval = 12 * common.BlockPackInterval
)

var maxDebtBatchSize = 5000

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
	chain       *core.Blockchain
	blockHeights []uint64 
	dmDB        database.Database
}

func NewDebtManager(debtChecker types.DebtVerifier, p propagateDebts, chain *core.Blockchain, debtManagerDB database.Database) *DebtManager {
	return &DebtManager{
		debts:       make(map[common.Hash]*DebtInfo),
		checker:     debtChecker,
		lock:        &sync.RWMutex{},
		propagation: p,
		log:         log.GetLogger("debt_manager"),
		chain:       chain,
		dmDB:        debtManagerDB, 
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

func (m *DebtManager) AddDebtMap(debtMap [][]*types.Debt, height uint64) {
	m.lock.Lock()
	defer m.lock.Unlock()

	var ToBeStoredDebts []*types.Debt
	for _, debts := range debtMap {
		for _, d := range debts {
			if len(m.debts) < core.DebtManagerPoolCapacity {
				m.debts[d.Hash] = &DebtInfo{
					debt:               d,
					lastCheckTimestamp: time.Now(),
				}
			} else {
				// debtManager pool is full, store the debts in the database
				if len(ToBeStoredDebts) == 0 {
					m.blockHeights = append(m.blockHeights, height)
				}
				     
				ToBeStoredDebts = append(ToBeStoredDebts, d) 
			}

		}
	}

	// commit the debts to the debtManager database
	if len(ToBeStoredDebts) > 0 {
		batch := m.dmDB.NewBatch()
		encoded := make([]byte, 8)
		binary.BigEndian.PutUint64(encoded, height)
		batch.Put(encoded, common.SerializePanic(ToBeStoredDebts))
		err := batch.Commit()
		if err != nil {
			m.log.Warn("failed to store extra debts in database, err %s", err)
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

// checking resend debt if it is not packed after timeout
func (m *DebtManager) checking() {
	toChecking := m.GetAll()

	wg := sync.WaitGroup{}
	pool := tunny.NewFunc(runtime.NumCPU(), func(i interface{}) interface{} {
		defer wg.Done()
		info := i.(*DebtInfo)
		debt := info.debt
		if time.Now().Sub(info.lastCheckTimestamp) > checkInterval {
			packed, confirmed, err := m.checker.IfDebtPacked(debt)

			// remove confirmed debt.
			if err != nil || confirmed {
				if confirmed {
					m.log.Debug("remove debt as confirmed. hash:%s", debt.Hash.Hex())
					m.Remove(debt.Hash)
				} else {
					m.log.Debug("got err when checking. err:%s. hash:%s", err, debt.Hash.Hex())
				}
			}

			// remove invalid debt
			_, err = m.chain.GetStore().GetTxIndex(debt.Data.TxHash)
			if err != nil {
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
			if len(toSend[shard]) < maxDebtBatchSize {
				toSend[shard] = append(toSend[shard], info.debt)
			}

			m.log.Debug("debt is not packed or confirmed, send again. hash:%s", info.debt.Hash.Hex())
		}
	}

	m.propagation.propagateDebtMap(toSend, false)

	err := m.reinjectDebtFromDatabase()
	if err != nil {
		m.log.Warn("Error in debt reinjection")
	}
}

func (m *DebtManager) TimingChecking() {
	for {
		m.log.Debug("start checking")
		m.checking()

		time.Sleep(2 * checkInterval)
	}
}

func (m *DebtManager) reinjectDebtFromDatabase() error {
	if len(m.blockHeights) > 0 {
		n := len(m.blockHeights)
		i := 0
		for i < n && i < 30 {
			// scan debts from at most 30 blocks
			height := m.blockHeights[0]
			key := make([]byte, 8)
			binary.BigEndian.PutUint64(key, height)
			value, err := m.dmDB.Get(key)
			if err != nil {
				return err
			}

			var debts []*types.Debt
			if err = common.Deserialize(value, &debts); err != nil {
				panic(err)
			}
			m.log.Debug("Got debts from database. height: %d, hash of the first debt:%s", height, debts[0].Hash.Hex())

			debtMap := make([][]*types.Debt, common.ShardCount + 1)
			for _, d := range debts {
				if d != nil {
					shard := d.Data.Account.Shard()
					debtMap[shard] = append(debtMap[shard], d)
				}
			}

			// remove the debts from debt manager database
			if err := m.dmDB.Delete(key); err != nil {
				m.log.Debug("Failed to delete debts from database.")
			}
			m.blockHeights = m.blockHeights[1:]

			// reinject debts to debt manager pool; if the debt manager 
			// pool is full, the debts will go back to the database
			m.AddDebtMap(debtMap, height)

			i++
		}
	}
	return nil

}
