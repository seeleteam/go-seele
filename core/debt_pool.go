/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"bytes"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
)

var DebtDataFlag = []byte{0x01}

// DebtPool debt pool
type DebtPool struct {
	hashMap map[common.Hash]*types.Debt
	mutex   sync.RWMutex

	chain blockchain
	log   *log.SeeleLog

	verifier types.DebtVerifier
}

func NewDebtPool(chain blockchain, verifier types.DebtVerifier) *DebtPool {
	return &DebtPool{
		hashMap:  make(map[common.Hash]*types.Debt, 0),
		mutex:    sync.RWMutex{},
		chain:    chain,
		log:      log.GetLogger("debtpool"),
		verifier: verifier,
	}
}

func (dp *DebtPool) HandleChainHeaderChanged(newHeader, lastHeader common.Hash) {
	reinject := dp.getReinjectDebts(newHeader, lastHeader)
	if len(reinject) > 0 {
		dp.log.Info("reinject %d debts", len(reinject))
	}

	dp.add(reinject)
	dp.removeDebts()
}

func (dp *DebtPool) getReinjectDebts(newHeader, lastHeader common.Hash) []*types.Debt {
	chainStore := dp.chain.GetStore()
	log := dp.log

	newBlock, err := chainStore.GetBlock(newHeader)
	if err != nil {
		log.Error("got block failed, %s", err)
		return nil
	}

	if newBlock.Header.PreviousBlockHash != lastHeader {
		lastBlock, err := chainStore.GetBlock(lastHeader)
		if err != nil {
			log.Error("got block failed, %s", err)
			return nil
		}

		log.Debug("handle chain header forked, last height %d, new height %d", lastBlock.Header.Height, newBlock.Header.Height)
		// add committed debts back in current branch.
		toDeleted := make(map[common.Hash]*types.Debt)
		toAdded := make(map[common.Hash]*types.Debt)
		for newBlock.Header.Height > lastBlock.Header.Height {
			for _, d := range newBlock.Debts {
				toDeleted[d.Hash] = d
			}

			if newBlock, err = chainStore.GetBlock(newBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		for lastBlock.Header.Height > newBlock.Header.Height {
			for _, d := range lastBlock.Debts {
				toAdded[d.Hash] = d
			}

			if lastBlock, err = chainStore.GetBlock(lastBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		for lastBlock.HeaderHash != newBlock.HeaderHash {
			for _, d := range lastBlock.Debts {
				toAdded[d.Hash] = d
			}

			for _, d := range newBlock.Debts {
				toDeleted[d.Hash] = d
			}

			if lastBlock, err = chainStore.GetBlock(lastBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}

			if newBlock, err = chainStore.GetBlock(newBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		reinject := make([]*types.Debt, 0)
		for key, d := range toAdded {
			if _, ok := toDeleted[key]; !ok {
				reinject = append(reinject, d)
			}
		}

		log.Debug("to added tx length %d, to deleted tx length %d, to reinject tx length %d",
			len(toAdded), len(toDeleted), len(reinject))
		return reinject
	}

	return nil
}

func (dp *DebtPool) removeDebts() {
	dp.mutex.Lock()
	defer dp.mutex.Unlock()

	state, err := dp.chain.GetCurrentState()
	if err != nil {
		dp.log.Warn("failed to get current state, err: %s", err)
		return
	}

	for _, d := range dp.hashMap {
		if !state.Exist(d.Data.Account) {
			continue
		}

		data := state.GetData(d.Data.Account, d.Hash)
		if bytes.Equal(data, DebtDataFlag) {
			delete(dp.hashMap, d.Hash)
		}
	}
}

func (dp *DebtPool) AddWithValidation(debts []*types.Debt) {
	var results []*types.Debt

	for _, d := range debts {
		if dp.Has(d.Hash) {
			continue
		}

		err := d.Validate(dp.verifier, true, common.LocalShardNumber)
		if err != nil {
			dp.log.Warn("validate debt failed. err %s", err)
			continue
		}

		results = append(results, d)
	}

	dp.add(results)
	event.DebtsInsertedEventManager.Fire(results)
}

func (dp *DebtPool) add(debts []*types.Debt) {
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

func (dp *DebtPool) Has(debt common.Hash) bool {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	return dp.hashMap[debt] != nil
}

func (dp *DebtPool) GetDebtByHash(debt common.Hash) *types.Debt {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	return dp.hashMap[debt]
}

func (dp *DebtPool) GetAll() []*types.Debt {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	results := make([]*types.Debt, 0)
	for _, v := range dp.hashMap {
		results = append(results, v)
	}

	return results
}
