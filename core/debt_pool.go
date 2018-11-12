/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"bytes"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
)

// DebtPool debt pool
type DebtPool struct {
	*Pool
}

func NewDebtPool(chain blockchain, verifier types.DebtVerifier) *DebtPool {
	log := log.GetLogger("debtpool")

	getObjectFromBlock := func(block *types.Block) []poolObject {
		var results []poolObject
		for _, obj := range block.Debts {
			results = append(results, obj)
		}

		return results
	}

	canRemove := func(chain blockchain, state *state.Statedb, item *poolItem) bool {
		if !state.Exist(item.ToAccount()) {
			return false
		}

		data := state.GetData(item.ToAccount(), item.GetHash())
		if bytes.Equal(data, types.DebtDataFlag) {
			return true
		}

		return false
	}

	objectValidation := func(state *state.Statedb, obj poolObject) error {
		debt := obj.(*types.Debt)
		_, err := debt.Validate(verifier, true, common.LocalShardNumber)
		if err != nil {
			return errors.NewStackedError(err, "validate debt failed")
		}

		return nil
	}

	afterAdd := func(obj poolObject) {
		log.Debug("receive debt and add it. debt hash: %v, time: %d", obj.GetHash(), time.Now().UnixNano())

		event.DebtsInsertedEventManager.Fire(obj.(*types.Debt))
	}

	pool := NewPool(DebtPoolCapacity, chain, getObjectFromBlock, canRemove, log, objectValidation, afterAdd)

	return &DebtPool{
		Pool: pool,
	}
}

func (dp *DebtPool) AddWithValidation(debts []*types.Debt) {
	for _, d := range debts {
		if d == nil {
			continue
		}

		dp.addObject(d)
	}
}

func (dp *DebtPool) GetProcessableDebts(size int) ([]*types.Debt, int) {
	objects, remainSize := dp.getProcessableObjects(size)

	return objectsToDebts(objects), remainSize
}

func objectsToDebts(objects []poolObject) []*types.Debt {
	results := make([]*types.Debt, len(objects))
	for index, obj := range objects {
		results[index] = obj.(*types.Debt)
	}

	return results
}

func (dp *DebtPool) GetDebtByHash(debt common.Hash) *types.Debt {
	return dp.GetObject(debt).(*types.Debt)
}

func (dp *DebtPool) RemoveDebtByHash(hash common.Hash) {
	dp.removeOject(hash)
}

func (dp *DebtPool) GetDebts(processing, pending bool) []*types.Debt {
	objects := dp.getObjects(processing, pending)
	return objectsToDebts(objects)
}
