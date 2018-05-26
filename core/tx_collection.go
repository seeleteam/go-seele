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
	nonceToTxMap map[uint64]*pooledTx // nonce -> transaction
}

func newTxCollection() *txCollection {
	return &txCollection{
		nonceToTxMap: make(map[uint64]*pooledTx),
	}
}

func (collection *txCollection) add(tx *pooledTx) {
	collection.nonceToTxMap[tx.transaction.Data.AccountNonce] = tx
}

func (collection *txCollection) getTxs(status byte) []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(collection.nonceToTxMap))

	for _, tx := range collection.nonceToTxMap {
		if tx.txStatus&status != 0 {
			txs = append(txs, tx.transaction)
		}
	}

	return txs
}

func (collection *txCollection) findTx(nonce uint64, status byte) *types.Transaction {
	if collection.nonceToTxMap[nonce] != nil && collection.nonceToTxMap[nonce].txStatus&status != 0 {
		return collection.nonceToTxMap[nonce].transaction
	}
	return nil
}

func (collection *txCollection) remove(nonce uint64) {
	delete(collection.nonceToTxMap, nonce)
}

func (collection *txCollection) count(status byte) int {
	var count int
	for _, tx := range collection.nonceToTxMap {
		if tx.txStatus&status != 0 {
			count++
		}
	}
	return count
}

func (collection *txCollection) getTxsOrderByNonceAsc(status byte) []*types.Transaction {
	txs := collection.getTxs(status)

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Data.AccountNonce < txs[j].Data.AccountNonce
	})

	return txs
}
