/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"sort"

	"github.com/seeleteam/go-seele/core/types"
)

type txCollection struct {
	nonceToTxMap map[uint64]*poolTransaction // nonce -> transaction
}

func newTxCollection() *txCollection {
	return &txCollection{
		nonceToTxMap: make(map[uint64]*poolTransaction),
	}
}

func (collection *txCollection) add(tx *poolTransaction) {
	collection.nonceToTxMap[tx.transaction.Data.AccountNonce] = tx
}

func (collection *txCollection) getPendingTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(collection.nonceToTxMap))

	for _, tx := range collection.nonceToTxMap {
		if tx.txStatus == PENDING {
			txs = append(txs, tx.transaction)
		}
	}

	return txs
}

func (collection *txCollection) getTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(collection.nonceToTxMap))

	for _, tx := range collection.nonceToTxMap {
		txs = append(txs, tx.transaction)
	}

	return txs
}

func (collection *txCollection) findPendingTx(nonce uint64) *types.Transaction {
	if collection.nonceToTxMap[nonce] != nil && collection.nonceToTxMap[nonce].txStatus == PENDING {
		return collection.nonceToTxMap[nonce].transaction
	}

	return nil
}

func (collection *txCollection) findTx(nonce uint64) *types.Transaction {
	if collection.nonceToTxMap[nonce] != nil {
		return collection.nonceToTxMap[nonce].transaction
	}
	return nil
}

func (collection *txCollection) remove(nonce uint64) {
	delete(collection.nonceToTxMap, nonce)
}

func (collection *txCollection) countPendingTxs() int {
	var count int
	for _, tx := range collection.nonceToTxMap {
		if tx.txStatus == PENDING {
			count++
		}
	}
	return count
}

func (collection *txCollection) count() int {
	return len(collection.nonceToTxMap)
}

func (collection *txCollection) getPendingTxsOrderByNonceAsc() []*types.Transaction {
	txs := collection.getPendingTxs()

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Data.AccountNonce < txs[j].Data.AccountNonce
	})

	return txs
}

func (collection *txCollection) getTxsOrderByNonceAsc() []*types.Transaction {
	txs := collection.getTxs()

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Data.AccountNonce < txs[j].Data.AccountNonce
	})

	return txs
}
